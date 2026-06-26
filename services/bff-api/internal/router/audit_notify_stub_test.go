package router

import (
	"context"

	"github.com/ops-platform/bff-api/internal/client"
)

type recordingAuditRouterService struct {
	query   client.AuditLogQuery
	userCtx client.UserContext
}

func (s *recordingAuditRouterService) ListLogs(ctx context.Context, query client.AuditLogQuery, userCtx client.UserContext) ([]map[string]any, int64, error) {
	s.query = query
	s.userCtx = userCtx
	return []map[string]any{{"id": "a_1", "event_type": "user.login"}}, 1, nil
}

func (s *recordingAuditRouterService) ListEventTypes(ctx context.Context, userCtx client.UserContext) ([]string, error) {
	s.userCtx = userCtx
	return []string{"user.login"}, nil
}

type recordingNotifyRouterService struct {
	channelReq  client.NotificationChannelRequest
	templateReq client.NotificationTemplateRequest
	logQuery    client.NotificationLogQuery
	userCtx     client.UserContext
	id          string
}

func (s *recordingNotifyRouterService) ListChannels(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userCtx = userCtx
	return []map[string]any{{"id": "ch_1"}}, nil
}

func (s *recordingNotifyRouterService) CreateChannel(ctx context.Context, req client.NotificationChannelRequest, userCtx client.UserContext) (map[string]any, error) {
	s.channelReq = req
	s.userCtx = userCtx
	return map[string]any{"id": "ch_2", "name": req.Name}, nil
}

func (s *recordingNotifyRouterService) UpdateChannel(ctx context.Context, id string, req client.NotificationChannelRequest, userCtx client.UserContext) (map[string]any, error) {
	s.id = id
	s.channelReq = req
	s.userCtx = userCtx
	return map[string]any{"id": id, "name": req.Name}, nil
}

func (s *recordingNotifyRouterService) DeleteChannel(ctx context.Context, id string, userCtx client.UserContext) error {
	s.id = id
	s.userCtx = userCtx
	return nil
}

func (s *recordingNotifyRouterService) ListTemplates(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userCtx = userCtx
	return []map[string]any{{"id": "tpl_1"}}, nil
}

func (s *recordingNotifyRouterService) CreateTemplate(ctx context.Context, req client.NotificationTemplateRequest, userCtx client.UserContext) (map[string]any, error) {
	s.templateReq = req
	s.userCtx = userCtx
	return map[string]any{"id": "tpl_2", "name": req.Name}, nil
}

func (s *recordingNotifyRouterService) UpdateTemplate(ctx context.Context, id string, req client.NotificationTemplateRequest, userCtx client.UserContext) (map[string]any, error) {
	s.id = id
	s.templateReq = req
	s.userCtx = userCtx
	return map[string]any{"id": id, "name": req.Name}, nil
}

func (s *recordingNotifyRouterService) DeleteTemplate(ctx context.Context, id string, userCtx client.UserContext) error {
	s.id = id
	s.userCtx = userCtx
	return nil
}

func (s *recordingNotifyRouterService) ListNotificationLogs(ctx context.Context, query client.NotificationLogQuery, userCtx client.UserContext) ([]map[string]any, int64, error) {
	s.logQuery = query
	s.userCtx = userCtx
	return []map[string]any{{"id": "log_1", "status": query.Status}}, 1, nil
}
