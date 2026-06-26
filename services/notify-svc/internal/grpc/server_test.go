package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/service"
	notifyv1 "github.com/Sakuya1998/ops-platform/pkg/proto/notify/v1"
	"gorm.io/gorm"
)

func TestServerListChannels(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	channelID := uuid.MustParse("00000000-0000-0000-0000-000000000101")
	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{"id", "org_id", "name", "channel_type", "config", "is_enabled", "created_at", "updated_at"}).
		AddRow(channelID, orgID, "Ops Webhook", "webhook", `{"url":"http://example"}`, true, now, now)
	mock.ExpectQuery(`SELECT \* FROM "notification_channels" WHERE org_id = \$1 ORDER BY created_at DESC`).
		WithArgs(orgID).
		WillReturnRows(rows)

	server := newTestServer(db)
	resp, err := server.ListChannels(context.Background(), &notifyv1.ListChannelsRequest{OrgId: orgID.String()})
	if err != nil {
		t.Fatalf("ListChannels: %v", err)
	}
	if len(resp.Channels) != 1 {
		t.Fatalf("expected one channel, got %d", len(resp.Channels))
	}
	got := resp.Channels[0]
	if got.Id != channelID.String() || got.OrgId != orgID.String() || got.ChannelType != "webhook" || !got.IsEnabled || got.CreatedAt == nil {
		t.Fatalf("unexpected channel: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestServerCreateChannel(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	mock.ExpectExec(`INSERT INTO "notification_channels"`).
		WithArgs(sqlmock.AnyArg(), orgID, "Ops Webhook", "webhook", `{"url":"http://example"}`, true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	server := newTestServer(db)
	resp, err := server.CreateChannel(context.Background(), &notifyv1.CreateChannelRequest{
		OrgId:       orgID.String(),
		Name:        "Ops Webhook",
		ChannelType: "webhook",
		ConfigJson:  `{"url":"http://example"}`,
		IsEnabled:   true,
	})
	if err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	if resp.Id == "" || resp.OrgId != orgID.String() || resp.Name != "Ops Webhook" {
		t.Fatalf("unexpected channel: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestServerListTemplatesAndLogs(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	templateID := uuid.MustParse("00000000-0000-0000-0000-000000000201")
	logID := uuid.MustParse("00000000-0000-0000-0000-000000000301")
	now := time.Now().UTC()
	templateRows := sqlmock.NewRows([]string{"id", "channel_type", "name", "title_template", "body_template", "created_at"}).
		AddRow(templateID, "email", "Login Notice", "title", "body", now)
	mock.ExpectQuery(`SELECT \* FROM "notification_templates" ORDER BY created_at DESC`).
		WillReturnRows(templateRows)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "notification_logs" WHERE event_type = \$1 AND status = \$2`).
		WithArgs("user.login", "success").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	logRows := sqlmock.NewRows([]string{"id", "channel_id", "event_type", "recipient", "title", "status", "error_msg", "created_at"}).
		AddRow(logID, nil, "user.login", "ops@example.com", "Login", "success", "", now)
	mock.ExpectQuery(`SELECT \* FROM "notification_logs" WHERE event_type = \$1 AND status = \$2 ORDER BY created_at DESC LIMIT \$3`).
		WithArgs("user.login", "success", 20).
		WillReturnRows(logRows)

	server := newTestServer(db)
	templates, err := server.ListTemplates(context.Background(), &notifyv1.ListTemplatesRequest{})
	if err != nil {
		t.Fatalf("ListTemplates: %v", err)
	}
	if len(templates.Templates) != 1 || templates.Templates[0].Id != templateID.String() {
		t.Fatalf("unexpected templates: %+v", templates)
	}
	logs, err := server.ListNotificationLogs(context.Background(), &notifyv1.ListNotificationLogsRequest{
		EventType: "user.login",
		Status:    "success",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("ListNotificationLogs: %v", err)
	}
	if logs.Total != 1 || len(logs.Logs) != 1 || logs.Logs[0].Id != logID.String() {
		t.Fatalf("unexpected logs: %+v", logs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestServerSendNotificationRecordsFailure(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	db := openMockDB(t, sqlDB)

	mock.ExpectExec(`INSERT INTO "notification_logs"`).
		WithArgs(sqlmock.AnyArg(), nil, "manual.test", "ops@example.com", "Subject", "failed", "unsupported channel type: sms", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	server := newTestServer(db)
	resp, err := server.SendNotification(context.Background(), &notifyv1.SendNotificationRequest{
		ChannelType: "sms",
		EventType:   "manual.test",
		Recipient:   "ops@example.com",
		Title:       "Subject",
		Body:        "Body",
	})
	if err != nil {
		t.Fatalf("SendNotification should return response with failed status, got error: %v", err)
	}
	if resp.Status != "failed" || resp.ErrorMsg == "" || resp.LogId == "" {
		t.Fatalf("unexpected send response: %+v", resp)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func newTestServer(db *gorm.DB) *Server {
	channelRepo := repository.NewChannelRepository(db)
	tmplRepo := repository.NewTemplateRepository(db)
	logRepo := repository.NewLogRepository(db)
	return NewServer(channelRepo, tmplRepo, logRepo, service.NewNotifyService(channelRepo, tmplRepo, logRepo))
}
