package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ops-platform/iam-svc/internal/repository"
	"github.com/ops-platform/iam-svc/internal/service"
	"github.com/ops-platform/pkg/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestHandlerReturnsBadRequestForInvalidRoleID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sqlDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)
	h := NewIAMHandler(service.NewIAMService(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		nil,
		config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"},
	))
	r := gin.New()
	r.GET("/api/v1/roles/:id", h.GetRole)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandlerReturnsNotFoundForMissingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)
	roleID := "00000000-0000-0000-0000-000000000020"
	mock.ExpectQuery(`SELECT \* FROM "roles" WHERE id = \$1 ORDER BY "roles"\."id" LIMIT \$2`).
		WithArgs(uuid.MustParse(roleID), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "org_id", "name", "code", "description", "is_system", "created_at", "updated_at"}))
	h := NewIAMHandler(service.NewIAMService(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		nil,
		config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"},
	))
	r := gin.New()
	r.GET("/api/v1/roles/:id", h.GetRole)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/roles/"+roleID, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", w.Code, w.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGetUserPermissionsReturnsPermissionCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
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
		AddRow(uuid.MustParse("00000000-0000-0000-0000-000000000030"), "Read User", "user:read", "user", "read", "api", 100, time.Now()).
		AddRow(uuid.MustParse("00000000-0000-0000-0000-000000000031"), "Read Audit", "audit:read", "audit", "read", "api", 500, time.Now())
	mock.ExpectQuery(`SELECT r\.\* FROM roles r JOIN user_roles ur ON r\.id = ur\.role_id WHERE ur\.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(roleRows)
	mock.ExpectQuery(`SELECT p\.\* FROM permissions p JOIN role_permissions rp ON p\.id = rp\.permission_id WHERE rp\.role_id = \$1`).
		WithArgs(roleID).
		WillReturnRows(permRows)

	h := NewIAMHandler(service.NewIAMService(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		nil,
		config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"},
	))
	r := gin.New()
	r.GET("/api/v1/users/:id/permissions", h.GetUserPermissions)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID.String()+"/permissions", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Data) != 2 || body.Data[0] != "user:read" || body.Data[1] != "audit:read" {
		t.Fatalf("unexpected permissions: %+v", body.Data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCheckPermissionAPIAllowsMappedPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
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

	h := NewIAMHandler(service.NewIAMService(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		nil,
		config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"},
	))
	r := gin.New()
	r.POST("/internal/v1/permissions/check", h.CheckPermissionAPI)

	body := []byte(`{"user_id":"` + userID.String() + `","org_id":"` + orgID.String() + `","method":"GET","path":"/api/v1/users"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/permissions/check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Allowed bool     `json:"allowed"`
			Reason  string   `json:"reason"`
			Roles   []string `json:"roles"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !resp.Data.Allowed || len(resp.Data.Roles) != 1 || resp.Data.Roles[0] != "ops_viewer" {
		t.Fatalf("unexpected response: %+v", resp.Data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestBatchCheckPermissionAPIReturnsResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
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

	h := NewIAMHandler(service.NewIAMService(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		nil,
		config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"},
	))
	r := gin.New()
	r.POST("/internal/v1/permissions/batch-check", h.BatchCheckPermissionAPI)

	body := []byte(`{"user_id":"` + userID.String() + `","org_id":"` + orgID.String() + `","checks":[{"method":"GET","path":"/api/v1/users"},{"method":"POST","path":"/api/v1/users"}]}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/internal/v1/permissions/batch-check", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Results []service.PermissionCheckResult `json:"results"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(resp.Data.Results) != 2 {
		t.Fatalf("expected 2 results, got %+v", resp.Data.Results)
	}
	if !resp.Data.Results[0].Allowed || resp.Data.Results[1].Allowed {
		t.Fatalf("unexpected results: %+v", resp.Data.Results)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCreateAPIPermissionReturnsCreatedRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
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

	h := NewIAMHandler(service.NewIAMService(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		nil,
		config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"},
	).WithAPIPermissionRepository(repository.NewAPIPermissionRepository(db)))
	r := gin.New()
	r.POST("/api/v1/api-permissions", h.CreateAPIPermission)

	body := []byte(`{"method":"get","path_pattern":"/api/v1/custom","permission_code":"custom:read","description":"Read custom","enabled":true}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-permissions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Method string `json:"Method"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp.Data.Method != "GET" {
		t.Fatalf("expected method GET, got %+v", resp.Data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func openMockDB(t *testing.T, sqlDB *sql.DB) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}
	return db
}
