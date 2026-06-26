package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/ops-platform/bff-api/internal/client"
	"github.com/ops-platform/pkg/response"
)

type AuditService interface {
	ListLogs(ctx context.Context, query client.AuditLogQuery, userCtx client.UserContext) ([]map[string]any, int64, error)
	ListEventTypes(ctx context.Context, userCtx client.UserContext) ([]string, error)
}

type AuditHandler struct {
	service AuditService
}

func NewAuditHandler(service AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

func (h *AuditHandler) Register(r gin.IRouter) {
	r.GET("", h.ListLogs)
	r.GET("/event-types", h.ListEventTypes)
}

func (h *AuditHandler) ListLogs(c *gin.Context) {
	logs, total, err := h.service.ListLogs(c.Request.Context(), client.AuditLogQuery{
		OrgID:     c.Query("org_id"),
		EventType: c.Query("event_type"),
		StartTime: c.Query("start_time"),
		EndTime:   c.Query("end_time"),
		Page:      queryInt(c, "page", 1),
		PageSize:  queryInt(c, "page_size", 20),
	}, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.SuccessWithTotal(c, logs, total)
}

func (h *AuditHandler) ListEventTypes(c *gin.Context) {
	types, err := h.service.ListEventTypes(c.Request.Context(), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, types)
}
