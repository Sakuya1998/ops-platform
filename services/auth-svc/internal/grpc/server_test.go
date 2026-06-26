package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	authconfig "github.com/ops-platform/auth-svc/internal/config"
	"github.com/ops-platform/auth-svc/internal/repository"
	"github.com/ops-platform/auth-svc/internal/service"
	"github.com/ops-platform/pkg/config"
	secretcrypto "github.com/ops-platform/pkg/crypto"
	sharedjwt "github.com/ops-platform/pkg/jwt"
	authv1 "github.com/ops-platform/pkg/proto/auth/v1"
	"gorm.io/gorm"
)

func TestServerListUsersMapsIdentityFields(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	now := time.Now().UTC()
	mock.ExpectQuery(`SELECT count\(\*\) FROM "users" WHERE org_id = \$1 AND deleted_at IS NULL`).
		WithArgs(orgID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	rows := sqlmock.NewRows(userColumns()).AddRow(
		userID, orgID, "admin", "admin@example.com", "13800000000", "hash", "Admin", "",
		"active", "local", 0, nil, now, false, true, "secret", now, now, nil, now, now,
	)
	mock.ExpectQuery(`SELECT \* FROM "users" WHERE org_id = \$1 AND deleted_at IS NULL ORDER BY created_at DESC LIMIT \$2`).
		WithArgs(orgID, 20).
		WillReturnRows(rows)

	server := NewServer(newTestAuthService(db), nil)
	resp, err := server.ListUsers(context.Background(), &authv1.ListUsersRequest{OrgId: orgID.String(), Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if resp.Total != 1 || len(resp.Users) != 1 {
		t.Fatalf("unexpected list response: %+v", resp)
	}
	got := resp.Users[0]
	if got.Id != userID.String() || got.Username != "admin" || got.Email != "admin@example.com" || !got.MfaEnabled || got.CreatedAt == nil {
		t.Fatalf("unexpected user: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestServerListOrganizations(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{"id", "name", "code", "description", "logo", "status", "created_at", "updated_at"}).
		AddRow(orgID, "Default", "default", "Default org", "", "active", now, now)
	mock.ExpectQuery(`SELECT \* FROM "organizations" ORDER BY created_at DESC`).
		WillReturnRows(rows)

	server := NewServer(newTestAuthService(db), nil)
	resp, err := server.ListOrganizations(context.Background(), &authv1.ListOrganizationsRequest{})
	if err != nil {
		t.Fatalf("ListOrganizations: %v", err)
	}
	if len(resp.Organizations) != 1 || resp.Organizations[0].Id != orgID.String() || resp.Organizations[0].Code != "default" {
		t.Fatalf("unexpected organizations: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestServerValidateTokenReturnsClaims(t *testing.T) {
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	authSvc := newTestAuthService(db)
	userID := "00000000-0000-0000-0000-000000000010"
	orgID := "00000000-0000-0000-0000-000000000001"
	token, err := serviceToken("test-secret", userID, orgID)
	if err != nil {
		t.Fatalf("token: %v", err)
	}

	server := NewServer(authSvc, nil)
	resp, err := server.ValidateToken(context.Background(), &authv1.ValidateTokenRequest{Token: token})
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if !resp.Valid || resp.UserId != userID || resp.OrgId != orgID {
		t.Fatalf("unexpected validate response: %+v", resp)
	}
}

func newTestAuthService(db *gorm.DB) *service.AuthService {
	userRepo := repository.NewUserRepository(db)
	providerRepo := repository.NewProviderRepository(db)
	orgRepo := repository.NewOrganizationRepository(db)
	ldapSvc := service.NewLdapService(&authconfig.LDAPConfig{}, userRepo)
	return service.NewAuthService(
		userRepo,
		providerRepo,
		orgRepo,
		nil,
		config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"},
		nil,
		ldapSvc,
		secretcrypto.NewSecretBox("test-secret"),
	)
}

func serviceToken(secret, userID, orgID string) (string, error) {
	manager := sharedjwt.NewManager(secret, 2, "ops-test")
	return manager.GenerateWithSession(userID, orgID, "00000000-0000-0000-0000-000000000099", "jti-test")
}

func userColumns() []string {
	return []string{
		"id", "org_id", "username", "email", "phone", "password_hash", "display_name", "avatar",
		"status", "source", "failed_login_attempts", "locked_until", "password_changed_at",
		"must_change_password", "mfa_enabled", "mfa_secret", "mfa_confirmed_at",
		"last_login_at", "deleted_at", "created_at", "updated_at",
	}
}
