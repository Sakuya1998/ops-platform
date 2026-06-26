package repository

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/ops-platform/iam-svc/internal/model"
)

func TestPolicyRepositoryCreatesResourcePolicyAndBinding(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	repo := NewPolicyRepository(db)
	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	roleID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	resourceID := uuid.MustParse("00000000-0000-0000-0000-000000000020")
	policyID := uuid.MustParse("00000000-0000-0000-0000-000000000030")

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "resources"`).
		WithArgs(resourceID, orgID, "server", "srv-001", "Prod Server", "{}", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.CreateResource(&model.Resource{
		ID: resourceID, OrgID: orgID, ResourceType: "server", ResourceKey: "srv-001", Name: "Prod Server", Attributes: "{}",
	}); err != nil {
		t.Fatalf("CreateResource: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "policies"`).
		WithArgs(policyID, orgID, "prod-server-read", "allow", "server", "read", `{"env":"prod"}`, true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.CreatePolicy(&model.Policy{
		ID: policyID, OrgID: orgID, Name: "prod-server-read", Effect: "allow",
		ResourceType: "server", Action: "read", Condition: `{"env":"prod"}`, IsEnabled: true,
	}); err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "policy_bindings"`).
		WithArgs(policyID, "role", roleID.String(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := repo.BindPolicy(&model.PolicyBinding{
		PolicyID: policyID, SubjectType: "role", SubjectID: roleID.String(), CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("BindPolicy: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
