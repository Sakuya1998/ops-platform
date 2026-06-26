package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ops-platform/pkg/kafka"
)

func TestAuditLogFromEventMapsSecurityContext(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	ts := time.Date(2026, 6, 23, 15, 0, 0, 0, time.UTC)

	log := auditLogFromEvent(kafka.Event{
		EventType:    "user.login_failed",
		UserID:       userID.String(),
		OrgID:        orgID.String(),
		Username:     "admin",
		Action:       "login",
		ResourceType: "auth",
		Detail:       "Wrong password",
		IP:           "10.0.0.1",
		UserAgent:    "test-agent",
		SessionID:    "session-1",
		ReasonCode:   "invalid_credentials",
		RequestID:    "request-1",
		Timestamp:    ts.Format(time.RFC3339),
	})

	if log.OrgID != orgID || log.UserID != userID {
		t.Fatalf("unexpected ids: org=%s user=%s", log.OrgID, log.UserID)
	}
	if log.IP != "10.0.0.1" || log.UserAgent != "test-agent" {
		t.Fatalf("unexpected client context: ip=%q ua=%q", log.IP, log.UserAgent)
	}
	if log.SessionID != "session-1" || log.ReasonCode != "invalid_credentials" || log.RequestID != "request-1" {
		t.Fatalf("unexpected security context: session=%q reason=%q request=%q", log.SessionID, log.ReasonCode, log.RequestID)
	}
	if !log.CreatedAt.Equal(ts) {
		t.Fatalf("expected timestamp %s, got %s", ts, log.CreatedAt)
	}
}
