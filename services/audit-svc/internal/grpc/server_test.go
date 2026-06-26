package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/ops-platform/audit-svc/internal/repository"
	auditv1 "github.com/ops-platform/pkg/proto/audit/v1"
)

func TestServerListAuditLogs(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	logID := uuid.MustParse("00000000-0000-0000-0000-000000000100")
	now := time.Now().UTC()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "audit_logs" WHERE org_id = \$1 AND event_type = \$2`).
		WithArgs(orgID.String(), "user.login").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	rows := sqlmock.NewRows([]string{
		"id", "org_id", "user_id", "username", "event_type", "action", "resource_type", "resource_id",
		"detail", "ip", "user_agent", "session_id", "reason_code", "request_id", "created_at",
	}).AddRow(logID, orgID, userID, "admin", "user.login", "login", "auth", "", "login ok", "127.0.0.1", "agent", "session-1", "", "req-1", now)
	mock.ExpectQuery(`SELECT \* FROM "audit_logs" WHERE org_id = \$1 AND event_type = \$2 ORDER BY created_at DESC LIMIT \$3`).
		WithArgs(orgID.String(), "user.login", 20).
		WillReturnRows(rows)

	server := NewServer(repository.NewAuditRepository(db))
	resp, err := server.ListAuditLogs(context.Background(), &auditv1.ListAuditLogsRequest{
		OrgId:     orgID.String(),
		EventType: "user.login",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("ListAuditLogs: %v", err)
	}
	if resp.Total != 1 || len(resp.Logs) != 1 {
		t.Fatalf("unexpected list response: %+v", resp)
	}
	got := resp.Logs[0]
	if got.Id != logID.String() || got.OrgId != orgID.String() || got.UserId != userID.String() ||
		got.EventType != "user.login" || got.RequestId != "req-1" || got.CreatedAt == nil {
		t.Fatalf("unexpected audit log: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestServerRecordAuditEvent(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	mock.ExpectExec(`INSERT INTO "audit_logs"`).
		WithArgs(sqlmock.AnyArg(), orgID, userID, "admin", "user.logout", "logout", "auth", "", "logout ok", "127.0.0.1", "agent", "session-1", "", "req-2", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	server := NewServer(repository.NewAuditRepository(db))
	resp, err := server.RecordAuditEvent(context.Background(), &auditv1.RecordAuditEventRequest{
		OrgId:        orgID.String(),
		UserId:       userID.String(),
		Username:     "admin",
		EventType:    "user.logout",
		Action:       "logout",
		ResourceType: "auth",
		Detail:       "logout ok",
		Ip:           "127.0.0.1",
		UserAgent:    "agent",
		SessionId:    "session-1",
		RequestId:    "req-2",
	})
	if err != nil {
		t.Fatalf("RecordAuditEvent: %v", err)
	}
	if resp.Id == "" {
		t.Fatalf("expected generated audit id")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestServerListEventTypes(t *testing.T) {
	server := NewServer(nil)
	resp, err := server.ListEventTypes(context.Background(), &auditv1.ListEventTypesRequest{})
	if err != nil {
		t.Fatalf("ListEventTypes: %v", err)
	}
	if len(resp.EventTypes) == 0 {
		t.Fatalf("expected event types")
	}
}
