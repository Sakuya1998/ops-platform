package service

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/ops-platform/iam-svc/internal/repository"
	"github.com/ops-platform/pkg/cache"
	"github.com/ops-platform/pkg/config"
	"github.com/ops-platform/pkg/kafka"
)

func TestIAMConsumer_UserDeletedClearsRoles(t *testing.T) {
	silenceStandardLogger(t)

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	db := openMockDB(t, sqlDB)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "user_roles" WHERE user_id = \$1`).WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	consumer := NewIAMConsumer(repository.NewRoleRepository(db))
	event := kafka.Event{EventType: "user.deleted", UserID: userID.String()}
	payload, _ := json.Marshal(event)
	if err := consumer.HandleEvent(context.Background(), event.EventType, payload); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestIAMConsumer_IgnoresUnrelatedEvent(t *testing.T) {
	silenceStandardLogger(t)

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	db := openMockDB(t, sqlDB)

	consumer := NewIAMConsumer(repository.NewRoleRepository(db))
	event := kafka.Event{EventType: "user.updated", UserID: uuid.NewString()}
	payload, _ := json.Marshal(event)
	if err := consumer.HandleEvent(context.Background(), event.EventType, payload); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestIAMConsumer_UserRoleChangedInvalidatesPermissionCache(t *testing.T) {
	silenceStandardLogger(t)

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	roleID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	expectPermissionLoad(mock, userID, orgID, roleID, "ops_viewer", "user:read")
	expectPermissionLoad(mock, userID, orgID, roleID, "ops_viewer", "user:read")

	permissionCache := cache.New(cache.Options{DefaultTTL: time.Minute})
	iamSvc := NewIAMService(
		repository.NewRoleRepository(db),
		repository.NewPermissionRepository(db),
		nil,
		config.JWTConfig{Secret: "test-secret", ExpireHour: 2, Issuer: "ops-test"},
	).WithPermissionCache(permissionCache)

	allowed, reason, _ := iamSvc.CheckPermission(userID.String(), orgID.String(), "GET", "/api/v1/users")
	if !allowed {
		t.Fatalf("expected first check allowed, reason=%s", reason)
	}

	consumer := NewIAMConsumer(repository.NewRoleRepository(db)).WithIAMService(iamSvc)
	event := kafka.Event{EventType: "user.role_changed", UserID: userID.String()}
	payload, _ := json.Marshal(event)
	if err := consumer.HandleEvent(context.Background(), event.EventType, payload); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}

	allowed, reason, _ = iamSvc.CheckPermission(userID.String(), orgID.String(), "GET", "/api/v1/users")
	if !allowed {
		t.Fatalf("expected second check allowed, reason=%s", reason)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func expectPermissionLoad(mock sqlmock.Sqlmock, userID, orgID, roleID uuid.UUID, roleCode, permissionCode string) {
	roleRows := sqlmock.NewRows([]string{"id", "org_id", "name", "code", "description", "is_system", "created_at", "updated_at"}).
		AddRow(roleID, orgID, "Role", roleCode, "", true, time.Now(), time.Now())
	permRows := sqlmock.NewRows([]string{"id", "name", "code", "resource", "action", "type", "sort", "created_at"}).
		AddRow(uuid.New(), "Permission", permissionCode, "user", "read", "api", 100, time.Now())
	mock.ExpectQuery(`SELECT r\.\* FROM roles r JOIN user_roles ur ON r\.id = ur\.role_id WHERE ur\.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(roleRows)
	mock.ExpectQuery(`SELECT p\.\* FROM permissions p JOIN role_permissions rp ON p\.id = rp\.permission_id WHERE rp\.role_id = \$1`).
		WithArgs(roleID).
		WillReturnRows(permRows)
}

func silenceStandardLogger(t *testing.T) {
	t.Helper()
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetOutput(io.Discard)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
	})
}
