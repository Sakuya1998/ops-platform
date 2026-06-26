package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNotifyHTTPClientChannelsTemplatesAndLogs(t *testing.T) {
	var gotOrg, gotBody string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		gotOrg = r.Header.Get("X-Org-Id")
		switch r.URL.Path {
		case "/api/v1/notifications":
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"code":0,"message":"success","data":[{"id":"ch_1","name":"Email"}]}`))
				return
			}
			gotBody = readJSONBody(t, r)
			_, _ = w.Write([]byte(`{"code":0,"message":"created","data":{"id":"ch_2","name":"Webhook"}}`))
		case "/api/v1/notifications/ch_1":
			if r.Method == http.MethodDelete {
				_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
				return
			}
			gotBody = readJSONBody(t, r)
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"id":"ch_1","name":"Email Updated"}}`))
		case "/api/v1/notify/templates":
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"code":0,"message":"success","data":[{"id":"tpl_1","name":"Alert"}]}`))
				return
			}
			gotBody = readJSONBody(t, r)
			_, _ = w.Write([]byte(`{"code":0,"message":"created","data":{"id":"tpl_2","name":"Deploy"}}`))
		case "/api/v1/notify/templates/tpl_1":
			if r.Method == http.MethodDelete {
				_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
				return
			}
			gotBody = readJSONBody(t, r)
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"id":"tpl_1","name":"Alert Updated"}}`))
		case "/api/v1/notify/logs":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","total":1,"data":[{"id":"log_1","status":"success"}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer backend.Close()

	c, err := NewNotifyHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	userCtx := UserContext{UserID: "u_1", OrgID: "org_1", SessionID: "s_1"}
	channels, err := c.ListChannels(testContext(t), userCtx)
	if err != nil || len(channels) != 1 || gotOrg != "org_1" {
		t.Fatalf("channels=%+v org=%s err=%v", channels, gotOrg, err)
	}
	channel, err := c.CreateChannel(testContext(t), NotificationChannelRequest{Name: "Webhook", ChannelType: "webhook", Config: map[string]any{"url": "https://example.com"}}, userCtx)
	if err != nil || channel["id"] != "ch_2" || !strings.Contains(gotBody, `"channel_type":"webhook"`) {
		t.Fatalf("channel=%+v body=%s err=%v", channel, gotBody, err)
	}
	if _, err := c.UpdateChannel(testContext(t), "ch_1", NotificationChannelRequest{Name: "Email Updated", ChannelType: "email", Config: map[string]any{"host": "smtp"}}, userCtx); err != nil {
		t.Fatalf("update channel: %v", err)
	}
	if err := c.DeleteChannel(testContext(t), "ch_1", userCtx); err != nil {
		t.Fatalf("delete channel: %v", err)
	}
	templates, err := c.ListTemplates(testContext(t), userCtx)
	if err != nil || len(templates) != 1 {
		t.Fatalf("templates=%+v err=%v", templates, err)
	}
	if _, err := c.CreateTemplate(testContext(t), NotificationTemplateRequest{Name: "Deploy", ChannelType: "webhook", BodyTemplate: "body"}, userCtx); err != nil {
		t.Fatalf("create template: %v", err)
	}
	if _, err := c.UpdateTemplate(testContext(t), "tpl_1", NotificationTemplateRequest{Name: "Alert Updated", ChannelType: "email", BodyTemplate: "body"}, userCtx); err != nil {
		t.Fatalf("update template: %v", err)
	}
	if err := c.DeleteTemplate(testContext(t), "tpl_1", userCtx); err != nil {
		t.Fatalf("delete template: %v", err)
	}
	logs, total, err := c.ListNotificationLogs(testContext(t), NotificationLogQuery{Status: "success", Page: 1, PageSize: 20}, userCtx)
	if err != nil || total != 1 || len(logs) != 1 {
		t.Fatalf("logs=%+v total=%d err=%v", logs, total, err)
	}
}
