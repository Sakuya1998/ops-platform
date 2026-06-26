package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/client"
)

type fakeTokenVerifier struct {
	result client.TokenContext
	err    error
	calls  int
	got    string
}

func (f *fakeTokenVerifier) VerifyToken(ctx context.Context, authorization string) (client.TokenContext, error) {
	f.calls++
	f.got = authorization
	return f.result, f.err
}

func TestRequireUserContextKeepsExistingGatewayContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := &fakeTokenVerifier{}
	r := gin.New()
	r.GET("/protected", RequireUserContext(verifier), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":    c.GetHeader("X-User-Id"),
			"org_id":     c.GetHeader("X-Org-Id"),
			"session_id": c.GetHeader("X-Session-Id"),
		})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	req.Header.Set("X-Session-Id", "s_1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if verifier.calls != 0 {
		t.Fatalf("expected verifier not called, got %d", verifier.calls)
	}
}

func TestRequireUserContextResolvesBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	verifier := &fakeTokenVerifier{result: client.TokenContext{
		Active: true, UserID: "u_1", OrgID: "org_1", SessionID: "s_1",
	}}
	r := gin.New()
	r.GET("/protected", RequireUserContext(verifier), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id":    c.GetHeader("X-User-Id"),
			"org_id":     c.GetHeader("X-Org-Id"),
			"session_id": c.GetHeader("X-Session-Id"),
		})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if verifier.calls != 1 || verifier.got != "Bearer token" {
		t.Fatalf("unexpected verifier call count=%d auth=%q", verifier.calls, verifier.got)
	}
	for _, key := range []string{"X-User-Id", "X-Org-Id", "X-Session-Id"} {
		if req.Header.Get(key) == "" {
			t.Fatalf("expected request header %s to be injected", key)
		}
	}
}

func TestRequireUserContextRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", RequireUserContext(&fakeTokenVerifier{}), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
