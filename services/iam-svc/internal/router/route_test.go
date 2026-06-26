package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/handler"
)

func TestNewRegistersIAMRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := New(new(handler.IAMHandler))
	routes := registeredRoutes(r.Routes())

	expected := []string{
		"GET /health",
		"POST /internal/v1/check-permission",
		"POST /internal/v1/permissions/check",
		"POST /internal/v1/permissions/batch-check",
		"GET /internal/v1/users/:id/roles",
		"PUT /api/v1/users/:id/roles",
		"GET /api/v1/users/:id/roles",
		"GET /api/v1/users/:id/permissions",
		"GET /api/v1/roles",
		"POST /api/v1/roles",
		"GET /api/v1/roles/:id",
		"PUT /api/v1/roles/:id",
		"DELETE /api/v1/roles/:id",
		"PUT /api/v1/roles/:id/permissions",
		"GET /api/v1/permissions",
		"GET /api/v1/api-permissions",
		"POST /api/v1/api-permissions",
		"PUT /api/v1/api-permissions/:id",
		"DELETE /api/v1/api-permissions/:id",
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
