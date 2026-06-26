package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ops-platform/auth-svc/internal/handler"
)

func TestNewRegistersAuthRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := handler.NewAuthHandler(nil, nil)
	r := New(h)

	want := map[string]string{
		"GET /health":                             "",
		"POST /api/v1/auth/login":                 "",
		"POST /api/v1/auth/logout":                "",
		"POST /api/v1/auth/refresh":               "",
		"POST /api/v1/auth/token/verify":          "",
		"GET /api/v1/auth/me":                     "",
		"PUT /api/v1/auth/me/password":            "",
		"POST /api/v1/auth/me/mfa/setup":          "",
		"POST /api/v1/auth/me/mfa/confirm":        "",
		"POST /api/v1/auth/me/mfa/recovery-codes": "",
		"DELETE /api/v1/auth/me/mfa":              "",
		"GET /api/v1/auth/sessions":               "",
		"DELETE /api/v1/auth/sessions/:id":        "",
		"DELETE /api/v1/auth/sessions":            "",
		"GET /api/v1/auth/oidc/login":             "",
		"GET /api/v1/auth/oidc/callback":          "",
		"GET /api/v1/auth/oidc/status":            "",
		"POST /api/v1/auth/oidc/exchange":         "",
		"GET /api/v1/users":                       "",
		"POST /api/v1/users":                      "",
		"GET /api/v1/users/:id":                   "",
		"PUT /api/v1/users/:id":                   "",
		"DELETE /api/v1/users/:id":                "",
		"PUT /api/v1/users/:id/status":            "",
		"PUT /api/v1/users/:id/password/reset":    "",
		"GET /api/v1/organizations":               "",
		"POST /api/v1/organizations":              "",
		"PUT /api/v1/organizations/:id":           "",
		"GET /api/v1/system/config":               "",
		"PUT /api/v1/system/config":               "",
	}

	for _, route := range r.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			delete(want, key)
		}
	}
	for route := range want {
		t.Fatalf("route not registered: %s", route)
	}
}
