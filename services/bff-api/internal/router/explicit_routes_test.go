package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/client"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/handler"
)

func TestPhaseOneServiceRoutesUseExplicitHandlers(t *testing.T) {
	auditSvc := &recordingAuditRouterService{}
	notifySvc := &recordingNotifyRouterService{}
	deps := Dependencies{
		Bootstrap:     handler.NewBootstrapHandler("bff-api"),
		Auth:          handler.NewAuthHandler(&recordingAuthHandlerService{}),
		IAM:           handler.NewIAMHandler(&recordingIAMRouterService{}),
		Audit:         handler.NewAuditHandler(auditSvc),
		Notify:        handler.NewNotifyHandler(notifySvc),
		Permission:    &recordingPermissionChecker{result: client.CheckPermissionResult{Allowed: true}},
		TokenVerifier: &recordingTokenVerifier{result: client.TokenContext{Active: true, UserID: "u_1", OrgID: "org_1", SessionID: "s_1"}},
	}
	r := New(deps)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/audit-logs?event_type=user.login"},
		{http.MethodGet, "/api/v1/audit-logs/event-types"},
		{http.MethodGet, "/api/v1/notifications"},
		{http.MethodGet, "/api/v1/notify/templates"},
		{http.MethodGet, "/api/v1/notify/logs?status=success"},
	}

	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("X-User-Id", "u_1")
		req.Header.Set("X-Org-Id", "org_1")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s %s: expected 200, got %d: %s", tc.method, tc.path, w.Code, w.Body.String())
		}
	}
	if auditSvc.query.EventType != "user.login" {
		t.Fatalf("expected audit handler used, got query %+v", auditSvc.query)
	}
	if notifySvc.logQuery.Status != "success" {
		t.Fatalf("expected notify handler used, got query %+v", notifySvc.logQuery)
	}
}
