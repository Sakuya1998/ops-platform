package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/audit-svc/internal/handler"
)

func New(h *handler.AuditHandler) *gin.Engine {
	r := gin.Default()
	Register(r, h)
	return r
}

func Register(r gin.IRouter, h *handler.AuditHandler) {
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/api/v1/audit-logs", h.ListLogs)
	r.GET("/api/v1/audit-logs/event-types", h.ListEventTypes)
}
