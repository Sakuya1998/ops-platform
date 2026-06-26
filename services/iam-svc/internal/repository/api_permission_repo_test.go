package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/model"
)

func TestAPIPermissionRepositoryFindEnabledByRoute(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	routeID := uuid.MustParse("00000000-0000-0000-0000-000000000040")
	rows := sqlmock.NewRows([]string{"id", "method", "path_pattern", "permission_code", "description", "enabled", "created_at", "updated_at"}).
		AddRow(routeID, "GET", "/api/v1/users/:id", "user:read", "Read user detail", true, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "api_permissions" WHERE method = \$1 AND path_pattern = \$2 AND enabled = \$3 ORDER BY "api_permissions"\."id" LIMIT \$4`).
		WithArgs("GET", "/api/v1/users/:id", true, 1).
		WillReturnRows(rows)

	got, err := NewAPIPermissionRepository(db).GetByRoute("GET", "/api/v1/users/:id")
	if err != nil {
		t.Fatalf("GetByRoute: %v", err)
	}
	if got.PermissionCode != "user:read" {
		t.Fatalf("expected user:read, got %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestAPIPermissionBeforeCreateSetsID(t *testing.T) {
	permission := &model.APIPermission{}
	if err := permission.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate: %v", err)
	}
	if permission.ID == uuid.Nil {
		t.Fatal("expected generated id")
	}
}

func TestAPIPermissionRepositoryCRUD(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)
	repo := NewAPIPermissionRepository(db)

	routeID := uuid.MustParse("00000000-0000-0000-0000-000000000041")
	listRows := sqlmock.NewRows([]string{"id", "method", "path_pattern", "permission_code", "description", "enabled", "created_at", "updated_at"}).
		AddRow(routeID, "GET", "/api/v1/users", "user:read", "List users", true, time.Now(), time.Now())
	mock.ExpectQuery(`SELECT \* FROM "api_permissions" ORDER BY method ASC,path_pattern ASC`).
		WillReturnRows(listRows)
	list, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].PermissionCode != "user:read" {
		t.Fatalf("unexpected list: %+v", list)
	}

	created := &model.APIPermission{
		ID: routeID, Method: "POST", PathPattern: "/api/v1/custom",
		PermissionCode: "custom:create", Description: "Create custom", Enabled: true,
	}
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "api_permissions"`).
		WithArgs(routeID, "POST", "/api/v1/custom", "custom:create", "Create custom", true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Create(created); err != nil {
		t.Fatalf("Create: %v", err)
	}

	updated := &model.APIPermission{
		ID: routeID, Method: "PUT", PathPattern: "/api/v1/custom/:id",
		PermissionCode: "custom:update", Description: "Update custom", Enabled: true,
	}
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "api_permissions" SET`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Update(updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "api_permissions" WHERE id = \$1`).
		WithArgs(routeID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.Delete(routeID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
