package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/client"
	"github.com/Sakuya1998/ops-platform/pkg/response"
)

type NotifyService interface {
	ListChannels(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error)
	CreateChannel(ctx context.Context, req client.NotificationChannelRequest, userCtx client.UserContext) (map[string]any, error)
	UpdateChannel(ctx context.Context, id string, req client.NotificationChannelRequest, userCtx client.UserContext) (map[string]any, error)
	DeleteChannel(ctx context.Context, id string, userCtx client.UserContext) error
	ListTemplates(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error)
	CreateTemplate(ctx context.Context, req client.NotificationTemplateRequest, userCtx client.UserContext) (map[string]any, error)
	UpdateTemplate(ctx context.Context, id string, req client.NotificationTemplateRequest, userCtx client.UserContext) (map[string]any, error)
	DeleteTemplate(ctx context.Context, id string, userCtx client.UserContext) error
	ListNotificationLogs(ctx context.Context, query client.NotificationLogQuery, userCtx client.UserContext) ([]map[string]any, int64, error)
}

type NotifyHandler struct {
	service NotifyService
}

func NewNotifyHandler(service NotifyService) *NotifyHandler {
	return &NotifyHandler{service: service}
}

func (h *NotifyHandler) RegisterChannels(r gin.IRouter) {
	r.GET("", h.ListChannels)
	r.POST("", h.CreateChannel)
	r.PUT("/:id", h.UpdateChannel)
	r.DELETE("/:id", h.DeleteChannel)
}

func (h *NotifyHandler) RegisterTemplates(r gin.IRouter) {
	r.GET("/templates", h.ListTemplates)
	r.POST("/templates", h.CreateTemplate)
	r.PUT("/templates/:id", h.UpdateTemplate)
	r.DELETE("/templates/:id", h.DeleteTemplate)
	r.GET("/logs", h.ListNotificationLogs)
}

func (h *NotifyHandler) ListChannels(c *gin.Context) {
	result, err := h.service.ListChannels(c.Request.Context(), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *NotifyHandler) CreateChannel(c *gin.Context) {
	var req client.NotificationChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.CreateChannel(c.Request.Context(), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, result)
}

func (h *NotifyHandler) UpdateChannel(c *gin.Context) {
	var req client.NotificationChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.UpdateChannel(c.Request.Context(), c.Param("id"), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *NotifyHandler) DeleteChannel(c *gin.Context) {
	if err := h.service.DeleteChannel(c.Request.Context(), c.Param("id"), userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *NotifyHandler) ListTemplates(c *gin.Context) {
	result, err := h.service.ListTemplates(c.Request.Context(), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *NotifyHandler) CreateTemplate(c *gin.Context) {
	var req client.NotificationTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.CreateTemplate(c.Request.Context(), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, result)
}

func (h *NotifyHandler) UpdateTemplate(c *gin.Context) {
	var req client.NotificationTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.UpdateTemplate(c.Request.Context(), c.Param("id"), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *NotifyHandler) DeleteTemplate(c *gin.Context) {
	if err := h.service.DeleteTemplate(c.Request.Context(), c.Param("id"), userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *NotifyHandler) ListNotificationLogs(c *gin.Context) {
	logs, total, err := h.service.ListNotificationLogs(c.Request.Context(), client.NotificationLogQuery{
		EventType: c.Query("event_type"),
		Status:    c.Query("status"),
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
