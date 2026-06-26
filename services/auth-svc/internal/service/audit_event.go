package service

import (
	"context"
	"time"

	"github.com/ops-platform/pkg/kafka"
)

type eventPublisher interface {
	PublishEvent(ctx context.Context, event kafka.Event) error
}

type AuditEvent struct {
	EventType    string
	UserID       string
	OrgID        string
	Username     string
	Action       string
	ResourceType string
	ResourceID   string
	Detail       string
	IP           string
	UserAgent    string
	SessionID    string
	ReasonCode   string
	RequestID    string
}

func (s *AuthService) publishEvent(eventType, userID, orgID, username, action, detail string) {
	s.publishAuditEvent(context.Background(), AuditEvent{
		EventType: eventType, UserID: userID, OrgID: orgID, Username: username,
		Action: action, ResourceType: "auth", Detail: detail,
	})
}

func (s *AuthService) publishLoginFailure(ctx context.Context, userID, orgID, username, provider, detail, reasonCode string, loginCtx LoginContext) {
	s.publishAuditEvent(ctx, AuditEvent{
		EventType:    "user.login_failed",
		UserID:       userID,
		OrgID:        orgID,
		Username:     username,
		Action:       "login",
		ResourceType: "auth",
		Detail:       detail,
		IP:           loginCtx.IP,
		UserAgent:    loginCtx.UserAgent,
		ReasonCode:   reasonCode,
		RequestID:    loginCtx.RequestID,
	})
}

func (s *AuthService) publishAuditEvent(ctx context.Context, event AuditEvent) {
	if s.kafkaProd == nil {
		return
	}
	if event.ResourceType == "" {
		event.ResourceType = "auth"
	}
	_ = s.kafkaProd.PublishEvent(ctx, kafka.Event{
		EventType:    event.EventType,
		UserID:       event.UserID,
		OrgID:        event.OrgID,
		Username:     event.Username,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   event.ResourceID,
		Detail:       event.Detail,
		IP:           event.IP,
		UserAgent:    event.UserAgent,
		SessionID:    event.SessionID,
		ReasonCode:   event.ReasonCode,
		RequestID:    event.RequestID,
		Timestamp:    time.Now().Format(time.RFC3339),
	})
}
