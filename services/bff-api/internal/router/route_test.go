package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ops-platform/bff-api/internal/client"
	"github.com/ops-platform/bff-api/internal/handler"
)

func testDeps(t *testing.T) Dependencies {
	t.Helper()
	gin.SetMode(gin.TestMode)
	return Dependencies{
		Bootstrap:  handler.NewBootstrapHandler("bff-api"),
		Auth:       handler.NewAuthHandler(&recordingAuthHandlerService{}),
		IAM:        handler.NewIAMHandler(&recordingIAMRouterService{}),
		Audit:      handler.NewAuditHandler(&recordingAuditRouterService{}),
		Notify:     handler.NewNotifyHandler(&recordingNotifyRouterService{}),
		Permission: &recordingPermissionChecker{result: client.CheckPermissionResult{Allowed: true}},
		TokenVerifier: &recordingTokenVerifier{result: client.TokenContext{
			Active: true, UserID: "u_1", OrgID: "org_1", SessionID: "s_1",
		}},
	}
}

func TestHealthRoute(t *testing.T) {
	r := New(testDeps(t))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" || body["service"] != "bff-api" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestBootstrapRouteDescribesTargetArchitecture(t *testing.T) {
	r := New(testDeps(t))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bootstrap", nil)

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Service      string `json:"service"`
		Architecture struct {
			Edge       string `json:"edge"`
			ClientAPI  string `json:"client_api"`
			DomainCall string `json:"domain_call"`
			EventBus   string `json:"event_bus"`
		} `json:"architecture"`
		Modules []struct {
			Name   string `json:"name"`
			Phase  int    `json:"phase"`
			Status string `json:"status"`
		} `json:"modules"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Service != "bff-api" {
		t.Fatalf("unexpected service: %s", body.Service)
	}
	if body.Architecture.Edge != "apisix" || body.Architecture.ClientAPI != "bff-api" ||
		body.Architecture.DomainCall != "grpc" || body.Architecture.EventBus != "kafka" {
		t.Fatalf("unexpected architecture: %+v", body.Architecture)
	}
	if len(body.Modules) != 9 {
		t.Fatalf("expected 9 modules, got %d", len(body.Modules))
	}
	if body.Modules[0].Name != "auth-svc" || body.Modules[1].Name != "iam-svc" {
		t.Fatalf("unexpected leading modules: %+v", body.Modules[:2])
	}
}
