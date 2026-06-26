package grpc

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ops-platform/audit-svc/internal/model"
	"github.com/ops-platform/audit-svc/internal/repository"
	auditv1 "github.com/ops-platform/pkg/proto/audit/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	auditv1.UnimplementedAuditServiceServer
	repo *repository.AuditRepository
}

func NewServer(repo *repository.AuditRepository) *Server {
	return &Server{repo: repo}
}

func (s *Server) ListAuditLogs(ctx context.Context, req *auditv1.ListAuditLogsRequest) (*auditv1.ListAuditLogsResponse, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	logs, total, err := s.repo.List(req.OrgId, req.EventType, req.StartTime, req.EndTime, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, err
	}
	resp := &auditv1.ListAuditLogsResponse{Logs: make([]*auditv1.AuditLog, 0, len(logs)), Total: total}
	for _, log := range logs {
		resp.Logs = append(resp.Logs, mapAuditLog(log))
	}
	return resp, nil
}

func (s *Server) ListEventTypes(ctx context.Context, req *auditv1.ListEventTypesRequest) (*auditv1.ListEventTypesResponse, error) {
	return &auditv1.ListEventTypesResponse{EventTypes: phaseOneEventTypes()}, nil
}

func (s *Server) RecordAuditEvent(ctx context.Context, req *auditv1.RecordAuditEventRequest) (*auditv1.RecordAuditEventResponse, error) {
	log := &model.AuditLog{
		OrgID:        parseUUID(req.OrgId),
		UserID:       parseUUID(req.UserId),
		Username:     req.Username,
		EventType:    req.EventType,
		Action:       req.Action,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceId,
		Detail:       req.Detail,
		IP:           req.Ip,
		UserAgent:    req.UserAgent,
		SessionID:    req.SessionId,
		ReasonCode:   req.ReasonCode,
		RequestID:    req.RequestId,
		CreatedAt:    time.Now(),
	}
	if req.OccurredAt != nil {
		log.CreatedAt = req.OccurredAt.AsTime()
	}
	if err := s.repo.Create(log); err != nil {
		return nil, err
	}
	return &auditv1.RecordAuditEventResponse{Id: log.ID.String()}, nil
}

func mapAuditLog(log model.AuditLog) *auditv1.AuditLog {
	return &auditv1.AuditLog{
		Id:           log.ID.String(),
		OrgId:        log.OrgID.String(),
		UserId:       log.UserID.String(),
		Username:     log.Username,
		EventType:    log.EventType,
		Action:       log.Action,
		ResourceType: log.ResourceType,
		ResourceId:   log.ResourceID,
		Detail:       log.Detail,
		Ip:           log.IP,
		UserAgent:    log.UserAgent,
		SessionId:    log.SessionID,
		ReasonCode:   log.ReasonCode,
		RequestId:    log.RequestID,
		CreatedAt:    timestamppb.New(log.CreatedAt),
	}
}

func normalizePage(page, pageSize int32) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return int(page), int(pageSize)
}

func parseUUID(value string) uuid.UUID {
	if value == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func phaseOneEventTypes() []string {
	return []string{
		"user.login", "user.login_failed", "user.login_limited", "user.logout",
		"auth.refresh_reuse",
		"user.created", "user.updated", "user.deleted", "user.role_changed",
		"user.password_changed", "user.password_reset",
		"user.mfa_enabled", "user.mfa_disabled", "user.mfa_recovery_codes_rotated",
		"role.created", "role.updated", "role.deleted", "role.permission_changed",
		"org.created", "org.updated",
		"system.config_updated",
	}
}
