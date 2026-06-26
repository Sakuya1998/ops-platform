package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/audit-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/pkg/response"
	"strconv"
)

type AuditHandler struct{ repo *repository.AuditRepository }

func NewAuditHandler(repo *repository.AuditRepository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

func (h *AuditHandler) ListLogs(c *gin.Context) {
	orgID := c.Query("org_id")
	eventType := c.Query("event_type")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	logs, total, err := h.repo.List(orgID, eventType, startTime, endTime, (page-1)*pageSize, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.SuccessWithTotal(c, logs, total)
}

func (h *AuditHandler) ListEventTypes(c *gin.Context) {
	types := []string{
		"user.login", "user.login_failed", "user.login_limited", "user.logout",
		"auth.refresh_reuse",
		"user.created", "user.updated", "user.deleted", "user.role_changed",
		"user.password_changed", "user.password_reset",
		"user.mfa_enabled", "user.mfa_disabled", "user.mfa_recovery_codes_rotated",
		"role.created", "role.updated", "role.deleted", "role.permission_changed",
		"org.created", "org.updated",
		"system.config_updated",
	}
	response.Success(c, types)
}
