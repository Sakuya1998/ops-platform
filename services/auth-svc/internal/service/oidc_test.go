package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	authconfig "github.com/ops-platform/auth-svc/internal/config"
	"github.com/ops-platform/auth-svc/internal/repository"
	"github.com/ops-platform/pkg/cache"
)

func TestOIDCStateCacheConsumeOnce(t *testing.T) {
	svc := NewOIDCServiceWithStateCache(&authconfig.OIDCConfig{}, nil, nil, nil, nil, nil)
	ctx := context.Background()
	state := &OIDCState{State: "state-1", Nonce: "nonce-1", CreatedAt: time.Now()}
	if err := svc.saveState(ctx, state); err != nil {
		t.Fatalf("saveState: %v", err)
	}
	got, err := svc.consumeState(ctx, "state-1")
	if err != nil {
		t.Fatalf("consumeState: %v", err)
	}
	if got.Nonce != "nonce-1" {
		t.Fatalf("expected nonce-1, got %s", got.Nonce)
	}
	if _, err := svc.consumeState(ctx, "state-1"); err == nil {
		t.Fatal("expected second consume to fail")
	}
}

func TestOIDCLoginCodeExchangeConsumesOnce(t *testing.T) {
	svc := NewOIDCServiceWithStateCache(&authconfig.OIDCConfig{}, nil, nil, nil, nil, cache.New(cache.Options{
		DefaultTTL: time.Minute,
		MaxEntries: 100,
	}))
	ctx := context.Background()
	want := &LoginResult{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		ExpiresIn:    7200,
		SessionID:    "session-1",
		JTI:          "jti-1",
		UserID:       "user-1",
		OrgID:        "org-1",
		Username:     "oidc-user",
		DisplayName:  "OIDC User",
		Email:        "oidc@example.com",
		Roles:        []string{"admin"},
		MFAEnabled:   true,
	}

	code, err := svc.IssueLoginCode(ctx, want)
	if err != nil {
		t.Fatalf("IssueLoginCode: %v", err)
	}
	if len(code) < 32 {
		t.Fatalf("expected opaque high entropy code, got %q", code)
	}

	got, err := svc.ExchangeLoginCode(ctx, code)
	if err != nil {
		t.Fatalf("ExchangeLoginCode: %v", err)
	}
	if got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken || got.SessionID != want.SessionID {
		t.Fatalf("unexpected login result: %+v", got)
	}
	if _, err := svc.ExchangeLoginCode(ctx, code); err == nil {
		t.Fatal("expected second exchange to fail")
	}
}

func TestResolveOIDCUserUsesBoundCredential(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	credID := uuid.MustParse("00000000-0000-0000-0000-000000000030")
	repo := repository.NewUserRepository(db)
	svc := &OIDCService{
		oidcCfg:  &authconfig.OIDCConfig{AutoProvision: true},
		userRepo: repo,
	}

	credColumns := []string{"id", "user_id", "org_id", "provider", "provider_user_id", "username", "email", "raw_profile", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT \* FROM "user_credentials"`).
		WithArgs(orgID, "oidc", "subject-1", 1).
		WillReturnRows(sqlmock.NewRows(credColumns).
			AddRow(credID, userID, orgID, "oidc", "subject-1", "old-name", "old@example.com", "{}", time.Now(), time.Now()))

	userColumns := []string{"id", "org_id", "username", "password_hash", "display_name", "email", "status", "source", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "bound-user", "", "Bound User", "bound@example.com", "active", "oidc", time.Now(), time.Now()))

	mock.ExpectQuery(`SELECT \* FROM "user_credentials"`).
		WithArgs(orgID, "oidc", "subject-1", 1).
		WillReturnRows(sqlmock.NewRows(credColumns).
			AddRow(credID, userID, orgID, "oidc", "subject-1", "old-name", "old@example.com", "{}", time.Now(), time.Now()))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "user_credentials"`).
		WithArgs(userID, orgID, "oidc", "subject-1", "new-name", "new@example.com", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), credID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	user, err := svc.resolveOIDCUser(orgID, &OIDCUserInfo{
		Sub:           "subject-1",
		PreferredName: "new-name",
		Name:          "New Name",
		Email:         "new@example.com",
	})
	if err != nil {
		t.Fatalf("resolveOIDCUser: %v", err)
	}
	if user.ID != userID {
		t.Fatalf("expected bound user %s, got %s", userID, user.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestValidateIDTokenRequiresExpectedOIDCClaims(t *testing.T) {
	ctx := context.Background()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	svc := &OIDCService{
		oidcCfg: &authconfig.OIDCConfig{
			Issuer:   "https://idp.example.com/realms/ops",
			ClientID: "ops-web",
		},
		discovery: &OIDCDiscovery{Issuer: "https://idp.example.com/realms/ops", JWKSUri: "http://unused.local/jwks"},
		jwksCache: &JWKS{Keys: []JWK{{
			Kid: "kid-1",
			Kty: "RSA",
			Alg: "RS256",
			N:   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
		}}},
		jwksCacheAt: time.Now(),
	}

	validToken := signOIDCIDToken(t, key, "kid-1", idTokenClaims{
		Sub:           "subject-1",
		Name:          "OIDC User",
		PreferredName: "oidc-user",
		Email:         "oidc@example.com",
		Nonce:         "nonce-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://idp.example.com/realms/ops",
			Audience:  jwt.ClaimStrings{"ops-web"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   "subject-1",
		},
	})
	userInfo, err := svc.validateIDToken(ctx, validToken, "nonce-1")
	if err != nil {
		t.Fatalf("validateIDToken valid token: %v", err)
	}
	if userInfo.Sub != "subject-1" || userInfo.Email != "oidc@example.com" {
		t.Fatalf("unexpected user info: %+v", userInfo)
	}

	tests := []struct {
		name   string
		claims idTokenClaims
		nonce  string
	}{
		{
			name: "issuer mismatch",
			claims: idTokenClaims{Sub: "subject-1", Nonce: "nonce-1", RegisteredClaims: jwt.RegisteredClaims{
				Issuer: "https://evil.example.com", Audience: jwt.ClaimStrings{"ops-web"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			}},
			nonce: "nonce-1",
		},
		{
			name: "audience mismatch",
			claims: idTokenClaims{Sub: "subject-1", Nonce: "nonce-1", RegisteredClaims: jwt.RegisteredClaims{
				Issuer: "https://idp.example.com/realms/ops", Audience: jwt.ClaimStrings{"other-client"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			}},
			nonce: "nonce-1",
		},
		{
			name: "nonce mismatch",
			claims: idTokenClaims{Sub: "subject-1", Nonce: "other-nonce", RegisteredClaims: jwt.RegisteredClaims{
				Issuer: "https://idp.example.com/realms/ops", Audience: jwt.ClaimStrings{"ops-web"},
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			}},
			nonce: "nonce-1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := signOIDCIDToken(t, key, "kid-1", tt.claims)
			if _, err := svc.validateIDToken(ctx, token, tt.nonce); err == nil {
				t.Fatal("expected token validation to fail")
			}
		})
	}
}

func signOIDCIDToken(t *testing.T, key *rsa.PrivateKey, kid string, claims idTokenClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}
