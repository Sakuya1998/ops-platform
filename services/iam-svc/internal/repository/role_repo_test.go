package repository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestAssignPermissionsRollsBackWhenCreateFails(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	roleID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	permID := uuid.MustParse("00000000-0000-0000-0000-000000000030")
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "role_permissions" WHERE role_id = \$1`).
		WithArgs(roleID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO "role_permissions"`).
		WithArgs(roleID, permID, sqlmock.AnyArg()).
		WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	err = NewRoleRepository(db).AssignPermissions(roleID, []uuid.UUID{permID})
	if err == nil {
		t.Fatal("expected insert error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
