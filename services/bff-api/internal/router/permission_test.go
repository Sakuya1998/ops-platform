package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/client"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/handler"
)

type recordingPermissionChecker struct {
	result client.CheckPermissionResult
	calls  []client.CheckPermissionRequest
}

func (c *recordingPermissionChecker) CheckPermission(ctx context.Context, req client.CheckPermissionRequest) (client.CheckPermissionResult, error) {
	c.calls = append(c.calls, req)
	return c.result, nil
}

type recordingTokenVerifier struct {
	result client.TokenContext
	calls  int
	got    string
}

func (v *recordingTokenVerifier) VerifyToken(ctx context.Context, authorization string) (client.TokenContext, error) {
	v.calls++
	v.got = authorization
	return v.result, nil
}

func TestPublicRoutesDoNotRequirePermission(t *testing.T) {
	checker := &recordingPermissionChecker{result: client.CheckPermissionResult{Allowed: false}}
	r := New(testDepsWithChecker(t, checker))

	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/health", ""},
		{http.MethodGet, "/api/v1/bootstrap", ""},
		{http.MethodPost, "/api/v1/auth/login", `{"username":"admin"}`},
		{http.MethodPost, "/api/v1/auth/refresh", `{"refresh_token":"rt"}`},
		{http.MethodPost, "/api/v1/auth/token/verify", `{"token":"t"}`},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s %s: expected 200, got %d: %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}

	if len(checker.calls) != 0 {
		t.Fatalf("expected no permission checks for public routes, got %+v", checker.calls)
	}
}

func TestProtectedRoutesRequirePermission(t *testing.T) {
	checker := &recordingPermissionChecker{result: client.CheckPermissionResult{Allowed: true, Roles: []string{"admin"}}}
	r := New(testDepsWithChecker(t, checker))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(checker.calls) != 1 {
		t.Fatalf("expected one permission check, got %+v", checker.calls)
	}
	got := checker.calls[0]
	if got.UserID != "u_1" || got.OrgID != "org_1" || got.Method != http.MethodGet || got.Path != "/api/v1/users" {
		t.Fatalf("unexpected permission request: %+v", got)
	}
}

func TestProtectedRoutesResolveTokenWhenGatewayContextMissing(t *testing.T) {
	checker := &recordingPermissionChecker{result: client.CheckPermissionResult{Allowed: true, Roles: []string{"admin"}}}
	verifier := &recordingTokenVerifier{result: client.TokenContext{Active: true, UserID: "u_1", OrgID: "org_1", SessionID: "s_1"}}
	deps := testDepsWithChecker(t, checker)
	deps.TokenVerifier = verifier
	r := New(deps)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer access-token")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if verifier.calls != 1 || verifier.got != "Bearer access-token" {
		t.Fatalf("unexpected verifier calls=%d auth=%q", verifier.calls, verifier.got)
	}
	if len(checker.calls) != 1 || checker.calls[0].UserID != "u_1" || checker.calls[0].OrgID != "org_1" {
		t.Fatalf("unexpected permission calls: %+v", checker.calls)
	}
}

func TestProtectedRoutesRejectMissingUserContext(t *testing.T) {
	checker := &recordingPermissionChecker{result: client.CheckPermissionResult{Allowed: true}}
	r := New(testDepsWithChecker(t, checker))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
	if len(checker.calls) != 0 {
		t.Fatalf("expected no permission check without user context, got %+v", checker.calls)
	}
}

func TestProtectedRoutesRejectDeniedPermission(t *testing.T) {
	checker := &recordingPermissionChecker{result: client.CheckPermissionResult{Allowed: false, Reason: "missing permission"}}
	r := New(testDepsWithChecker(t, checker))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/roles/r_1", nil)
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func testDepsWithChecker(t *testing.T, checker *recordingPermissionChecker) Dependencies {
	t.Helper()
	return Dependencies{
		Bootstrap:  handler.NewBootstrapHandler("bff-api"),
		Auth:       handler.NewAuthHandler(&recordingAuthHandlerService{}),
		IAM:        handler.NewIAMHandler(&recordingIAMRouterService{}),
		Audit:      handler.NewAuditHandler(&recordingAuditRouterService{}),
		Notify:     handler.NewNotifyHandler(&recordingNotifyRouterService{}),
		Permission: checker,
		TokenVerifier: &recordingTokenVerifier{result: client.TokenContext{
			Active: true, UserID: "u_1", OrgID: "org_1", SessionID: "s_1",
		}},
	}
}
