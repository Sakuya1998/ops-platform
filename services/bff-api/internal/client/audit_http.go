package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AuditLogQuery struct {
	OrgID     string
	EventType string
	StartTime string
	EndTime   string
	Page      int
	PageSize  int
}

type AuditHTTPClient struct {
	baseURL *url.URL
	client  *http.Client
}

func NewAuditHTTPClient(baseURL string, timeout time.Duration) (*AuditHTTPClient, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("absolute http url required: %s", baseURL)
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &AuditHTTPClient{baseURL: parsed, client: &http.Client{Timeout: timeout}}, nil
}

func (c *AuditHTTPClient) ListLogs(ctx context.Context, query AuditLogQuery, userCtx UserContext) ([]map[string]any, int64, error) {
	values := url.Values{}
	if query.OrgID != "" {
		values.Set("org_id", query.OrgID)
	}
	if query.EventType != "" {
		values.Set("event_type", query.EventType)
	}
	if query.StartTime != "" {
		values.Set("start_time", query.StartTime)
	}
	if query.EndTime != "" {
		values.Set("end_time", query.EndTime)
	}
	if query.Page > 0 {
		values.Set("page", fmt.Sprintf("%d", query.Page))
	}
	if query.PageSize > 0 {
		values.Set("page_size", fmt.Sprintf("%d", query.PageSize))
	}
	path := "/api/v1/audit-logs"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result []map[string]any
	total, err := c.doJSON(ctx, http.MethodGet, path, nil, func(req *http.Request) {
		applyUserContext(req.Header, userCtx)
	}, &result)
	return result, total, err
}

func (c *AuditHTTPClient) ListEventTypes(ctx context.Context, userCtx UserContext) ([]string, error) {
	var result []string
	_, err := c.doJSON(ctx, http.MethodGet, "/api/v1/audit-logs/event-types", nil, func(req *http.Request) {
		applyUserContext(req.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuditHTTPClient) doJSON(ctx context.Context, method, path string, body any, mutate func(*http.Request), out any) (int64, error) {
	reader := bytes.NewReader(nil)
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint(path), reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if mutate != nil {
		mutate(req)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("audit request %s %s failed with status %d", method, path, resp.StatusCode)
	}
	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
		Total   int64           `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return 0, err
	}
	if envelope.Code != 0 {
		msg := envelope.Message
		if msg == "" {
			msg = "audit request failed"
		}
		return 0, fmt.Errorf(msg)
	}
	if out != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return 0, err
		}
	}
	return envelope.Total, nil
}

func (c *AuditHTTPClient) endpoint(path string) string {
	u := *c.baseURL
	relative := strings.TrimLeft(path, "/")
	if parsed, err := url.Parse(relative); err == nil {
		u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(parsed.Path, "/")
		u.RawQuery = parsed.RawQuery
		return u.String()
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	return u.String()
}
