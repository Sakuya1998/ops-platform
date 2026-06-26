package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ops-platform/bff-api/internal/client"
	"github.com/ops-platform/bff-api/internal/handler"
)

func TestIAMOwnedRoutesUseExplicitHandlers(t *testing.T) {
	iamSvc := &recordingIAMRouterService{}
	r := New(Dependencies{
		Bootstrap:  handler.NewBootstrapHandler("bff-api"),
		Auth:       handler.NewAuthHandler(&recordingAuthHandlerService{}),
		IAM:        handler.NewIAMHandler(iamSvc),
		Audit:      handler.NewAuditHandler(&recordingAuditRouterService{}),
		Notify:     handler.NewNotifyHandler(&recordingNotifyRouterService{}),
		Permission: &recordingPermissionChecker{result: client.CheckPermissionResult{Allowed: true}},
		TokenVerifier: &recordingTokenVerifier{result: client.TokenContext{
			Active: true, UserID: "u_1", OrgID: "org_1", SessionID: "s_1",
		}},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/u_1/roles", strings.NewReader(`{"role_ids":["r_1"]}`))
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || iamSvc.userID != "u_1" || len(iamSvc.assigned.RoleIDs) != 1 {
		t.Fatalf("unexpected assign roles status=%d svc=%+v body=%s", w.Code, iamSvc, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/roles", strings.NewReader(`{"org_id":"org_1","name":"Ops","code":"ops"}`))
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated || iamSvc.roleReq.Code != "ops" {
		t.Fatalf("unexpected create role status=%d roleReq=%+v body=%s", w.Code, iamSvc.roleReq, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/api-permissions", strings.NewReader(`{"method":"GET","path_pattern":"/api/v1/users","permission_code":"user:read","enabled":true}`))
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated || iamSvc.apiReq.PermissionCode != "user:read" {
		t.Fatalf("unexpected create api permission status=%d req=%+v body=%s", w.Code, iamSvc.apiReq, w.Body.String())
	}
}
