package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type BootstrapHandler struct {
	serviceName string
}

func NewBootstrapHandler(serviceName string) *BootstrapHandler {
	return &BootstrapHandler{serviceName: serviceName}
}

func (h *BootstrapHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": h.serviceName})
}

func (h *BootstrapHandler) Bootstrap(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": h.serviceName,
		"architecture": gin.H{
			"edge":        "apisix",
			"client_api":  "bff-api",
			"domain_call": "grpc",
			"event_bus":   "kafka",
		},
		"modules": []gin.H{
			{"name": "auth-svc", "phase": 1, "status": "transition-http"},
			{"name": "iam-svc", "phase": 1, "status": "grpc-ready"},
			{"name": "audit-svc", "phase": 1, "status": "transition-http"},
			{"name": "notify-svc", "phase": 1, "status": "transition-http"},
			{"name": "cmdb-svc", "phase": 2, "status": "planned"},
			{"name": "monitor-svc", "phase": 3, "status": "planned"},
			{"name": "ticket-svc", "phase": 4, "status": "planned"},
			{"name": "deploy-svc", "phase": 5, "status": "planned"},
			{"name": "automation-svc", "phase": 6, "status": "planned"},
		},
	})
}
