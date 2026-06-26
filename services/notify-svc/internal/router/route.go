package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ops-platform/notify-svc/internal/handler"
)

func New(h *handler.NotifyHandler) *gin.Engine {
	r := gin.Default()
	Register(r, h)
	return r
}

func Register(r gin.IRouter, h *handler.NotifyHandler) {
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/api/v1/notifications", h.ListChannels)
	r.POST("/api/v1/notifications", h.CreateChannel)
	r.PUT("/api/v1/notifications/:id", h.UpdateChannel)
	r.DELETE("/api/v1/notifications/:id", h.DeleteChannel)
	r.GET("/api/v1/notify/templates", h.ListTemplates)
	r.POST("/api/v1/notify/templates", h.CreateTemplate)
	r.PUT("/api/v1/notify/templates/:id", h.UpdateTemplate)
	r.DELETE("/api/v1/notify/templates/:id", h.DeleteTemplate)
	r.GET("/api/v1/notify/logs", h.ListNotificationLogs)
}
