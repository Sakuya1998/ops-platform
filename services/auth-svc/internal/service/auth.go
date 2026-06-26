package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/model"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/repository"
	sharedconfig "github.com/Sakuya1998/ops-platform/pkg/config"
	secretcrypto "github.com/Sakuya1998/ops-platform/pkg/crypto"
	"github.com/Sakuya1998/ops-platform/pkg/jwt"
	"github.com/Sakuya1998/ops-platform/pkg/kafka"
	"golang.org/x/crypto/bcrypt"
)

const (
	maxFailedLoginAttempts = 5
	loginLockDuration      = 30 * time.Minute
)

type AuthService struct {
	userRepo     *repository.UserRepository
	providerRepo *repository.ProviderRepository
	orgRepo      *repository.OrganizationRepository
	iamClient    *IAMClient
	ldapSvc      *LdapService
	jwtManager   *jwt.Manager
	secretBox    *secretcrypto.SecretBox
	kafkaProd    eventPublisher
	cfg          sharedconfig.JWTConfig
	secretPepper []byte
	loginLimiter *LoginLimiter
}

func NewAuthService(
	userRepo *repository.UserRepository,
	providerRepo *repository.ProviderRepository,
	orgRepo *repository.OrganizationRepository,
	iamClient *IAMClient,
	jwtCfg sharedconfig.JWTConfig,
	kafkaProd *kafka.Producer,
	ldapSvc *LdapService,
	secretBox *secretcrypto.SecretBox,
) *AuthService {
	if secretBox == nil {
		secretBox = secretcrypto.NewSecretBox(jwtCfg.Secret)
	}
	svc := &AuthService{
		userRepo:     userRepo,
		providerRepo: providerRepo,
		orgRepo:      orgRepo,
		iamClient:    iamClient,
		ldapSvc:      ldapSvc,
		jwtManager:   jwt.NewManager(jwtCfg.Secret, jwtCfg.ExpireHour, jwtCfg.Issuer),
		secretBox:    secretBox,
		cfg:          jwtCfg,
		secretPepper: secretBox.DeriveKey("auth-svc:mfa-recovery-code:v1"),
	}
	if kafkaProd != nil {
		svc.kafkaProd = kafkaProd
	}
	return svc
}

func (s *AuthService) WithLoginLimiter(limiter *LoginLimiter) *AuthService {
	s.loginLimiter = limiter
	return s
}

type LoginResult struct {
	AccessToken        string
	RefreshToken       string
	ExpiresIn          int64
	SessionID          string
	JTI                string
	UserID             string
	OrgID              string
	Username           string
	DisplayName        string
	Email              string
	Roles              []string
	MustChangePassword bool
	MFAEnabled         bool
}

type LoginContext struct {
	IP         string
	UserAgent  string
	DeviceName string
	MFACode    string
	RequestID  string
}

func (s *AuthService) Login(ctx context.Context, username, password, orgCode, provider string, loginCtx LoginContext) (*LoginResult, error) {
	orgID := s.resolveOrgID(orgCode)
	var user *model.User
	attempt := LoginAttemptKey{
		OrgID:    orgID.String(),
		Provider: normalizeProvider(provider),
		Username: username,
		IP:       loginCtx.IP,
	}
	if err := s.loginLimiter.Allow(ctx, attempt); err != nil {
		s.publishAuditEvent(ctx, AuditEvent{
			EventType: "user.login_limited", OrgID: orgID.String(), Username: username,
			Action: "login", ResourceType: "auth", Detail: err.Error(),
			IP: loginCtx.IP, UserAgent: loginCtx.UserAgent, ReasonCode: "login_rate_limited", RequestID: loginCtx.RequestID,
		})
		return nil, err
	}

	switch provider {
	case "local", "":
		var err error
		user, err = s.userRepo.GetByUsername(orgID, username)
		if err != nil {
			s.loginLimiter.RecordFailure(ctx, attempt)
			s.publishLoginFailure(ctx, "", orgID.String(), username, "local", "User not found", "user_not_found", loginCtx)
			return nil, fmt.Errorf("invalid credentials")
		}
		if s.isUserLocked(user) {
			s.loginLimiter.RecordFailure(ctx, attempt)
			s.publishLoginFailure(ctx, user.ID.String(), user.OrgID.String(), username, "local", "User locked", "user_locked", loginCtx)
			return nil, fmt.Errorf("user is locked")
		}
		if user.Status != "active" {
			s.loginLimiter.RecordFailure(ctx, attempt)
			s.publishLoginFailure(ctx, user.ID.String(), user.OrgID.String(), username, "local", "User not active", "user_not_active", loginCtx)
			return nil, fmt.Errorf("user is %s", user.Status)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			_ = s.userRepo.RecordFailedLogin(user.ID, maxFailedLoginAttempts, loginLockDuration)
			s.loginLimiter.RecordFailure(ctx, attempt)
			s.publishLoginFailure(ctx, user.ID.String(), user.OrgID.String(), username, "local", "Wrong password", "invalid_credentials", loginCtx)
			return nil, fmt.Errorf("invalid credentials")
		}
		_ = s.markPasswordExpiredIfNeeded(user)
	case "ldap":
		if s.ldapSvc == nil {
			return nil, fmt.Errorf("LDAP authentication is disabled")
		}
		var err error
		ldapCfg := s.ldapConfigForOrg(orgID)
		user, err = s.ldapSvc.AuthenticateWithConfig(ctx, username, password, ldapCfg)
		if err != nil {
			s.loginLimiter.RecordFailure(ctx, attempt)
			s.publishLoginFailure(ctx, "", orgID.String(), username, "ldap", err.Error(), "ldap_auth_failed", loginCtx)
			return nil, fmt.Errorf("invalid credentials")
		}
	default:
		return nil, fmt.Errorf("unsupported auth provider: %s", provider)
	}

	if user.MFAEnabled {
		ok, err := s.verifyUserMFA(user, loginCtx.MFACode)
		if err != nil {
			return nil, err
		}
		if !ok {
			s.loginLimiter.RecordFailure(ctx, attempt)
			s.publishLoginFailure(ctx, user.ID.String(), user.OrgID.String(), username, attempt.Provider, "Invalid MFA code", "invalid_mfa_code", loginCtx)
			return nil, fmt.Errorf("mfa code required or invalid")
		}
	}

	roles := s.loadRoleCodes(ctx, user.ID.String())
	_ = s.userRepo.ResetLoginFailures(user.ID)

	session, err := s.createSession(user, loginCtx)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	jti := uuid.New().String()
	accessToken, err := s.jwtManager.GenerateWithSession(user.ID.String(), user.OrgID.String(), session.ID.String(), jti)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user.ID, session.ID, jti)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	_ = s.userRepo.UpdateLastLogin(user.ID)
	s.loginLimiter.RecordSuccess(ctx, attempt)

	s.publishAuditEvent(ctx, AuditEvent{
		EventType: "user.login", UserID: user.ID.String(), OrgID: user.OrgID.String(), Username: username,
		Action: "login", ResourceType: "auth", Detail: "Login successful",
		IP: loginCtx.IP, UserAgent: loginCtx.UserAgent, SessionID: session.ID.String(), ReasonCode: "success", RequestID: loginCtx.RequestID,
	})

	return &LoginResult{
		AccessToken:        accessToken,
		RefreshToken:       refreshToken,
		ExpiresIn:          int64(s.cfg.ExpireHour) * 3600,
		SessionID:          session.ID.String(),
		JTI:                jti,
		UserID:             user.ID.String(),
		OrgID:              user.OrgID.String(),
		Username:           user.Username,
		DisplayName:        user.DisplayName,
		Email:              user.Email,
		Roles:              roles,
		MustChangePassword: user.MustChangePassword,
		MFAEnabled:         user.MFAEnabled,
	}, nil
}

func (s *AuthService) isUserLocked(user *model.User) bool {
	if user.LockedUntil != nil {
		if time.Now().Before(*user.LockedUntil) {
			return true
		}
		_ = s.userRepo.ResetLoginFailures(user.ID)
		user.Status = "active"
		user.LockedUntil = nil
		user.FailedLoginAttempts = 0
		return false
	}
	return user.Status == "locked"
}

func (s *AuthService) loadRoleCodes(ctx context.Context, userID string) []string {
	if s.iamClient == nil {
		return []string{}
	}
	roles, err := s.iamClient.GetUserRoleCodes(ctx, userID)
	if err != nil {
		return []string{}
	}
	return roles
}

func normalizeProvider(provider string) string {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return "local"
	}
	return provider
}
