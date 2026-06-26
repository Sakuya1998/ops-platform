package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ops-platform/notify-svc/internal/handler"
)

func TestNewRegistersNotifyRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := New(new(handler.NotifyHandler))
	routes := registeredRoutes(r.Routes())

	expected := []string{
		"GET /health",
		"GET /api/v1/notifications",
		"POST /api/v1/notifications",
		"PUT /api/v1/notifications/:id",
		"DELETE /api/v1/notifications/:id",
		"GET /api/v1/notify/templates",
		"POST /api/v1/notify/templates",
		"PUT /api/v1/notify/templates/:id",
		"DELETE /api/v1/notify/templates/:id",
		"GET /api/v1/notify/logs",
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
