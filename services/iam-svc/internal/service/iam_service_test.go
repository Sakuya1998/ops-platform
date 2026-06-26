package service

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/ops-platform/iam-svc/internal/model"
	"github.com/ops-platform/iam-svc/internal/repository"
	"github.com/ops-platform/pkg/cache"
	"github.com/ops-platform/pkg/config"
)

func TestIAMServiceRejectsInvalidUUIDInputs(t *testing.T) {
	svc := NewIAMService(nil, nil, nil, config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"})

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "assign roles invalid user", run: func() error { return svc.AssignRoles("not-a-uuid", []string{}) }},
		{name: "assign roles invalid role", run: func() error { return svc.AssignRoles(uuid.NewString(), []string{"bad-role"}) }},
		{name: "create role invalid org", run: func() error {
			_, err := svc.CreateRole("not-a-uuid", "Admin", "admin", "")
			return err
		}},
		{name: "list roles invalid org", run: func() error {
			_, err := svc.ListRoles("not-a-uuid")
			return err
		}},
		{name: "get user roles invalid user", run: func() error {
			_, err := svc.GetUserRoles("not-a-uuid")
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("expected error, got panic: %v", r)
				}
			}()
			err := tt.run()
			if err == nil {
				t.Fatal("expected invalid uuid error")
			}
			if !strings.Contains(err.Error(), "invalid") {
				t.Fatalf("expected invalid uuid error, got %v", err)
			}
		})
	}
}

func TestResolveToken(t *testing.T) {
	cfg := config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"}
	svc := NewIAMService(nil, nil, nil, cfg)
	token, err := svc.jwtMgr.GenerateWithSession("user-1", "org-1", "session-1", "jti-1")
	if err != nil {
		t.Fatalf("Generate token: %v", err)
	}
	userCtx, err := svc.ResolveToken("Bearer " + token)
	if err != nil {
		t.Fatalf("ResolveToken: %v", err)
	}
	if userCtx.UserID != "user-1" || userCtx.OrgID != "org-1" || userCtx.SessionID != "session-1" {
		t.Fatalf("unexpected user ctx: %+v", userCtx)
	}
}

func TestBuildTree_Empty(t *testing.T) {
	result := buildTree(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 roots, got %d", len(result))
	}
}

func TestCheckPermissionRejectsUnmappedGETRoute(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	roleID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	roleRows := sqlmock.NewRows([]string{"id", "org_id", "name", "code", "description", "is_system", "created_at", "updated_at"}).
		AddRow(roleID, orgID, "Viewer", "viewer", "", false, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT r\.\* FROM roles r JOIN user_roles ur ON r\.id = ur\.role_id WHERE ur\.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(roleRows)
	mock.ExpectQuery(`SELECT p\.\* FROM permissions p JOIN role_permissions rp ON p\.id = rp\.permission_id WHERE rp\.role_id = \$1`).
		WithArgs(roleID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "resource", "action", "type", "sort", "created_at"}))

	svc := NewIAMService(repository.NewRoleRepository(db), repository.NewPermissionRepository(db), nil, config.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	})
	allowed, reason, _ := svc.CheckPermission(userID.String(), orgID.String(), "GET", "/api/v1/unmapped")
	if allowed {
		t.Fatalf("expected unmapped GET route to be denied")
	}
	if !strings.Contains(reason, "no permission mapping") {
		t.Fatalf("expected missing mapping reason, got %q", reason)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestBatchCheckPermissionReusesUserPermissions(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	roleID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	roleRows := sqlmock.NewRows([]string{"id", "org_id", "name", "code", "description", "is_system", "created_at", "updated_at"}).
		AddRow(roleID, orgID, "Viewer", "ops_viewer", "", true, time.Now(), time.Now())
	permRows := sqlmock.NewRows([]string{"id", "name", "code", "resource", "action", "type", "sort", "created_at"}).
		AddRow(uuid.MustParse("00000000-0000-0000-0000-000000000030"), "Read User", "user:read", "user", "read", "api", 100, time.Now())
	mock.ExpectQuery(`SELECT r\.\* FROM roles r JOIN user_roles ur ON r\.id = ur\.role_id WHERE ur\.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(roleRows)
	mock.ExpectQuery(`SELECT p\.\* FROM permissions p JOIN role_permissions rp ON p\.id = rp\.permission_id WHERE rp\.role_id = \$1`).
		WithArgs(roleID).
		WillReturnRows(permRows)

	svc := NewIAMService(repository.NewRoleRepository(db), repository.NewPermissionRepository(db), nil, config.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	})
	results, err := svc.BatchCheckPermission(userID.String(), orgID.String(), []PermissionCheck{
		{Method: "GET", Path: "/api/v1/users"},
		{Method: "POST", Path: "/api/v1/users"},
	})
	if err != nil {
		t.Fatalf("BatchCheckPermission: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].Allowed || results[0].RequiredPermission != "user:read" {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
	if results[1].Allowed || results[1].RequiredPermission != "user:create" {
		t.Fatalf("unexpected second result: %+v", results[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCheckPermissionUsesPermissionCache(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	roleID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	roleRows := sqlmock.NewRows([]string{"id", "org_id", "name", "code", "description", "is_system", "created_at", "updated_at"}).
		AddRow(roleID, orgID, "Viewer", "ops_viewer", "", true, time.Now(), time.Now())
	permRows := sqlmock.NewRows([]string{"id", "name", "code", "resource", "action", "type", "sort", "created_at"}).
		AddRow(uuid.MustParse("00000000-0000-0000-0000-000000000030"), "Read User", "user:read", "user", "read", "api", 100, time.Now())
	mock.ExpectQuery(`SELECT r\.\* FROM roles r JOIN user_roles ur ON r\.id = ur\.role_id WHERE ur\.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(roleRows)
	mock.ExpectQuery(`SELECT p\.\* FROM permissions p JOIN role_permissions rp ON p\.id = rp\.permission_id WHERE rp\.role_id = \$1`).
		WithArgs(roleID).
		WillReturnRows(permRows)

	svc := NewIAMService(repository.NewRoleRepository(db), repository.NewPermissionRepository(db), nil, config.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}).WithPermissionCache(cache.New(cache.Options{DefaultTTL: time.Minute}))

	for i := 0; i < 2; i++ {
		allowed, reason, _ := svc.CheckPermission(userID.String(), orgID.String(), "GET", "/api/v1/users")
		if !allowed {
			t.Fatalf("expected allowed on iteration %d, reason=%s", i, reason)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCheckPermissionUsesAPIPermissionRepository(t *testing.T) {
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
		AddRow(roleID, orgID, "Custom Reader", "custom_reader", "", true, time.Now(), time.Now())
	permRows := sqlmock.NewRows([]string{"id", "name", "code", "resource", "action", "type", "sort", "created_at"}).
		AddRow(uuid.MustParse("00000000-0000-0000-0000-000000000030"), "Read Custom", "custom:read", "custom", "read", "api", 100, time.Now())
	apiRows := sqlmock.NewRows([]string{"id", "method", "path_pattern", "permission_code", "description", "enabled", "created_at", "updated_at"}).
		AddRow(apiPermissionID, "GET", "/api/v1/custom", "custom:read", "Read custom resource", true, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT r\.\* FROM roles r JOIN user_roles ur ON r\.id = ur\.role_id WHERE ur\.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(roleRows)
	mock.ExpectQuery(`SELECT p\.\* FROM permissions p JOIN role_permissions rp ON p\.id = rp\.permission_id WHERE rp\.role_id = \$1`).
		WithArgs(roleID).
		WillReturnRows(permRows)
	mock.ExpectQuery(`SELECT \* FROM "api_permissions" WHERE method = \$1 AND path_pattern = \$2 AND enabled = \$3 ORDER BY "api_permissions"\."id" LIMIT \$4`).
		WithArgs("GET", "/api/v1/custom", true, 1).
		WillReturnRows(apiRows)

	svc := NewIAMService(repository.NewRoleRepository(db), repository.NewPermissionRepository(db), nil, config.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}).WithAPIPermissionRepository(repository.NewAPIPermissionRepository(db))

	allowed, reason, _ := svc.CheckPermission(userID.String(), orgID.String(), "GET", "/api/v1/custom")
	if !allowed {
		t.Fatalf("expected allowed from api_permissions, reason=%s", reason)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestBuildTree_SingleRoot(t *testing.T) {
	perms := []model.Permission{
		{ID: uuid.MustParse("10000000-0000-0000-0000-000000000001"), Name: "Root", Code: "root", Resource: "test", Action: "read"},
	}
	result := buildTree(perms)
	if len(result) != 1 {
		t.Errorf("expected 1 root, got %d", len(result))
	}
	if result[0].Name != "Root" {
		t.Errorf("expected 'Root', got '%s'", result[0].Name)
	}
}

func TestBuildTree_ParentChild(t *testing.T) {
	pid := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	cid := uuid.MustParse("10000000-0000-0000-0000-000000000002")
	perms := []model.Permission{
		{ID: pid, Name: "Parent", Code: "parent", Resource: "test", Action: "read"},
		{ID: cid, ParentID: &pid, Name: "Child", Code: "child", Resource: "test", Action: "create"},
	}
	result := buildTree(perms)
	if len(result) != 1 {
		t.Fatalf("expected 1 root, got %d", len(result))
	}
	if len(result[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result[0].Children))
	}
	if result[0].Children[0].Name != "Child" {
		t.Errorf("expected 'Child', got '%s'", result[0].Children[0].Name)
	}
}

func TestBuildTree_TwoRoots(t *testing.T) {
	id1 := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	id2 := uuid.MustParse("10000000-0000-0000-0000-000000000002")
	pid := uuid.MustParse("10000000-0000-0000-0000-000000000003")
	perms := []model.Permission{
		{ID: id1, Name: "Root1", Code: "r1", Resource: "a", Action: "read"},
		{ID: id2, Name: "Root2", Code: "r2", Resource: "b", Action: "read"},
		{ID: pid, ParentID: &id1, Name: "ChildOf1", Code: "c1", Resource: "a", Action: "create"},
	}
	result := buildTree(perms)
	if len(result) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(result))
	}
	root1 := result[0]
	if root1.ID != id1 {
		t.Errorf("expected first root to be Root1")
	}
	if len(root1.Children) != 1 {
		t.Errorf("expected Root1 to have 1 child, got %d", len(root1.Children))
	}
}

func TestBuildTree_IgnoresMissingParent(t *testing.T) {
	pid := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	cid := uuid.MustParse("10000000-0000-0000-0000-000000000002")
	bad := uuid.MustParse("10000000-0000-0000-0000-00000000FFFF")
	perms := []model.Permission{
		{ID: pid, Name: "Parent", Code: "parent", Resource: "test", Action: "read"},
		{ID: cid, ParentID: &bad, Name: "Orphan", Code: "orphan", Resource: "test", Action: "create"},
	}
	result := buildTree(perms)
	if len(result) != 1 {
		t.Errorf("expected 1 root (orphan ignored), got %d", len(result))
	}
}

func TestAPIPermissionMigrationHasSeedMappings(t *testing.T) {
	expected := []string{
		"('GET', '/api/v1/auth/me', 'user:read'",
		"('GET', '/api/v1/users', 'user:read'",
		"('PUT', '/api/v1/users/:id/roles', 'role:assign'",
		"('GET', '/api/v1/permissions', 'permission:read'",
		"('GET', '/api/v1/api-permissions', 'api_permission:read'",
		"('POST', '/api/v1/api-permissions', 'api_permission:create'",
		"('GET', '/api/v1/audit-logs/event-types', 'audit:read'",
		"('GET', '/api/v1/system/config', 'org:read'",
	}
	migration, err := os.ReadFile("../../migrations/004_api_permissions.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	content := string(migration)
	for _, snippet := range expected {
		if !strings.Contains(content, snippet) {
			t.Errorf("missing api permission seed snippet: %s", snippet)
		}
	}
}

func TestCreateAPIPermissionRequiresRepository(t *testing.T) {
	svc := NewIAMService(nil, nil, nil, config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"})
	_, err := svc.CreateAPIPermission(CreateAPIPermissionInput{
		Method: "GET", PathPattern: "/api/v1/custom", PermissionCode: "custom:read", Enabled: true,
	})
	if err == nil {
		t.Fatal("expected missing repository error")
	}
}

func TestCreateAPIPermissionUppercasesMethod(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "api_permissions"`).
		WithArgs(sqlmock.AnyArg(), "GET", "/api/v1/custom", "custom:read", "Read custom", true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svc := NewIAMService(repository.NewRoleRepository(db), repository.NewPermissionRepository(db), nil, config.JWTConfig{
		Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test",
	}).WithAPIPermissionRepository(repository.NewAPIPermissionRepository(db))
	created, err := svc.CreateAPIPermission(CreateAPIPermissionInput{
		Method: "get", PathPattern: "/api/v1/custom", PermissionCode: "custom:read", Description: "Read custom", Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateAPIPermission: %v", err)
	}
	if created.Method != "GET" {
		t.Fatalf("expected method GET, got %+v", created)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestNormalizeRoutePath(t *testing.T) {
	cases := map[string]string{
		"/api/v1/users/10000000-0000-0000-0000-000000000001":                       "/api/v1/users/:id",
		"/api/v1/notify/templates/10000000-0000-0000-0000-000000000001?debug=true": "/api/v1/notify/templates/:id",
		"/api/v1/roles/role_123456":                                                "/api/v1/roles/:id",
		"/api/v1/users":                                                            "/api/v1/users",
		"/api/v1":                                                                  "/api/v1",
	}
	for input, expected := range cases {
		if got := normalizeRoutePath(input); got != expected {
			t.Errorf("normalizeRoutePath(%q) = %q, expected %q", input, got, expected)
		}
	}
}
