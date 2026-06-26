package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/audit-svc/internal/handler"
)

func TestNewRegistersAuditRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := New(new(handler.AuditHandler))
	routes := registeredRoutes(r.Routes())

	expected := []string{
		"GET /health",
		"GET /api/v1/audit-logs",
		"GET /api/v1/audit-logs/event-types",
	}
	for _, route := range expected {
		if !routes[route] {
			t.Fatalf("expected route %s to be registered", route)
		}
	}
}

func registeredRoutes(infos gin.RoutesInfo) map[string]bool {
	routes := make(map[string]bool, len(infos))
	for _, info := range infos {
		routes[info.Method+" "+info.Path] = true
	}
	return routes
}
