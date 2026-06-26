package client

import (
	"context"

	auditv1 "github.com/ops-platform/pkg/proto/audit/v1"
	"google.golang.org/grpc"
)

type AuditGRPCClient struct {
	client auditv1.AuditServiceClient
}

func NewAuditGRPCClient(conn grpc.ClientConnInterface) *AuditGRPCClient {
	return &AuditGRPCClient{client: auditv1.NewAuditServiceClient(conn)}
}

func (c *AuditGRPCClient) ListLogs(ctx context.Context, query AuditLogQuery, userCtx UserContext) ([]map[string]any, int64, error) {
	req := &auditv1.ListAuditLogsRequest{
		OrgId:     firstNonEmpty(query.OrgID, userCtx.OrgID),
		EventType: query.EventType,
		StartTime: query.StartTime,
		EndTime:   query.EndTime,
		Page:      int32(query.Page),
		PageSize:  int32(query.PageSize),
	}
	resp, err := c.client.ListAuditLogs(ctx, req)
	if err != nil {
		return nil, 0, err
	}
	logs := make([]map[string]any, 0, len(resp.Logs))
	for _, log := range resp.Logs {
		logs = append(logs, auditLogToMap(log))
	}
	return logs, resp.Total, nil
}

func (c *AuditGRPCClient) ListEventTypes(ctx context.Context, userCtx UserContext) ([]string, error) {
	resp, err := c.client.ListEventTypes(ctx, &auditv1.ListEventTypesRequest{})
	if err != nil {
		return nil, err
	}
	return resp.EventTypes, nil
}

func auditLogToMap(log *auditv1.AuditLog) map[string]any {
	if log == nil {
		return map[string]any{}
	}
	result := map[string]any{
		"id":            log.Id,
		"org_id":        log.OrgId,
		"user_id":       log.UserId,
		"username":      log.Username,
		"event_type":    log.EventType,
		"action":        log.Action,
		"resource_type": log.ResourceType,
		"resource_id":   log.ResourceId,
		"detail":        log.Detail,
		"ip":            log.Ip,
		"user_agent":    log.UserAgent,
		"session_id":    log.SessionId,
		"reason_code":   log.ReasonCode,
		"request_id":    log.RequestId,
	}
	if log.CreatedAt != nil {
		result["created_at"] = log.CreatedAt.AsTime()
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
