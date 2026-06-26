package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/config"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/model"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/pkg/cache"
	secretcrypto "github.com/Sakuya1998/ops-platform/pkg/crypto"
)

type OIDCDiscovery struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	JWKSUri               string   `json:"jwks_uri"`
	UserinfoEndpoint      string   `json:"userinfo_endpoint"`
	ScopesSupported       []string `json:"scopes_supported"`
}

type OIDCTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type OIDCUserInfo struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	PreferredName string `json:"preferred_username"`
	Email         string `json:"email"`
	EmailVerified *bool  `json:"email_verified"`
}

type OIDCState struct {
	State     string    `json:"state"`
	Nonce     string    `json:"nonce"`
	CreatedAt time.Time `json:"created_at"`
}

type OIDCLoginCodeResult struct {
	Code     string `json:"code"`
	Redirect string `json:"redirect"`
}

type OIDCService struct {
	oidcCfg      *config.OIDCConfig
	discovery    *OIDCDiscovery
	httpClient   *http.Client
	authService  *AuthService
	userRepo     *repository.UserRepository
	providerRepo *repository.ProviderRepository
	secretBox    *secretcrypto.SecretBox
	stateCache   *cache.Cache
	jwksCache    *JWKS
	jwksCacheAt  time.Time
}

func NewOIDCService(cfg *config.OIDCConfig, authSvc *AuthService,
	userRepo *repository.UserRepository, providerRepo *repository.ProviderRepository, secretBox *secretcrypto.SecretBox) *OIDCService {
	return NewOIDCServiceWithStateCache(cfg, authSvc, userRepo, providerRepo, secretBox, nil)
}

func NewOIDCServiceWithStateCache(cfg *config.OIDCConfig, authSvc *AuthService,
	userRepo *repository.UserRepository, providerRepo *repository.ProviderRepository, secretBox *secretcrypto.SecretBox, stateCache *cache.Cache) *OIDCService {
	if cfg == nil {
		cfg = &config.OIDCConfig{}
	}
	if secretBox == nil && authSvc != nil {
		secretBox = authSvc.secretBox
	}
	if stateCache == nil {
		stateCache = cache.New(cache.Options{DefaultTTL: 10 * time.Minute, MaxEntries: 10000})
	}
	svc := &OIDCService{
		oidcCfg:      cfg,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		authService:  authSvc,
		userRepo:     userRepo,
		providerRepo: providerRepo,
		secretBox:    secretBox,
		stateCache:   stateCache,
	}
	if cfg.Enabled && cfg.Issuer != "" {
		if err := svc.discover(context.Background()); err != nil {
			log.Printf("[OIDC] Warning: failed to discover provider %s: %v", cfg.ProviderName, err)
		}
	}
	return svc
}

func (s *OIDCService) discover(ctx context.Context) error {
	wellKnown := strings.TrimSuffix(s.oidcCfg.Issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, "GET", wellKnown, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch discovery: %w", err)
	}
	defer resp.Body.Close()
	var disc OIDCDiscovery
	if err := json.NewDecoder(resp.Body).Decode(&disc); err != nil {
		return fmt.Errorf("parse discovery: %w", err)
	}
	s.discovery = &disc
	log.Printf("[OIDC] Discovered provider: %s (issuer: %s)", s.oidcCfg.ProviderName, disc.Issuer)
	return nil
}

func (s *OIDCService) loadDefaultProviderConfig(ctx context.Context) {
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if s.providerRepo == nil {
		return
	}
	provider, err := s.providerRepo.GetByOrgAndProvider(orgID, "oidc")
	if err != nil || provider.Config == "" {
		return
	}
	var cfg config.OIDCConfig
	if err := json.Unmarshal([]byte(provider.Config), &cfg); err != nil {
		log.Printf("[OIDC] Invalid provider config: %v", err)
		return
	}
	cfg.Enabled = provider.IsEnabled
	if s.secretBox != nil {
		decrypted, err := s.secretBox.DecryptString(cfg.ClientSecret)
		if err != nil {
			log.Printf("[OIDC] Invalid encrypted client secret: %v", err)
			return
		}
		cfg.ClientSecret = decrypted
	}
	if cfg.DefaultOrgCode == "" {
		cfg.DefaultOrgCode = orgID.String()
	}
	if s.oidcCfg == nil || s.oidcCfg.Issuer != cfg.Issuer || s.oidcCfg.ClientID != cfg.ClientID {
		s.discovery = nil
		s.jwksCache = nil
	}
	s.oidcCfg = &cfg
	if cfg.Enabled && cfg.Issuer != "" && s.discovery == nil {
		if err := s.discover(ctx); err != nil {
			log.Printf("[OIDC] Warning: failed to discover provider %s: %v", cfg.ProviderName, err)
		}
	}
}

func (s *OIDCService) LoginURL() (string, string, error) {
	s.loadDefaultProviderConfig(context.Background())
	if s.discovery == nil {
		return "", "", fmt.Errorf("OIDC discovery not loaded for %s", s.oidcCfg.ProviderName)
	}
	state, err := generateRandomString(32)
	if err != nil {
		return "", "", fmt.Errorf("generate state: %w", err)
	}
	nonce, err := generateRandomString(16)
	if err != nil {
		return "", "", fmt.Errorf("generate nonce: %w", err)
	}
	if err := s.saveState(context.Background(), &OIDCState{State: state, Nonce: nonce, CreatedAt: time.Now()}); err != nil {
		return "", "", fmt.Errorf("save state: %w", err)
	}
	scopes := "openid profile email"
	if len(s.oidcCfg.Scopes) > 0 {
		scopes = strings.Join(s.oidcCfg.Scopes, " ")
	}
	authURL := fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s&nonce=%s",
		s.discovery.AuthorizationEndpoint,
		url.QueryEscape(s.oidcCfg.ClientID),
		url.QueryEscape(s.oidcCfg.RedirectURI),
		url.QueryEscape(scopes), url.QueryEscape(state), url.QueryEscape(nonce))
	return authURL, state, nil
}

func (s *OIDCService) HandleCallback(ctx context.Context, code, state string) (*LoginResult, error) {
	s.loadDefaultProviderConfig(ctx)
	stored, err := s.consumeState(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("invalid state parameter")
	}
	if time.Since(stored.CreatedAt) > 10*time.Minute {
		return nil, fmt.Errorf("state expired")
	}

	tokenResp, err := s.exchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	userInfo, err := s.validateIDToken(ctx, tokenResp.IDToken, stored.Nonce)
	if err != nil || tokenResp.IDToken == "" {
		userInfo, err = s.fetchUserInfo(ctx, tokenResp.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("fetch userinfo: %w", err)
		}
	}

	orgID := resolveOrgIDFromCode(s.oidcCfg.DefaultOrgCode)
	user, err := s.resolveOIDCUser(orgID, userInfo)
	if err != nil {
		return nil, err
	}

	roles := s.authService.loadRoleCodes(ctx, user.ID.String())
	session, err := s.authService.createSession(user, LoginContext{})
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	jti := uuid.New().String()
	accessToken, err := s.authService.jwtManager.GenerateWithSession(
		user.ID.String(), user.OrgID.String(), session.ID.String(), jti)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	refreshToken, err := s.authService.generateRefreshToken(user.ID, session.ID, jti)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &LoginResult{
		AccessToken: accessToken, RefreshToken: refreshToken, ExpiresIn: int64(s.authService.cfg.ExpireHour) * 3600,
		SessionID: session.ID.String(), JTI: jti,
		UserID: user.ID.String(), OrgID: user.OrgID.String(),
		Username: user.Username, DisplayName: user.DisplayName,
		Email: user.Email, Roles: roles,
		MustChangePassword: user.MustChangePassword,
		MFAEnabled:         user.MFAEnabled,
	}, nil
}

func (s *OIDCService) HandleCallbackCode(ctx context.Context, code, state string) (*OIDCLoginCodeResult, error) {
	result, err := s.HandleCallback(ctx, code, state)
	if err != nil {
		return nil, err
	}
	loginCode, err := s.IssueLoginCode(ctx, result)
	if err != nil {
		return nil, err
	}
	redirect := "/login/callback?code=" + url.QueryEscape(loginCode)
	return &OIDCLoginCodeResult{Code: loginCode, Redirect: redirect}, nil
}

func (s *OIDCService) saveState(ctx context.Context, state *OIDCState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return s.stateCache.Set(ctx, oidcStateKey(state.State), data, 10*time.Minute)
}

func (s *OIDCService) consumeState(ctx context.Context, state string) (*OIDCState, error) {
	data, err := s.stateCache.Get(ctx, oidcStateKey(state))
	if err != nil {
		return nil, err
	}
	_ = s.stateCache.Delete(ctx, oidcStateKey(state))
	var stored OIDCState
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}
	return &stored, nil
}

func oidcStateKey(state string) string {
	return "oidc:state:" + state
}

func (s *OIDCService) IssueLoginCode(ctx context.Context, result *LoginResult) (string, error) {
	code, err := generateRandomString(32)
	if err != nil {
		return "", fmt.Errorf("generate login code: %w", err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal login result: %w", err)
	}
	if err := s.stateCache.Set(ctx, oidcLoginCodeKey(code), data, 2*time.Minute); err != nil {
		return "", fmt.Errorf("save login code: %w", err)
	}
	return code, nil
}

func (s *OIDCService) ExchangeLoginCode(ctx context.Context, code string) (*LoginResult, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("code is required")
	}
	key := oidcLoginCodeKey(code)
	data, err := s.stateCache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired login code")
	}
	_ = s.stateCache.Delete(ctx, key)
	var result LoginResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse login code: %w", err)
	}
	return &result, nil
}

func oidcLoginCodeKey(code string) string {
	return "oidc:login-code:" + code
}

func (s *OIDCService) resolveOIDCUser(orgID uuid.UUID, userInfo *OIDCUserInfo) (*model.User, error) {
	if userInfo.Sub == "" {
		return nil, fmt.Errorf("OIDC subject is required")
	}
	if userInfo.PreferredName == "" {
		userInfo.PreferredName = userInfo.Email
	}
	if userInfo.PreferredName == "" {
		userInfo.PreferredName = userInfo.Sub
	}

	if cred, err := s.userRepo.GetCredential(orgID, "oidc", userInfo.Sub); err == nil {
		user, err := s.userRepo.GetByID(cred.UserID)
		if err != nil {
			return nil, fmt.Errorf("bound user not found")
		}
		_ = s.upsertOIDCCredential(user, userInfo)
		return user, nil
	}

	user, err := s.userRepo.GetByUsername(orgID, userInfo.PreferredName)
	if err != nil {
		if !s.oidcCfg.AutoProvision {
			return nil, fmt.Errorf("user %s not found and auto-provision disabled", userInfo.PreferredName)
		}
		displayName := userInfo.Name
		if displayName == "" {
			displayName = userInfo.PreferredName
		}
		user = &model.User{
			OrgID: orgID, Username: userInfo.PreferredName,
			Email: userInfo.Email, DisplayName: displayName,
			Status: "active", Source: "oidc",
		}
		if err := s.userRepo.Create(user); err != nil {
			return nil, fmt.Errorf("auto-create user: %w", err)
		}
		log.Printf("[OIDC] Auto-created user: %s", userInfo.PreferredName)
	} else {
		user.Email = userInfo.Email
		if userInfo.Name != "" {
			user.DisplayName = userInfo.Name
		}
		user.Source = "oidc"
		_ = s.userRepo.Update(user)
	}

	if err := s.upsertOIDCCredential(user, userInfo); err != nil {
		return nil, fmt.Errorf("save OIDC credential: %w", err)
	}
	return user, nil
}

func (s *OIDCService) upsertOIDCCredential(user *model.User, userInfo *OIDCUserInfo) error {
	raw, _ := json.Marshal(userInfo)
	return s.userRepo.UpsertCredential(&model.UserCredential{
		UserID:         user.ID,
		OrgID:          user.OrgID,
		Provider:       "oidc",
		ProviderUserID: userInfo.Sub,
		Username:       userInfo.PreferredName,
		Email:          userInfo.Email,
		RawProfile:     string(raw),
	})
}

func (s *OIDCService) exchangeCode(ctx context.Context, code string) (*OIDCTokenResponse, error) {
	data := url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {s.oidcCfg.RedirectURI}, "client_id": {s.oidcCfg.ClientID},
	}
	if s.oidcCfg.ClientSecret != "" {
		data.Set("client_secret", s.oidcCfg.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}

	var tokenResp OIDCTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	return &tokenResp, nil
}

func (s *OIDCService) fetchUserInfo(ctx context.Context, accessToken string) (*OIDCUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.discovery.UserinfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo endpoint: %w", err)
	}
	defer resp.Body.Close()
	var ui OIDCUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&ui); err != nil {
		return nil, fmt.Errorf("parse userinfo: %w", err)
	}
	return &ui, nil
}

func (s *OIDCService) IsEnabled() bool {
	s.loadDefaultProviderConfig(context.Background())
	return s.oidcCfg != nil && s.oidcCfg.Enabled && s.discovery != nil
}

func (s *OIDCService) ProviderName() string {
	if s.oidcCfg == nil {
		return ""
	}
	return s.oidcCfg.ProviderName
}

type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	N   string `json:"n,omitempty"`
	E   string `json:"e,omitempty"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type idTokenClaims struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	PreferredName string `json:"preferred_username"`
	Email         string `json:"email"`
	EmailVerified *bool  `json:"email_verified"`
	Nonce         string `json:"nonce"`
	jwt.RegisteredClaims
}

func (s *OIDCService) refreshJWKS(ctx context.Context) error {
	if s.discovery == nil || s.discovery.JWKSUri == "" {
		return fmt.Errorf("no JWKS URI")
	}
	if s.jwksCache != nil && time.Since(s.jwksCacheAt) < 10*time.Minute {
		return nil
	}
	req, _ := http.NewRequestWithContext(ctx, "GET", s.discovery.JWKSUri, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("parse jwks: %w", err)
	}
	s.jwksCache = &jwks
	s.jwksCacheAt = time.Now()
	log.Printf("[OIDC] Cached %d JWKS keys from %s", len(jwks.Keys), s.oidcCfg.ProviderName)
	return nil
}

func (s *OIDCService) validateIDToken(ctx context.Context, idToken, expectedNonce string) (*OIDCUserInfo, error) {
	header, err := parseJWTHeader(idToken)
	if err != nil {
		return nil, fmt.Errorf("parse header: %w", err)
	}
	kid, _ := header["kid"].(string)
	if kid == "" {
		return nil, fmt.Errorf("no kid in JWT header")
	}
	alg, _ := header["alg"].(string)
	if alg != "" && alg != jwt.SigningMethodRS256.Alg() {
		return nil, fmt.Errorf("unsupported signing algorithm: %s", alg)
	}

	if err := s.refreshJWKS(ctx); err != nil {
		return nil, err
	}
	if s.jwksCache == nil {
		return nil, fmt.Errorf("no JWKS available")
	}

	var matchingKey *JWK
	for i := range s.jwksCache.Keys {
		if s.jwksCache.Keys[i].Kid == kid {
			matchingKey = &s.jwksCache.Keys[i]
			break
		}
	}
	if matchingKey == nil {
		return nil, fmt.Errorf("key not found: %s", kid)
	}

	pubKey, err := rsaPublicKeyFromJWK(matchingKey)
	if err != nil {
		return nil, fmt.Errorf("public key: %w", err)
	}

	expectedIssuer := strings.TrimSuffix(s.oidcCfg.Issuer, "/")
	if s.discovery != nil && s.discovery.Issuer != "" {
		expectedIssuer = strings.TrimSuffix(s.discovery.Issuer, "/")
	}
	token, err := jwt.ParseWithClaims(idToken, &idTokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
		}
		return pubKey, nil
	}, jwt.WithIssuer(expectedIssuer), jwt.WithAudience(s.oidcCfg.ClientID), jwt.WithExpirationRequired())
	if err != nil {
		return nil, fmt.Errorf("verify signature: %w", err)
	}

	claims := token.Claims.(*idTokenClaims)
	if !token.Valid {
		return nil, fmt.Errorf("invalid id token")
	}
	if claims.Sub == "" {
		return nil, fmt.Errorf("subject is required")
	}
	if expectedNonce == "" {
		return nil, fmt.Errorf("expected nonce is required")
	}
	if claims.Nonce != expectedNonce {
		return nil, fmt.Errorf("nonce mismatch")
	}
	return &OIDCUserInfo{
		Sub: claims.Sub, Name: claims.Name,
		PreferredName: claims.PreferredName,
		Email:         claims.Email, EmailVerified: claims.EmailVerified,
	}, nil
}

func rsaPublicKeyFromJWK(jwk *JWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("decode n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("decode e: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	eBytesPadded := make([]byte, 8)
	copy(eBytesPadded[8-len(eBytes):], eBytes)
	e := int(binary.BigEndian.Uint64(eBytesPadded))
	return &rsa.PublicKey{N: n, E: e}, nil
}

func parseJWTHeader(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var header map[string]interface{}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, err
	}
	return header, nil
}
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}
