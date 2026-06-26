package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/service"
	"github.com/Sakuya1998/ops-platform/pkg/config"
	iamv1 "github.com/Sakuya1998/ops-platform/pkg/proto/iam/v1"
)

func TestServerCheckPermission(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	roleID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	apiPermissionID := uuid.MustParse("00000000-0000-0000-0000-000000000040")
	roleRows := sqlmock.NewRows([]string{"id", "org_id", "name", "code", "description", "is_system", "created_at", "updated_at"}).
		AddRow(roleID, orgID, "Viewer", "ops_viewer", "", true, time.Now(), time.Now())
	permRows := sqlmock.NewRows([]string{"id", "name", "code", "resource", "action", "type", "sort", "created_at"}).
		AddRow(uuid.MustParse("00000000-0000-0000-0000-000000000030"), "Read User", "user:read", "user", "read", "api", 100, time.Now())
	apiRows := sqlmock.NewRows([]string{"id", "method", "path_pattern", "permission_code", "description", "enabled", "created_at", "updated_at"}).
		AddRow(apiPermissionID, "GET", "/api/v1/users", "user:read", "List users", true, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT r\.\* FROM roles r JOIN user_roles ur ON r\.id = ur\.role_id WHERE ur\.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(roleRows)
	mock.ExpectQuery(`SELECT p\.\* FROM permissions p JOIN role_permissions rp ON p\.id = rp\.permission_id WHERE rp\.role_id = \$1`).
		WithArgs(roleID).
		WillReturnRows(permRows)
	mock.ExpectQuery(`SELECT \* FROM "api_permissions" WHERE method = \$1 AND path_pattern = \$2 AND enabled = \$3 ORDER BY "api_permissions"\."id" LIMIT \$4`).
		WithArgs("GET", "/api/v1/users", true, 1).
		WillReturnRows(apiRows)

	iamSvc := service.NewIAMService(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		nil,
		config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"},
	).WithAPIPermissionRepository(repository.NewAPIPermissionRepository(db))
	server := NewServer(iamSvc)

	resp, err := server.CheckPermission(context.Background(), &iamv1.CheckPermissionRequest{
		UserId: userID.String(),
		OrgId:  orgID.String(),
		Method: "GET",
		Path:   "/api/v1/users",
	})
	if err != nil {
		t.Fatalf("CheckPermission: %v", err)
	}
	if !resp.Allowed || len(resp.Roles) != 1 || resp.Roles[0] != "ops_viewer" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestServerListAPIPermissions(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	permissionID := uuid.MustParse("00000000-0000-0000-0000-000000000041")
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "method", "path_pattern", "permission_code", "description", "enabled", "created_at", "updated_at"}).
		AddRow(permissionID, "POST", "/api/v1/roles", "role:create", "Create role", true, now, now)
	mock.ExpectQuery(`SELECT \* FROM "api_permissions" ORDER BY method ASC,path_pattern ASC`).
		WillReturnRows(rows)

	iamSvc := service.NewIAMService(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		nil,
		config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"},
	).WithAPIPermissionRepository(repository.NewAPIPermissionRepository(db))
	server := NewServer(iamSvc)

	resp, err := server.ListAPIPermissions(context.Background(), &iamv1.ListAPIPermissionsRequest{})
	if err != nil {
		t.Fatalf("ListAPIPermissions: %v", err)
	}
	if len(resp.ApiPermissions) != 1 {
		t.Fatalf("expected one api permission, got %d", len(resp.ApiPermissions))
	}
	got := resp.ApiPermissions[0]
	if got.Id != permissionID.String() || got.Method != "POST" || got.PathPattern != "/api/v1/roles" ||
		got.PermissionCode != "role:create" || got.Description != "Create role" || !got.Enabled {
		t.Fatalf("unexpected api permission: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestServerGetUserPermissions(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	roleID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	now := time.Now()
	roleRows := sqlmock.NewRows([]string{"id", "org_id", "name", "code", "description", "is_system", "created_at", "updated_at"}).
		AddRow(roleID, orgID, "Operator", "ops_engineer", "", false, now, now)
	permRows := sqlmock.NewRows([]string{"id", "name", "code", "resource", "action", "type", "sort", "created_at"}).
		AddRow(uuid.MustParse("00000000-0000-0000-0000-000000000031"), "Read Role", "role:read", "role", "read", "api", 100, now).
		AddRow(uuid.MustParse("00000000-0000-0000-0000-000000000032"), "Create Role", "role:create", "role", "create", "api", 110, now)
	mock.ExpectQuery(`SELECT r\.\* FROM roles r JOIN user_roles ur ON r\.id = ur\.role_id WHERE ur\.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(roleRows)
	mock.ExpectQuery(`SELECT p\.\* FROM permissions p JOIN role_permissions rp ON p\.id = rp\.permission_id WHERE rp\.role_id = \$1`).
		WithArgs(roleID).
		WillReturnRows(permRows)

	iamSvc := service.NewIAMService(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		nil,
		config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"},
	)
	server := NewServer(iamSvc)

	resp, err := server.GetUserPermissions(context.Background(), &iamv1.GetUserPermissionsRequest{UserId: userID.String()})
	if err != nil {
		t.Fatalf("GetUserPermissions: %v", err)
	}
	want := []string{"role:read", "role:create"}
	if len(resp.Codes) != len(want) {
		t.Fatalf("expected %d codes, got %d: %+v", len(want), len(resp.Codes), resp.Codes)
	}
	for i := range want {
		if resp.Codes[i] != want[i] {
			t.Fatalf("unexpected permission codes: got %+v want %+v", resp.Codes, want)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
