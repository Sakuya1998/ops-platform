package client

import (
	"context"
	"encoding/json"

	notifyv1 "github.com/ops-platform/pkg/proto/notify/v1"
	"google.golang.org/grpc"
)

type NotifyGRPCClient struct {
	client notifyv1.NotifyServiceClient
}

func NewNotifyGRPCClient(conn grpc.ClientConnInterface) *NotifyGRPCClient {
	return &NotifyGRPCClient{client: notifyv1.NewNotifyServiceClient(conn)}
}

func (c *NotifyGRPCClient) ListChannels(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	resp, err := c.client.ListChannels(ctx, &notifyv1.ListChannelsRequest{OrgId: userCtx.OrgID})
	if err != nil {
		return nil, err
	}
	channels := make([]map[string]any, 0, len(resp.Channels))
	for _, channel := range resp.Channels {
		channels = append(channels, channelToMap(channel))
	}
	return channels, nil
}

func (c *NotifyGRPCClient) CreateChannel(ctx context.Context, req NotificationChannelRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.CreateChannel(ctx, &notifyv1.CreateChannelRequest{
		OrgId:       firstNonEmpty(req.OrgID, userCtx.OrgID),
		Name:        req.Name,
		ChannelType: req.ChannelType,
		ConfigJson:  mustJSON(req.Config),
		IsEnabled:   req.IsEnabled == nil || *req.IsEnabled,
	})
	if err != nil {
		return nil, err
	}
	return channelToMap(resp), nil
}

func (c *NotifyGRPCClient) UpdateChannel(ctx context.Context, id string, req NotificationChannelRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.UpdateChannel(ctx, &notifyv1.UpdateChannelRequest{
		Id:          id,
		Name:        req.Name,
		ChannelType: req.ChannelType,
		ConfigJson:  mustJSON(req.Config),
		IsEnabled:   req.IsEnabled == nil || *req.IsEnabled,
	})
	if err != nil {
		return nil, err
	}
	return channelToMap(resp), nil
}

func (c *NotifyGRPCClient) DeleteChannel(ctx context.Context, id string, userCtx UserContext) error {
	_, err := c.client.DeleteChannel(ctx, &notifyv1.DeleteChannelRequest{Id: id})
	return err
}

func (c *NotifyGRPCClient) ListTemplates(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	resp, err := c.client.ListTemplates(ctx, &notifyv1.ListTemplatesRequest{})
	if err != nil {
		return nil, err
	}
	templates := make([]map[string]any, 0, len(resp.Templates))
	for _, template := range resp.Templates {
		templates = append(templates, templateToMap(template))
	}
	return templates, nil
}

func (c *NotifyGRPCClient) CreateTemplate(ctx context.Context, req NotificationTemplateRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.CreateTemplate(ctx, &notifyv1.CreateTemplateRequest{
		ChannelType: req.ChannelType, Name: req.Name, TitleTemplate: req.TitleTemplate, BodyTemplate: req.BodyTemplate,
	})
	if err != nil {
		return nil, err
	}
	return templateToMap(resp), nil
}

func (c *NotifyGRPCClient) UpdateTemplate(ctx context.Context, id string, req NotificationTemplateRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.UpdateTemplate(ctx, &notifyv1.UpdateTemplateRequest{
		Id: id, ChannelType: req.ChannelType, Name: req.Name, TitleTemplate: req.TitleTemplate, BodyTemplate: req.BodyTemplate,
	})
	if err != nil {
		return nil, err
	}
	return templateToMap(resp), nil
}

func (c *NotifyGRPCClient) DeleteTemplate(ctx context.Context, id string, userCtx UserContext) error {
	_, err := c.client.DeleteTemplate(ctx, &notifyv1.DeleteTemplateRequest{Id: id})
	return err
}

func (c *NotifyGRPCClient) ListNotificationLogs(ctx context.Context, query NotificationLogQuery, userCtx UserContext) ([]map[string]any, int64, error) {
	resp, err := c.client.ListNotificationLogs(ctx, &notifyv1.ListNotificationLogsRequest{
		EventType: query.EventType,
		Status:    query.Status,
		StartTime: query.StartTime,
		EndTime:   query.EndTime,
		Page:      int32(query.Page),
		PageSize:  int32(query.PageSize),
	})
	if err != nil {
		return nil, 0, err
	}
	logs := make([]map[string]any, 0, len(resp.Logs))
	for _, log := range resp.Logs {
		logs = append(logs, notificationLogToMap(log))
	}
	return logs, resp.Total, nil
}

func channelToMap(channel *notifyv1.NotificationChannel) map[string]any {
	if channel == nil {
		return map[string]any{}
	}
	result := map[string]any{
		"id":           channel.Id,
		"org_id":       channel.OrgId,
		"name":         channel.Name,
		"channel_type": channel.ChannelType,
		"config":       jsonToMap(channel.ConfigJson),
		"is_enabled":   channel.IsEnabled,
	}
	if channel.CreatedAt != nil {
		result["created_at"] = channel.CreatedAt.AsTime()
	}
	if channel.UpdatedAt != nil {
		result["updated_at"] = channel.UpdatedAt.AsTime()
	}
	return result
}

func templateToMap(template *notifyv1.NotificationTemplate) map[string]any {
	if template == nil {
		return map[string]any{}
	}
	result := map[string]any{
		"id":             template.Id,
		"channel_type":   template.ChannelType,
		"name":           template.Name,
		"title_template": template.TitleTemplate,
		"body_template":  template.BodyTemplate,
	}
	if template.CreatedAt != nil {
		result["created_at"] = template.CreatedAt.AsTime()
	}
	return result
}

func notificationLogToMap(log *notifyv1.NotificationLog) map[string]any {
	if log == nil {
		return map[string]any{}
	}
	result := map[string]any{
		"id":         log.Id,
		"channel_id": log.ChannelId,
		"event_type": log.EventType,
		"recipient":  log.Recipient,
		"title":      log.Title,
		"status":     log.Status,
		"error_msg":  log.ErrorMsg,
	}
	if log.CreatedAt != nil {
		result["created_at"] = log.CreatedAt.AsTime()
	}
	return result
}

func mustJSON(value map[string]any) string {
	if value == nil {
		return "{}"
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func jsonToMap(value string) map[string]any {
	if value == "" {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return map[string]any{}
	}
	return out
}
