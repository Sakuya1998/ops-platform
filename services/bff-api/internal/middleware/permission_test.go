package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/client"
)

type fakePermissionChecker struct {
	result client.CheckPermissionResult
	err    error
	got    client.CheckPermissionRequest
}

func (f *fakePermissionChecker) CheckPermission(ctx context.Context, req client.CheckPermissionRequest) (client.CheckPermissionResult, error) {
	f.got = req
	return f.result, f.err
}

func TestRequirePermissionAllowsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	checker := &fakePermissionChecker{result: client.CheckPermissionResult{Allowed: true, Roles: []string{"admin"}}}
	r := gin.New()
	r.GET("/api/v1/users", RequirePermission(checker), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if checker.got.UserID != "u_1" || checker.got.OrgID != "org_1" ||
		checker.got.Method != http.MethodGet || checker.got.Path != "/api/v1/users" {
		t.Fatalf("unexpected permission request: %+v", checker.got)
	}
}

func TestRequirePermissionRejectsMissingIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/users", RequirePermission(&fakePermissionChecker{}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRequirePermissionRejectsDeniedDecision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	checker := &fakePermissionChecker{result: client.CheckPermissionResult{Allowed: false, Reason: "missing permission"}}
	r := gin.New()
	r.DELETE("/api/v1/users/:id", RequirePermission(checker), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/u_2", nil)
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
