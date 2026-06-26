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

type CheckPermissionRequest struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
	Method string `json:"method"`
	Path   string `json:"path"`
}

type CheckPermissionResult struct {
	Allowed bool     `json:"allowed"`
	Reason  string   `json:"reason"`
	Roles   []string `json:"roles"`
}

type AssignRolesRequest struct {
	RoleIDs []string `json:"role_ids"`
}

type RoleRequest struct {
	OrgID       string `json:"org_id,omitempty"`
	Name        string `json:"name,omitempty"`
	Code        string `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}

type AssignPermissionsRequest struct {
	PermissionIDs []string `json:"permission_ids"`
}

type APIPermissionRequest struct {
	Method         string `json:"method"`
	PathPattern    string `json:"path_pattern"`
	PermissionCode string `json:"permission_code"`
	Description    string `json:"description,omitempty"`
	Enabled        bool   `json:"enabled"`
}

type IAMHTTPClient struct {
	baseURL *url.URL
	client  *http.Client
}

func NewIAMHTTPClient(baseURL string, timeout time.Duration) (*IAMHTTPClient, error) {
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
	return &IAMHTTPClient{baseURL: parsed, client: &http.Client{Timeout: timeout}}, nil
}

func (c *IAMHTTPClient) CheckPermission(ctx context.Context, req CheckPermissionRequest) (CheckPermissionResult, error) {
	var result CheckPermissionResult
	err := c.doJSON(ctx, http.MethodPost, "/internal/v1/check-permission", req, nil, &result)
	return result, err
}

func (c *IAMHTTPClient) AssignUserRoles(ctx context.Context, userID string, req AssignRolesRequest, userCtx UserContext) error {
	return c.doJSON(ctx, http.MethodPut, "/api/v1/users/"+url.PathEscape(userID)+"/roles", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, nil)
}

func (c *IAMHTTPClient) GetUserRoles(ctx context.Context, userID string, userCtx UserContext) ([]map[string]any, error) {
	var result []map[string]any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/users/"+url.PathEscape(userID)+"/roles", nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *IAMHTTPClient) GetUserPermissions(ctx context.Context, userID string, userCtx UserContext) ([]any, error) {
	var result []any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/users/"+url.PathEscape(userID)+"/permissions", nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *IAMHTTPClient) ListRoles(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	var result []map[string]any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/roles", nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *IAMHTTPClient) CreateRole(ctx context.Context, req RoleRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/roles", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *IAMHTTPClient) GetRole(ctx context.Context, id string, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/roles/"+url.PathEscape(id), nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *IAMHTTPClient) UpdateRole(ctx context.Context, id string, req RoleRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/roles/"+url.PathEscape(id), req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *IAMHTTPClient) DeleteRole(ctx context.Context, id string, userCtx UserContext) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/roles/"+url.PathEscape(id), nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, nil)
}

func (c *IAMHTTPClient) AssignRolePermissions(ctx context.Context, roleID string, req AssignPermissionsRequest, userCtx UserContext) error {
	return c.doJSON(ctx, http.MethodPut, "/api/v1/roles/"+url.PathEscape(roleID)+"/permissions", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, nil)
}

func (c *IAMHTTPClient) GetPermissionTree(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	var result []map[string]any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/permissions", nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *IAMHTTPClient) ListAPIPermissions(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	var result []map[string]any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/api-permissions", nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *IAMHTTPClient) CreateAPIPermission(ctx context.Context, req APIPermissionRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/api-permissions", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *IAMHTTPClient) UpdateAPIPermission(ctx context.Context, id string, req APIPermissionRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/api-permissions/"+url.PathEscape(id), req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *IAMHTTPClient) DeleteAPIPermission(ctx context.Context, id string, userCtx UserContext) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/api-permissions/"+url.PathEscape(id), nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, nil)
}

func (c *IAMHTTPClient) doJSON(ctx context.Context, method, path string, body any, mutate func(*http.Request), out any) error {
	reader := bytes.NewReader(nil)
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, c.endpoint(path), reader)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Accept", "application/json")
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if mutate != nil {
		mutate(httpReq)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("iam request %s %s failed with status %d", method, path, resp.StatusCode)
	}

	var envelope struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}
	if envelope.Code != 0 {
		msg := envelope.Message
		if msg == "" {
			msg = "iam request failed"
		}
		return fmt.Errorf(msg)
	}
	if out == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	return json.Unmarshal(envelope.Data, out)
}

func (c *IAMHTTPClient) endpoint(path string) string {
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + path
	return u.String()
}
