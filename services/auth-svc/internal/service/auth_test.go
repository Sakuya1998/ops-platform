package service

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	authconfig "github.com/Sakuya1998/ops-platform/services/auth-svc/internal/config"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/pkg/cache"
	sharedcfg "github.com/Sakuya1998/ops-platform/pkg/config"
	secretcrypto "github.com/Sakuya1998/ops-platform/pkg/crypto"
	"github.com/Sakuya1998/ops-platform/pkg/kafka"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestLoginLocalSuccess(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	hash, err := bcrypt.GenerateFromPassword([]byte("admin@2026"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	userRows := sqlmock.NewRows([]string{
		"id", "org_id", "username", "password_hash", "display_name", "email", "status", "source", "created_at", "updated_at",
	}).AddRow(userID, orgID, "admin", string(hash), "Admin", "admin@example.com", "active", "local", time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs(orgID, "admin", 1).WillReturnRows(userRows)

	iamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/users/"+userID.String()+"/roles" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    0,
			"message": "success",
			"data": []map[string]string{
				{"id": "role-1", "code": "admin", "name": "Admin"},
			},
		})
	}))
	defer iamServer.Close()

	repo := repository.NewUserRepository(db)
	svc := NewAuthService(repo, repository.NewProviderRepository(db), nil, NewIAMClient(iamServer.URL), sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users" SET "failed_login_attempts"=\$1,"locked_until"=\$2,"status"=\$3,"updated_at"=\$4 WHERE id = \$5`).
		WithArgs(0, nil, "active", sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "sessions"`).
		WithArgs(sqlmock.AnyArg(), userID, orgID, "active", "127.0.0.1", "test-agent", "test-device", sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "refresh_tokens"`).
		WithArgs(sqlmock.AnyArg(), userID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), false, nil, "", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users" SET "last_login_at"=NOW\(\),"updated_at"=\$1 WHERE id = \$2`).
		WithArgs(sqlmock.AnyArg(), userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := svc.Login(context.Background(), "admin", "admin@2026", "default", "local", LoginContext{
		IP:         "127.0.0.1",
		UserAgent:  "test-agent",
		DeviceName: "test-device",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("expected tokens, got %+v", result)
	}
	if result.SessionID == "" || result.JTI == "" {
		t.Fatalf("expected session and jti, got %+v", result)
	}
	if len(result.Roles) != 1 || result.Roles[0] != "admin" {
		t.Fatalf("expected admin role, got %+v", result.Roles)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestNewRefreshTokenUsesHighEntropyRandomValue(t *testing.T) {
	svc := NewAuthService(nil, nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)

	token, record, err := svc.newRefreshToken(uuid.New(), uuid.New(), "jti-1")
	if err != nil {
		t.Fatalf("newRefreshToken: %v", err)
	}
	if len(token) < 43 {
		t.Fatalf("expected at least 256 bits of base64url entropy, got token length %d", len(token))
	}
	if _, err := uuid.Parse(token); err == nil {
		t.Fatalf("refresh token must not use UUID format")
	}
	hash := sha256.Sum256([]byte(token))
	if record.TokenHash != hex.EncodeToString(hash[:]) {
		t.Fatalf("refresh token hash mismatch")
	}
}

func TestSystemConfigRoundTripKeepsDisabledProviderConfig(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	providerRepo := repository.NewProviderRepository(db)
	ldapSvc := NewLdapService(&authconfig.LDAPConfig{
		Port:            389,
		UserFilter:      "(uid=%s)",
		UIDAttr:         "uid",
		DisplayNameAttr: "cn",
		EmailAttr:       "mail",
	}, nil)
	svc := NewAuthService(nil, providerRepo, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, ldapSvc, nil)

	ldapCfg := authconfig.LDAPConfig{
		Enabled:         false,
		Host:            "ldap.example.com",
		Port:            636,
		Security:        "tls",
		BindDN:          "cn=admin,dc=example,dc=com",
		BindPassword:    "secret",
		BaseDN:          "dc=example,dc=com",
		UserFilter:      "(uid=%s)",
		UIDAttr:         "uid",
		DisplayNameAttr: "cn",
		EmailAttr:       "mail",
		AutoProvision:   true,
		SkipVerify:      true,
	}
	oidcCfg := authconfig.OIDCConfig{
		Enabled:       false,
		ProviderName:  "Keycloak",
		Issuer:        "https://idp.example.com/realms/ops",
		ClientID:      "ops-platform",
		ClientSecret:  "oidc-secret",
		RedirectURI:   "http://localhost:3000/api/v1/auth/oidc/callback",
		Scopes:        []string{"openid", "profile", "email"},
		AutoProvision: true,
	}

	mock.ExpectQuery(`SELECT \* FROM "auth_providers"`).
		WithArgs(orgID, "ldap", 1).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "auth_providers"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), false, "LDAP Authentication", orgID, "ldap", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	mock.ExpectQuery(`SELECT \* FROM "auth_providers"`).
		WithArgs(orgID, "oidc", 1).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "auth_providers"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), false, "OIDC Authentication", orgID, "oidc", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := svc.UpdateSystemConfig(orgID.String(), SystemConfig{LDAP: ldapCfg, OIDC: oidcCfg}); err != nil {
		t.Fatalf("UpdateSystemConfig: %v", err)
	}

	ldapData, _ := json.Marshal(func() authconfig.LDAPConfig {
		ldapCfg.DefaultOrgCode = orgID.String()
		_ = svc.encryptLDAPConfig(&ldapCfg)
		return ldapCfg
	}())
	oidcData, _ := json.Marshal(func() authconfig.OIDCConfig {
		oidcCfg.DefaultOrgCode = orgID.String()
		_ = svc.encryptOIDCConfig(&oidcCfg)
		return oidcCfg
	}())

	providerColumns := []string{"id", "org_id", "provider", "name", "config", "is_enabled", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT \* FROM "auth_providers"`).
		WithArgs(orgID, "ldap", 1).
		WillReturnRows(sqlmock.NewRows(providerColumns).
			AddRow(uuid.New(), orgID, "ldap", "LDAP Authentication", string(ldapData), false, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT \* FROM "auth_providers"`).
		WithArgs(orgID, "oidc", 1).
		WillReturnRows(sqlmock.NewRows(providerColumns).
			AddRow(uuid.New(), orgID, "oidc", "OIDC Authentication", string(oidcData), false, time.Now(), time.Now()))

	cfg, err := svc.GetSystemConfig(orgID.String())
	if err != nil {
		t.Fatalf("GetSystemConfig: %v", err)
	}
	if cfg.LDAP.Enabled || cfg.OIDC.Enabled {
		t.Fatalf("expected disabled providers, got ldap=%v oidc=%v", cfg.LDAP.Enabled, cfg.OIDC.Enabled)
	}
	if cfg.LDAP.Host != "ldap.example.com" || cfg.OIDC.ProviderName != "Keycloak" {
		t.Fatalf("expected saved config to be returned, got %+v", cfg)
	}
	if cfg.LDAP.BindPassword != "secret" || cfg.OIDC.ClientSecret != "oidc-secret" {
		t.Fatalf("expected decrypted secrets, got ldap=%q oidc=%q", cfg.LDAP.BindPassword, cfg.OIDC.ClientSecret)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestProviderSecretEncryptionKeepsEmptyValuesEmpty(t *testing.T) {
	svc := NewAuthService(nil, nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)

	ldapCfg := authconfig.LDAPConfig{}
	if err := svc.encryptLDAPConfig(&ldapCfg); err != nil {
		t.Fatalf("encryptLDAPConfig: %v", err)
	}
	if ldapCfg.BindPassword != "" {
		t.Fatalf("expected empty LDAP bind password to stay empty, got %q", ldapCfg.BindPassword)
	}
	if err := svc.decryptLDAPConfig(&ldapCfg); err != nil {
		t.Fatalf("decryptLDAPConfig: %v", err)
	}
	if ldapCfg.BindPassword != "" {
		t.Fatalf("expected empty LDAP bind password after decrypt, got %q", ldapCfg.BindPassword)
	}

	oidcCfg := authconfig.OIDCConfig{}
	if err := svc.encryptOIDCConfig(&oidcCfg); err != nil {
		t.Fatalf("encryptOIDCConfig: %v", err)
	}
	if oidcCfg.ClientSecret != "" {
		t.Fatalf("expected empty OIDC client secret to stay empty, got %q", oidcCfg.ClientSecret)
	}
	if err := svc.decryptOIDCConfig(&oidcCfg); err != nil {
		t.Fatalf("decryptOIDCConfig: %v", err)
	}
	if oidcCfg.ClientSecret != "" {
		t.Fatalf("expected empty OIDC client secret after decrypt, got %q", oidcCfg.ClientSecret)
	}
}

func TestValidatePasswordRejectsWeakPassword(t *testing.T) {
	if err := validatePassword("password"); err == nil {
		t.Fatal("expected weak password to be rejected")
	}
	if err := validatePassword("Strong@2026"); err != nil {
		t.Fatalf("expected strong password to pass: %v", err)
	}
}

func TestVerifyTOTPWithRFCVector(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
	code := generateTOTPCode([]byte("12345678901234567890"), 1)
	if code != "287082" {
		t.Fatalf("expected RFC vector code 287082, got %s", code)
	}
	if !verifyTOTP(secret, code, time.Unix(59, 0)) {
		t.Fatal("expected TOTP code to verify")
	}
}

func TestLoginRequiresMFAWhenEnabled(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	hash, err := bcrypt.GenerateFromPassword([]byte("Strong@2026"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	box := secretcrypto.NewSecretBox("test-secret")
	secret, err := box.EncryptString("JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	userColumns := []string{
		"id", "org_id", "username", "email", "phone", "password_hash", "display_name", "avatar", "status", "source",
		"failed_login_attempts", "locked_until", "password_changed_at", "must_change_password", "mfa_enabled", "mfa_secret", "mfa_confirmed_at", "last_login_at",
		"created_at", "updated_at",
	}
	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs(orgID, "admin", 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", "admin@example.com", "", string(hash), "Admin", "", "active", "local",
				0, nil, time.Now(), false, true, secret, time.Now(), nil, time.Now(), time.Now()))

	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, box)
	_, err = svc.Login(context.Background(), "admin", "Strong@2026", "default", "local", LoginContext{})
	if err == nil || !strings.Contains(err.Error(), "mfa") {
		t.Fatalf("expected mfa error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestRecoveryCodeNormalizeAndHash(t *testing.T) {
	hashA := hashRecoveryCode("ABCD-EFGH")
	hashB := hashRecoveryCode("abcd efgh")
	if hashA == "" || hashA != hashB {
		t.Fatalf("expected stable recovery code hash, got %q and %q", hashA, hashB)
	}
	if normalizeRecoveryCode("bad") != "" {
		t.Fatal("expected invalid recovery code to normalize to empty string")
	}
}

func TestRecoveryCodeHashUsesServiceSecretPepper(t *testing.T) {
	svcA := NewAuthService(nil, nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "jwt-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, secretcrypto.NewSecretBox("pepper-a"))
	svcB := NewAuthService(nil, nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "jwt-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, secretcrypto.NewSecretBox("pepper-b"))

	hashA := svcA.hashRecoveryCode("ABCD-EFGH")
	hashANormalized := svcA.hashRecoveryCode("abcd efgh")
	hashB := svcB.hashRecoveryCode("ABCD-EFGH")

	if hashA == "" || hashA != hashANormalized {
		t.Fatalf("expected stable hash for same service secret, got %q and %q", hashA, hashANormalized)
	}
	if hashA == hashB {
		t.Fatal("expected different service secrets to produce different recovery code hashes")
	}
}

func TestLoginLocalFailureLocksUser(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	hash, err := bcrypt.GenerateFromPassword([]byte("Strong@2026"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	userColumns := []string{
		"id", "org_id", "username", "password_hash", "display_name", "email", "status", "source",
		"failed_login_attempts", "locked_until", "password_changed_at", "must_change_password",
		"mfa_enabled", "mfa_secret", "mfa_confirmed_at", "created_at", "updated_at",
	}
	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs(orgID, "admin", 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", string(hash), "Admin", "admin@example.com", "active", "local", 4, nil, time.Now(), false, false, "", nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", string(hash), "Admin", "admin@example.com", "active", "local", 4, nil, time.Now(), false, false, "", nil, time.Now(), time.Now()))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WithArgs(
		orgID, "admin", "admin@example.com", "", string(hash), "Admin", "", "locked", "local",
		5, sqlmock.AnyArg(), sqlmock.AnyArg(), false, false, "", nil, nil, nil, sqlmock.AnyArg(), sqlmock.AnyArg(), userID,
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)
	_, err = svc.Login(context.Background(), "admin", "wrong-password", "default", "local", LoginContext{})
	if err == nil {
		t.Fatal("expected login to fail")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestLoginBlockedByLimiterBeforeCredentialLookup(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	hash, err := bcrypt.GenerateFromPassword([]byte("Strong@2026"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	userColumns := []string{
		"id", "org_id", "username", "password_hash", "display_name", "email", "status", "source",
		"failed_login_attempts", "locked_until", "password_changed_at", "must_change_password",
		"mfa_enabled", "mfa_secret", "mfa_confirmed_at", "created_at", "updated_at",
	}
	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs(orgID, "admin", 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", string(hash), "Admin", "admin@example.com", "active", "local", 0, nil, time.Now(), false, false, "", nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", string(hash), "Admin", "admin@example.com", "active", "local", 0, nil, time.Now(), false, false, "", nil, time.Now(), time.Now()))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil).WithLoginLimiter(NewLoginLimiter(cache.NewMemoryBackend(time.Minute), LoginLimiterOptions{
		MaxAttempts: 1,
		Window:      time.Minute,
	}))
	ctx := context.Background()
	loginCtx := LoginContext{IP: "127.0.0.1"}
	_, err = svc.Login(ctx, "admin", "wrong-password", "default", "local", loginCtx)
	if err == nil {
		t.Fatal("expected first login to fail")
	}
	_, err = svc.Login(ctx, "admin", "Strong@2026", "default", "local", loginCtx)
	if err == nil || !strings.Contains(err.Error(), "too many login attempts") {
		t.Fatalf("expected limiter error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestLoginFailurePublishesSecurityAuditContext(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	hash, err := bcrypt.GenerateFromPassword([]byte("Strong@2026"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	userColumns := []string{
		"id", "org_id", "username", "password_hash", "display_name", "email", "status", "source",
		"failed_login_attempts", "locked_until", "password_changed_at", "must_change_password",
		"mfa_enabled", "mfa_secret", "mfa_confirmed_at", "created_at", "updated_at",
	}
	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs(orgID, "admin", 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", string(hash), "Admin", "admin@example.com", "active", "local", 0, nil, time.Now(), false, false, "", nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT \* FROM "users"`).WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", string(hash), "Admin", "admin@example.com", "active", "local", 0, nil, time.Now(), false, false, "", nil, time.Now(), time.Now()))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	events := &captureEventPublisher{}
	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)
	svc.kafkaProd = events

	_, err = svc.Login(context.Background(), "admin", "wrong-password", "default", "local", LoginContext{
		IP:        "10.0.0.1",
		UserAgent: "test-agent",
		RequestID: "request-1",
	})
	if err == nil {
		t.Fatal("expected login failure")
	}
	if len(events.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(events.events))
	}
	event := events.events[0]
	if event.EventType != "user.login_failed" || event.ReasonCode != "invalid_credentials" {
		t.Fatalf("unexpected event type/reason: %+v", event)
	}
	if event.IP != "10.0.0.1" || event.UserAgent != "test-agent" || event.RequestID != "request-1" {
		t.Fatalf("missing security context: %+v", event)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

type captureEventPublisher struct {
	events []kafka.Event
}

func (p *captureEventPublisher) PublishEvent(ctx context.Context, event kafka.Event) error {
	p.events = append(p.events, event)
	return nil
}

func TestResetUserPasswordClearsLockAndRevokesSessions(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	oldHash, err := bcrypt.GenerateFromPassword([]byte("Old@2026"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	lockedUntil := time.Now().Add(15 * time.Minute)
	passwordChangedAt := time.Now().Add(-24 * time.Hour)
	createdAt := time.Now().Add(-48 * time.Hour)
	updatedAt := time.Now().Add(-24 * time.Hour)
	userColumns := []string{
		"id", "org_id", "username", "email", "phone", "password_hash", "display_name", "avatar", "status", "source",
		"failed_login_attempts", "locked_until", "password_changed_at", "must_change_password", "mfa_enabled", "mfa_secret", "mfa_confirmed_at", "last_login_at",
		"created_at", "updated_at",
	}
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", "admin@example.com", "", string(oldHash), "Admin", "", "locked", "local",
				5, lockedUntil, passwordChangedAt, false, false, "", nil, nil, createdAt, updatedAt))
	mock.ExpectQuery(`SELECT \* FROM "password_histories"`).
		WithArgs(userID, passwordHistoryLimit).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "password_hash", "created_at"}))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WithArgs(
		orgID, "admin", "admin@example.com", "", sqlmock.AnyArg(), "Admin", "", "active", "local",
		0, nil, sqlmock.AnyArg(), true, false, "", nil, nil, nil, sqlmock.AnyArg(), sqlmock.AnyArg(), userID,
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "password_histories"`).
		WithArgs(sqlmock.AnyArg(), userID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT "id" FROM "password_histories"`).
		WithArgs(userID, passwordHistoryLimit).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "sessions"`).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`UPDATE "refresh_tokens"`).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)
	user, err := svc.ResetUserPassword(userID.String(), "NewStrong@2026", true)
	if err != nil {
		t.Fatalf("ResetUserPassword: %v", err)
	}
	if user.Status != "active" || user.FailedLoginAttempts != 0 || user.LockedUntil != nil || !user.MustChangePassword {
		t.Fatalf("expected reset user security state, got %+v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestResetUserPasswordRejectsRecentPassword(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	currentHash, err := bcrypt.GenerateFromPassword([]byte("Current@2026"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt current: %v", err)
	}
	recentHash, err := bcrypt.GenerateFromPassword([]byte("Recent@2026"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("bcrypt recent: %v", err)
	}
	userColumns := []string{
		"id", "org_id", "username", "email", "phone", "password_hash", "display_name", "avatar", "status", "source",
		"failed_login_attempts", "locked_until", "password_changed_at", "must_change_password", "last_login_at",
		"created_at", "updated_at",
	}
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", "admin@example.com", "", string(currentHash), "Admin", "", "active", "local",
				0, nil, time.Now(), false, nil, time.Now(), time.Now()))
	mock.ExpectQuery(`SELECT \* FROM "password_histories"`).
		WithArgs(userID, passwordHistoryLimit).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "password_hash", "created_at"}).
			AddRow(uuid.New(), userID, string(recentHash), time.Now()))

	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)
	_, err = svc.ResetUserPassword(userID.String(), "Recent@2026", true)
	if err == nil {
		t.Fatal("expected recent password reuse to fail")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestUpdateUserStatusRevokesSessionsWhenDisabled(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	userColumns := []string{
		"id", "org_id", "username", "email", "phone", "password_hash", "display_name", "avatar", "status", "source",
		"failed_login_attempts", "locked_until", "password_changed_at", "must_change_password", "mfa_enabled", "mfa_secret", "mfa_confirmed_at", "last_login_at",
		"created_at", "updated_at",
	}
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", "admin@example.com", "", "", "Admin", "", "active", "local",
				0, nil, time.Now(), false, false, "", nil, nil, time.Now(), time.Now()))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "sessions"`).
		WithArgs(sqlmock.AnyArg(), "user_disabled", "revoked", sqlmock.AnyArg(), userID, "active").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`UPDATE "refresh_tokens"`).
		WithArgs(true, sqlmock.AnyArg(), "user_disabled", userID).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)
	user, err := svc.UpdateUserStatus(userID.String(), "disabled")
	if err != nil {
		t.Fatalf("UpdateUserStatus: %v", err)
	}
	if user.Status != "disabled" {
		t.Fatalf("expected disabled user, got %s", user.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestVerifyTokenRequiresActiveSession(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)

	token, err := svc.jwtManager.GenerateWithSession(userID.String(), orgID.String(), sessionID.String(), "jti-1")
	if err != nil {
		t.Fatalf("GenerateWithSession: %v", err)
	}

	sessionColumns := []string{"id", "user_id", "org_id", "status", "ip", "user_agent", "device_name", "last_seen_at", "expires_at", "revoked_at", "revoked_reason", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT \* FROM "sessions"`).
		WithArgs(sessionID, 1).
		WillReturnRows(sqlmock.NewRows(sessionColumns).
			AddRow(sessionID, userID, orgID, "revoked", "", "", "", time.Now(), time.Now().Add(time.Hour), time.Now(), "test", time.Now(), time.Now()))

	result := svc.VerifyToken(token)
	if result.Active {
		t.Fatalf("expected inactive token for revoked session, got %+v", result)
	}
	if result.Reason != "session revoked" {
		t.Fatalf("expected session revoked reason, got %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestVerifyTokenRequiresActiveUser(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)

	token, err := svc.jwtManager.GenerateWithSession(userID.String(), orgID.String(), sessionID.String(), "jti-1")
	if err != nil {
		t.Fatalf("GenerateWithSession: %v", err)
	}

	sessionColumns := []string{"id", "user_id", "org_id", "status", "ip", "user_agent", "device_name", "last_seen_at", "expires_at", "revoked_at", "revoked_reason", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT \* FROM "sessions"`).
		WithArgs(sessionID, 1).
		WillReturnRows(sqlmock.NewRows(sessionColumns).
			AddRow(sessionID, userID, orgID, "active", "", "", "", time.Now(), time.Now().Add(time.Hour), nil, "", time.Now(), time.Now()))

	userColumns := []string{
		"id", "org_id", "username", "email", "phone", "password_hash", "display_name", "avatar", "status", "source",
		"failed_login_attempts", "locked_until", "password_changed_at", "must_change_password", "mfa_enabled", "mfa_secret", "mfa_confirmed_at", "last_login_at",
		"created_at", "updated_at",
	}
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", "admin@example.com", "", "", "Admin", "", "disabled", "local",
				0, nil, time.Now(), false, false, "", nil, nil, time.Now(), time.Now()))

	result := svc.VerifyToken(token)
	if result.Active {
		t.Fatalf("expected inactive token for disabled user, got %+v", result)
	}
	if result.Reason != "user is disabled" {
		t.Fatalf("expected user disabled reason, got %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestRefreshTokenRotatesAtomically(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	refreshID := uuid.MustParse("00000000-0000-0000-0000-000000000030")
	refreshToken := "refresh-token-1"
	hash := sha256.Sum256([]byte(refreshToken))
	hashStr := hex.EncodeToString(hash[:])

	refreshColumns := []string{"id", "user_id", "session_id", "jti", "token_hash", "expires_at", "revoked", "revoked_at", "revoked_reason", "created_at"}
	mock.ExpectQuery(`SELECT \* FROM "refresh_tokens"`).
		WithArgs(hashStr, 1).
		WillReturnRows(sqlmock.NewRows(refreshColumns).
			AddRow(refreshID, userID, sessionID, "jti-old", hashStr, time.Now().Add(time.Hour), false, nil, "", time.Now()))

	userColumns := []string{
		"id", "org_id", "username", "email", "phone", "password_hash", "display_name", "avatar", "status", "source",
		"failed_login_attempts", "locked_until", "password_changed_at", "must_change_password", "last_login_at",
		"created_at", "updated_at",
	}
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", "admin@example.com", "", "", "Admin", "", "active", "local",
				0, nil, time.Now(), false, nil, time.Now(), time.Now()))

	sessionColumns := []string{"id", "user_id", "org_id", "status", "ip", "user_agent", "device_name", "last_seen_at", "expires_at", "revoked_at", "revoked_reason", "created_at", "updated_at"}
	mock.ExpectQuery(`SELECT \* FROM "sessions"`).
		WithArgs(sessionID, 1).
		WillReturnRows(sqlmock.NewRows(sessionColumns).
			AddRow(sessionID, userID, orgID, "active", "", "", "", time.Now(), time.Now().Add(time.Hour), nil, "", time.Now(), time.Now()))

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "refresh_tokens"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "refresh_tokens"`).
		WithArgs(sqlmock.AnyArg(), userID, sessionID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), false, nil, "", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "sessions" SET "last_seen_at"=NOW\(\),"updated_at"=\$1 WHERE id = \$2 AND status = \$3`).
		WithArgs(sqlmock.AnyArg(), sessionID, "active").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)
	result, err := svc.RefreshToken(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" || result.RefreshToken == refreshToken {
		t.Fatalf("expected rotated tokens, got %+v", result)
	}
	if result.SessionID != sessionID.String() {
		t.Fatalf("expected session %s, got %s", sessionID, result.SessionID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestRefreshTokenReplayRevokesSession(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	refreshID := uuid.MustParse("00000000-0000-0000-0000-000000000030")
	refreshToken := "refresh-token-1"
	hash := sha256.Sum256([]byte(refreshToken))
	hashStr := hex.EncodeToString(hash[:])

	mock.ExpectQuery(`SELECT \* FROM "refresh_tokens"`).
		WithArgs(hashStr, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	refreshColumns := []string{"id", "user_id", "session_id", "jti", "token_hash", "expires_at", "revoked", "revoked_at", "revoked_reason", "created_at"}
	mock.ExpectQuery(`SELECT \* FROM "refresh_tokens"`).
		WithArgs(hashStr, 1).
		WillReturnRows(sqlmock.NewRows(refreshColumns).
			AddRow(refreshID, userID, sessionID, "jti-old", hashStr, time.Now().Add(time.Hour), true, time.Now(), "rotated", time.Now()))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "sessions"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE "refresh_tokens"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)
	_, err = svc.RefreshToken(context.Background(), refreshToken)
	if err == nil || !strings.Contains(err.Error(), "reuse") {
		t.Fatalf("expected refresh token reuse error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestDeleteUserSoftDeletesAndCleansIdentityDataInTransaction(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	userColumns := []string{
		"id", "org_id", "username", "email", "phone", "password_hash", "display_name", "avatar", "status", "source",
		"failed_login_attempts", "locked_until", "password_changed_at", "must_change_password", "mfa_enabled", "mfa_secret", "mfa_confirmed_at", "last_login_at",
		"deleted_at", "created_at", "updated_at",
	}
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows(userColumns).
			AddRow(userID, orgID, "admin", "admin@example.com", "", "", "Admin", "", "active", "local",
				0, nil, time.Now(), false, true, "encrypted-secret", time.Now(), nil, nil, time.Now(), time.Now()))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "sessions"`).
		WithArgs(sqlmock.AnyArg(), "user_deleted", "revoked", sqlmock.AnyArg(), userID, "active").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`UPDATE "refresh_tokens"`).
		WithArgs(true, sqlmock.AnyArg(), "user_deleted", userID).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`DELETE FROM "user_credentials" WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM "password_histories" WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec(`DELETE FROM "mfa_recovery_codes" WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 10))
	mock.ExpectExec(`UPDATE "users"`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	events := &captureEventPublisher{}
	svc := NewAuthService(repository.NewUserRepository(db), nil, nil, nil, sharedcfg.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}, nil, nil, nil)
	svc.kafkaProd = events
	if err := svc.DeleteUser(userID.String()); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if len(events.events) != 1 {
		t.Fatalf("expected one delete event, got %d", len(events.events))
	}
	if events.events[0].EventType != "user.deleted" || events.events[0].UserID != userID.String() || events.events[0].OrgID != orgID.String() {
		t.Fatalf("unexpected delete event: %+v", events.events[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
