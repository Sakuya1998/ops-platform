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

type RequestMeta struct {
	UserAgent  string
	DeviceName string
	RequestID  string
}

type UserContext struct {
	UserID    string
	OrgID     string
	SessionID string
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	OrgCode  string `json:"org_code,omitempty"`
	Provider string `json:"provider,omitempty"`
	MFACode  string `json:"mfa_code,omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ListUsersRequest struct {
	OrgID    string
	Page     int
	PageSize int
	Keyword  string
}

type CreateUserRequest struct {
	OrgID       string `json:"org_id"`
	Username    string `json:"username"`
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Password    string `json:"password"`
}

type UpdateUserRequest struct {
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

type UpdateUserStatusRequest struct {
	Status string `json:"status"`
}

type ResetUserPasswordRequest struct {
	NewPassword        string `json:"new_password"`
	MustChangePassword bool   `json:"must_change_password"`
}

type OrganizationRequest struct {
	Name        string `json:"name,omitempty"`
	Code        string `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type MFAConfirmRequest struct {
	Code string `json:"code"`
}

type MFACodeRequest struct {
	Code string `json:"code,omitempty"`
}

type OIDCCallbackRequest struct {
	Code  string
	State string
}

type OIDCExchangeRequest struct {
	Code string `json:"code"`
}

type LoginUser struct {
	UserID             string   `json:"user_id"`
	OrgID              string   `json:"org_id"`
	Username           string   `json:"username"`
	DisplayName        string   `json:"display_name,omitempty"`
	Email              string   `json:"email,omitempty"`
	Roles              []string `json:"roles,omitempty"`
	MustChangePassword bool     `json:"must_change_password"`
	MFAEnabled         bool     `json:"mfa_enabled"`
}

type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	TokenType    string    `json:"token_type"`
	SessionID    string    `json:"session_id,omitempty"`
	JTI          string    `json:"jti,omitempty"`
	User         LoginUser `json:"user"`
}

type TokenContext struct {
	Active    bool   `json:"active"`
	UserID    string `json:"user_id"`
	OrgID     string `json:"org_id"`
	SessionID string `json:"session_id"`
	JTI       string `json:"jti"`
	Reason    string `json:"reason"`
}

type AuthHTTPClient struct {
	baseURL *url.URL
	client  *http.Client
}

func NewAuthHTTPClient(baseURL string, timeout time.Duration) (*AuthHTTPClient, error) {
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
	return &AuthHTTPClient{
		baseURL: parsed,
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (c *AuthHTTPClient) Login(ctx context.Context, req LoginRequest, meta RequestMeta) (LoginResponse, error) {
	var result LoginResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/login", req, func(httpReq *http.Request) {
		applyRequestMeta(httpReq.Header, meta)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) Refresh(ctx context.Context, req RefreshRequest) (LoginResponse, error) {
	var result LoginResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/refresh", req, nil, &result)
	return result, err
}

func (c *AuthHTTPClient) VerifyToken(ctx context.Context, authorization string) (TokenContext, error) {
	if strings.TrimSpace(authorization) == "" {
		return TokenContext{}, fmt.Errorf("authorization header required")
	}
	var result TokenContext
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/token/verify", nil, func(httpReq *http.Request) {
		httpReq.Header.Set("Authorization", authorization)
	}, &result)
	if err != nil {
		return TokenContext{}, err
	}
	if !result.Active {
		reason := result.Reason
		if reason == "" {
			reason = "inactive token"
		}
		return TokenContext{}, fmt.Errorf(reason)
	}
	if result.UserID == "" || result.OrgID == "" {
		return TokenContext{}, fmt.Errorf("auth token verify returned incomplete context")
	}
	return result, nil
}

func (c *AuthHTTPClient) Logout(ctx context.Context, authorization string, userCtx UserContext) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/auth/logout", nil, func(httpReq *http.Request) {
		if authorization != "" {
			httpReq.Header.Set("Authorization", authorization)
		}
		applyUserContext(httpReq.Header, userCtx)
	}, nil)
}

func (c *AuthHTTPClient) GetCurrentUser(ctx context.Context, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/auth/me", nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) ListUsers(ctx context.Context, req ListUsersRequest, userCtx UserContext) ([]map[string]any, int64, error) {
	values := url.Values{}
	if req.OrgID != "" {
		values.Set("org_id", req.OrgID)
	}
	if req.Page > 0 {
		values.Set("page", fmt.Sprintf("%d", req.Page))
	}
	if req.PageSize > 0 {
		values.Set("page_size", fmt.Sprintf("%d", req.PageSize))
	}
	if req.Keyword != "" {
		values.Set("keyword", req.Keyword)
	}
	path := "/api/v1/users"
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var users []map[string]any
	total, err := c.doPagedJSON(ctx, http.MethodGet, path, nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &users)
	return users, total, err
}

func (c *AuthHTTPClient) CreateUser(ctx context.Context, req CreateUserRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/users", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) GetUser(ctx context.Context, id string, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/users/"+url.PathEscape(id), nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) UpdateUser(ctx context.Context, id string, req UpdateUserRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/users/"+url.PathEscape(id), req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) DeleteUser(ctx context.Context, id string, userCtx UserContext) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/users/"+url.PathEscape(id), nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, nil)
}

func (c *AuthHTTPClient) UpdateUserStatus(ctx context.Context, id string, req UpdateUserStatusRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/users/"+url.PathEscape(id)+"/status", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) ResetUserPassword(ctx context.Context, id string, req ResetUserPasswordRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/users/"+url.PathEscape(id)+"/password/reset", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) ListOrganizations(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	var result []map[string]any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/organizations", nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) CreateOrganization(ctx context.Context, req OrganizationRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/organizations", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) UpdateOrganization(ctx context.Context, id string, req OrganizationRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPut, "/api/v1/organizations/"+url.PathEscape(id), req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) GetSystemConfig(ctx context.Context, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/system/config", nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) UpdateSystemConfig(ctx context.Context, req map[string]any, userCtx UserContext) error {
	return c.doJSON(ctx, http.MethodPut, "/api/v1/system/config", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, nil)
}

func (c *AuthHTTPClient) ChangePassword(ctx context.Context, req ChangePasswordRequest, userCtx UserContext) error {
	return c.doJSON(ctx, http.MethodPut, "/api/v1/auth/me/password", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, nil)
}

func (c *AuthHTTPClient) SetupMFA(ctx context.Context, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/me/mfa/setup", nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) ConfirmMFA(ctx context.Context, req MFAConfirmRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/me/mfa/confirm", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) DisableMFA(ctx context.Context, req MFACodeRequest, userCtx UserContext) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/auth/me/mfa", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, nil)
}

func (c *AuthHTTPClient) RegenerateMFARecoveryCodes(ctx context.Context, req MFACodeRequest, userCtx UserContext) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/me/mfa/recovery-codes", req, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) ListSessions(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	var result []map[string]any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/auth/sessions", nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, &result)
	return result, err
}

func (c *AuthHTTPClient) RevokeSession(ctx context.Context, sessionID string, userCtx UserContext) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/auth/sessions/"+url.PathEscape(sessionID), nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, nil)
}

func (c *AuthHTTPClient) RevokeOtherSessions(ctx context.Context, userCtx UserContext) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/auth/sessions", nil, func(httpReq *http.Request) {
		applyUserContext(httpReq.Header, userCtx)
	}, nil)
}

func (c *AuthHTTPClient) OIDCLogin(ctx context.Context) (string, error) {
	return c.doRedirect(ctx, "/api/v1/auth/oidc/login")
}

func (c *AuthHTTPClient) OIDCCallback(ctx context.Context, req OIDCCallbackRequest) (string, error) {
	values := url.Values{}
	values.Set("code", req.Code)
	values.Set("state", req.State)
	return c.doRedirect(ctx, "/api/v1/auth/oidc/callback?"+values.Encode())
}

func (c *AuthHTTPClient) OIDCStatus(ctx context.Context) (map[string]any, error) {
	var result map[string]any
	err := c.doJSON(ctx, http.MethodGet, "/api/v1/auth/oidc/status", nil, nil, &result)
	return result, err
}

func (c *AuthHTTPClient) OIDCExchange(ctx context.Context, req OIDCExchangeRequest) (LoginResponse, error) {
	var result LoginResponse
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/oidc/exchange", req, nil, &result)
	return result, err
}

func (c *AuthHTTPClient) endpoint(path string) string {
	u := *c.baseURL
	relative := path
	if strings.HasPrefix(relative, "/") {
		relative = relative[1:]
	}
	if parsed, err := url.Parse(relative); err == nil {
		u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(parsed.Path, "/")
		u.RawQuery = parsed.RawQuery
		return u.String()
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	return u.String()
}

func (c *AuthHTTPClient) doJSON(ctx context.Context, method, path string, body any, mutate func(*http.Request), out any) error {
	_, err := c.doEnvelope(ctx, method, path, body, mutate, out)
	return err
}

func (c *AuthHTTPClient) doPagedJSON(ctx context.Context, method, path string, body any, mutate func(*http.Request), out any) (int64, error) {
	return c.doEnvelope(ctx, method, path, body, mutate, out)
}

func (c *AuthHTTPClient) doRedirect(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(path), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusMultipleChoices || resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("auth redirect %s failed with status %d", path, resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("auth redirect %s returned empty location", path)
	}
	return location, nil
}

func (c *AuthHTTPClient) doEnvelope(ctx context.Context, method, path string, body any, mutate func(*http.Request), out any) (int64, error) {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
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
		return 0, fmt.Errorf("auth request %s %s failed with status %d", method, path, resp.StatusCode)
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
			msg = "auth request failed"
		}
		return 0, fmt.Errorf(msg)
	}
	if out == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return envelope.Total, nil
	}
	return envelope.Total, json.Unmarshal(envelope.Data, out)
}

func applyRequestMeta(header http.Header, meta RequestMeta) {
	if meta.UserAgent != "" {
		header.Set("User-Agent", meta.UserAgent)
	}
	if meta.DeviceName != "" {
		header.Set("X-Device-Name", meta.DeviceName)
	}
	if meta.RequestID != "" {
		header.Set("X-Request-Id", meta.RequestID)
	}
}

func applyUserContext(header http.Header, userCtx UserContext) {
	if userCtx.UserID != "" {
		header.Set("X-User-Id", userCtx.UserID)
	}
	if userCtx.OrgID != "" {
		header.Set("X-Org-Id", userCtx.OrgID)
	}
	if userCtx.SessionID != "" {
		header.Set("X-Session-Id", userCtx.SessionID)
	}
}
