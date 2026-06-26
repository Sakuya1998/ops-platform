package handler

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/model"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/pkg/response"
)

type NotifyHandler struct {
	channelRepo *repository.ChannelRepository
	tmplRepo    *repository.TemplateRepository
	logRepo     *repository.LogRepository
}

func NewNotifyHandler(cr *repository.ChannelRepository, tr *repository.TemplateRepository, lr *repository.LogRepository) *NotifyHandler {
	return &NotifyHandler{channelRepo: cr, tmplRepo: tr, logRepo: lr}
}

func (h *NotifyHandler) ListChannels(c *gin.Context) {
	orgID, err := parseOptionalUUID(c.GetHeader("X-Org-Id"))
	if err != nil {
		response.BadRequest(c, "invalid X-Org-Id")
		return
	}
	channels, err := h.channelRepo.ListByOrg(orgID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, channels)
}

func (h *NotifyHandler) CreateChannel(c *gin.Context) {
	var req channelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	orgID, err := parseRequiredUUID(firstNonEmpty(req.OrgID, c.GetHeader("X-Org-Id")), "org_id")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	configJSON, err := normalizeJSON(req.Config)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	channel := &model.NotificationChannel{
		OrgID: orgID, Name: req.Name, ChannelType: req.ChannelType,
		Config: configJSON, IsEnabled: req.IsEnabled == nil || *req.IsEnabled,
	}
	if err := h.channelRepo.Create(channel); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, channel)
}

func (h *NotifyHandler) UpdateChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid channel id")
		return
	}
	channel, err := h.channelRepo.GetByID(id)
	if err != nil {
		response.NotFound(c, "channel not found")
		return
	}
	var req channelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	configJSON, err := normalizeJSON(req.Config)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	channel.Name = req.Name
	channel.ChannelType = req.ChannelType
	channel.Config = configJSON
	if req.IsEnabled != nil {
		channel.IsEnabled = *req.IsEnabled
	}
	if err := h.channelRepo.Update(channel); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, channel)
}

func (h *NotifyHandler) DeleteChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid channel id")
		return
	}
	if err := h.channelRepo.Delete(id); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *NotifyHandler) ListTemplates(c *gin.Context) {
	templates, err := h.tmplRepo.List()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, templates)
}

func (h *NotifyHandler) CreateTemplate(c *gin.Context) {
	var req templateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	tmpl := &model.NotificationTemplate{
		ChannelType: req.ChannelType, Name: req.Name,
		TitleTemplate: req.TitleTemplate, BodyTemplate: req.BodyTemplate,
	}
	if err := h.tmplRepo.Create(tmpl); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, tmpl)
}

func (h *NotifyHandler) UpdateTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid template id")
		return
	}
	tmpl, err := h.tmplRepo.GetByID(id)
	if err != nil {
		response.NotFound(c, "template not found")
		return
	}
	var req templateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	tmpl.ChannelType = req.ChannelType
	tmpl.Name = req.Name
	tmpl.TitleTemplate = req.TitleTemplate
	tmpl.BodyTemplate = req.BodyTemplate
	if err := h.tmplRepo.Update(tmpl); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, tmpl)
}

func (h *NotifyHandler) DeleteTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid template id")
		return
	}
	if err := h.tmplRepo.Delete(id); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *NotifyHandler) ListNotificationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	logs, total, err := h.logRepo.List(
		c.Query("event_type"), c.Query("status"), c.Query("start_time"), c.Query("end_time"),
		(page-1)*pageSize, pageSize,
	)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.SuccessWithTotal(c, logs, total)
}

type channelRequest struct {
	OrgID       string                 `json:"org_id"`
	Name        string                 `json:"name" binding:"required"`
	ChannelType string                 `json:"channel_type" binding:"required,oneof=email dingtalk wechat feishu webhook"`
	Config      map[string]interface{} `json:"config" binding:"required"`
	IsEnabled   *bool                  `json:"is_enabled"`
}

type templateRequest struct {
	ChannelType   string `json:"channel_type" binding:"required,oneof=email dingtalk wechat feishu webhook"`
	Name          string `json:"name" binding:"required"`
	TitleTemplate string `json:"title_template"`
	BodyTemplate  string `json:"body_template" binding:"required"`
}

func parseOptionalUUID(value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(value)
}

func parseRequiredUUID(value, field string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, &fieldError{field: field, message: "is required"}
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, &fieldError{field: field, message: "must be uuid"}
	}
	return id, nil
}

func normalizeJSON(value map[string]interface{}) (string, error) {
	if value == nil {
		return "", &fieldError{field: "config", message: "is required"}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

type fieldError struct {
	field   string
	message string
}

func (e *fieldError) Error() string {
	return e.field + " " + e.message
}
