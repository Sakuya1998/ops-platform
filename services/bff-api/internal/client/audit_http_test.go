package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuditHTTPClientListLogsAndEventTypes(t *testing.T) {
	var gotOrg, gotQuery string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		gotOrg = r.Header.Get("X-Org-Id")
		gotQuery = r.URL.RawQuery
		switch r.URL.Path {
		case "/api/v1/audit-logs":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","total":1,"data":[{"id":"a_1","event_type":"user.login"}]}`))
		case "/api/v1/audit-logs/event-types":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":["user.login","role.created"]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer backend.Close()

	c, err := NewAuditHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	userCtx := UserContext{UserID: "u_1", OrgID: "org_1", SessionID: "s_1"}
	logs, total, err := c.ListLogs(testContext(t), AuditLogQuery{
		OrgID:     "org_1",
		EventType: "user.login",
		Page:      2,
		PageSize:  10,
	}, userCtx)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if total != 1 || len(logs) != 1 || logs[0]["event_type"] != "user.login" {
		t.Fatalf("unexpected logs total=%d logs=%+v", total, logs)
	}
	if gotOrg != "org_1" || gotQuery == "" {
		t.Fatalf("expected context and query forwarded org=%q query=%q", gotOrg, gotQuery)
	}
	types, err := c.ListEventTypes(testContext(t), userCtx)
	if err != nil || len(types) != 2 {
		t.Fatalf("types=%+v err=%v", types, err)
	}
}

func decodeEnvelopeData(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var envelope struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	return envelope.Data
}
