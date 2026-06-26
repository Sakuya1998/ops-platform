package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ops-platform/bff-api/internal/client"
)

type recordingAuthService struct {
	loginReq        client.LoginRequest
	loginMeta       client.RequestMeta
	refreshReq      client.RefreshRequest
	listUsersReq    client.ListUsersRequest
	createUserReq   client.CreateUserRequest
	userID          string
	userContext     client.UserContext
	verifyAuth      string
	logoutAuth      string
	logoutContext   client.UserContext
	currentUserCtx  client.UserContext
	loginResp       client.LoginResponse
	refreshResp     client.LoginResponse
	verifyResp      client.TokenContext
	currentUserResp map[string]any
	usersResp       []map[string]any
	total           int64
	userResp        map[string]any
	orgsResp        []map[string]any
	orgResp         map[string]any
	configResp      map[string]any
	changePassword  client.ChangePasswordRequest
	mfaCode         string
	sessionID       string
	oidcCallback    client.OIDCCallbackRequest
	redirect        string
}

func (s *recordingAuthService) Login(ctx context.Context, req client.LoginRequest, meta client.RequestMeta) (client.LoginResponse, error) {
	s.loginReq = req
	s.loginMeta = meta
	return s.loginResp, nil
}

func (s *recordingAuthService) Refresh(ctx context.Context, req client.RefreshRequest) (client.LoginResponse, error) {
	s.refreshReq = req
	return s.refreshResp, nil
}

func (s *recordingAuthService) VerifyToken(ctx context.Context, authorization string) (client.TokenContext, error) {
	s.verifyAuth = authorization
	return s.verifyResp, nil
}

func (s *recordingAuthService) Logout(ctx context.Context, authorization string, userCtx client.UserContext) error {
	s.logoutAuth = authorization
	s.logoutContext = userCtx
	return nil
}

func (s *recordingAuthService) GetCurrentUser(ctx context.Context, userCtx client.UserContext) (map[string]any, error) {
	s.currentUserCtx = userCtx
	return s.currentUserResp, nil
}

func (s *recordingAuthService) ListUsers(ctx context.Context, req client.ListUsersRequest, userCtx client.UserContext) ([]map[string]any, int64, error) {
	s.listUsersReq = req
	s.userContext = userCtx
	return s.usersResp, s.total, nil
}

func (s *recordingAuthService) CreateUser(ctx context.Context, req client.CreateUserRequest, userCtx client.UserContext) (map[string]any, error) {
	s.createUserReq = req
	s.userContext = userCtx
	return s.userResp, nil
}

func (s *recordingAuthService) GetUser(ctx context.Context, id string, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userContext = userCtx
	return s.userResp, nil
}

func (s *recordingAuthService) UpdateUser(ctx context.Context, id string, req client.UpdateUserRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userContext = userCtx
	return s.userResp, nil
}

func (s *recordingAuthService) DeleteUser(ctx context.Context, id string, userCtx client.UserContext) error {
	s.userID = id
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthService) UpdateUserStatus(ctx context.Context, id string, req client.UpdateUserStatusRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userContext = userCtx
	return s.userResp, nil
}

func (s *recordingAuthService) ResetUserPassword(ctx context.Context, id string, req client.ResetUserPasswordRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userContext = userCtx
	return s.userResp, nil
}

func (s *recordingAuthService) ListOrganizations(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userContext = userCtx
	return s.orgsResp, nil
}

func (s *recordingAuthService) CreateOrganization(ctx context.Context, req client.OrganizationRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userContext = userCtx
	return s.orgResp, nil
}

func (s *recordingAuthService) UpdateOrganization(ctx context.Context, id string, req client.OrganizationRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userContext = userCtx
	return s.orgResp, nil
}

func (s *recordingAuthService) GetSystemConfig(ctx context.Context, userCtx client.UserContext) (map[string]any, error) {
	s.userContext = userCtx
	return s.configResp, nil
}

func (s *recordingAuthService) UpdateSystemConfig(ctx context.Context, req map[string]any, userCtx client.UserContext) error {
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthService) ChangePassword(ctx context.Context, req client.ChangePasswordRequest, userCtx client.UserContext) error {
	s.changePassword = req
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthService) SetupMFA(ctx context.Context, userCtx client.UserContext) (map[string]any, error) {
	s.userContext = userCtx
	return map[string]any{"secret": "otp-secret"}, nil
}

func (s *recordingAuthService) ConfirmMFA(ctx context.Context, req client.MFAConfirmRequest, userCtx client.UserContext) (map[string]any, error) {
	s.mfaCode = req.Code
	s.userContext = userCtx
	return map[string]any{"recovery_codes": []string{"r1"}}, nil
}

func (s *recordingAuthService) DisableMFA(ctx context.Context, req client.MFACodeRequest, userCtx client.UserContext) error {
	s.mfaCode = req.Code
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthService) RegenerateMFARecoveryCodes(ctx context.Context, req client.MFACodeRequest, userCtx client.UserContext) (map[string]any, error) {
	s.mfaCode = req.Code
	s.userContext = userCtx
	return map[string]any{"recovery_codes": []string{"r2"}}, nil
}

func (s *recordingAuthService) ListSessions(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userContext = userCtx
	return []map[string]any{{"id": "s_1"}}, nil
}

func (s *recordingAuthService) RevokeSession(ctx context.Context, sessionID string, userCtx client.UserContext) error {
	s.sessionID = sessionID
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthService) RevokeOtherSessions(ctx context.Context, userCtx client.UserContext) error {
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthService) OIDCLogin(ctx context.Context) (string, error) {
	return s.redirect, nil
}

func (s *recordingAuthService) OIDCCallback(ctx context.Context, req client.OIDCCallbackRequest) (string, error) {
	s.oidcCallback = req
	return s.redirect, nil
}

func (s *recordingAuthService) OIDCStatus(ctx context.Context) (map[string]any, error) {
	return map[string]any{"enabled": true}, nil
}

func (s *recordingAuthService) OIDCExchange(ctx context.Context, req client.OIDCExchangeRequest) (client.LoginResponse, error) {
	return client.LoginResponse{AccessToken: "at", RefreshToken: "rt"}, nil
}

func TestAuthHandlerLoginUsesExplicitDTOAndReturnsEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &recordingAuthService{loginResp: client.LoginResponse{
		AccessToken:  "at",
		RefreshToken: "rt",
		ExpiresIn:    7200,
		TokenType:    "Bearer",
		User: client.LoginUser{
			UserID:   "u_1",
			OrgID:    "org_1",
			Username: "admin",
			Roles:    []string{"admin"},
		},
	}}
	r := gin.New()
	NewAuthHandler(svc).Register(r.Group("/api/v1/auth"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret","org_code":"default","provider":"local","mfa_code":"123456"}`))
	req.Header.Set("User-Agent", "Codex Test")
	req.Header.Set("X-Device-Name", "browser")
	req.Header.Set("X-Request-Id", "req_1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if svc.loginReq.Username != "admin" || svc.loginReq.Password != "secret" || svc.loginReq.OrgCode != "default" || svc.loginReq.MFACode != "123456" {
		t.Fatalf("unexpected login request: %+v", svc.loginReq)
	}
	if svc.loginMeta.UserAgent != "Codex Test" || svc.loginMeta.DeviceName != "browser" || svc.loginMeta.RequestID != "req_1" {
		t.Fatalf("unexpected login meta: %+v", svc.loginMeta)
	}
	var body struct {
		Code int                  `json:"code"`
		Data client.LoginResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != 0 || body.Data.AccessToken != "at" || body.Data.User.UserID != "u_1" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestAuthHandlerProtectedEndpointsUseUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &recordingAuthService{
		refreshResp:     client.LoginResponse{AccessToken: "at_new", RefreshToken: "rt_new"},
		verifyResp:      client.TokenContext{Active: true, UserID: "u_1", OrgID: "org_1"},
		currentUserResp: map[string]any{"id": "u_1", "username": "admin"},
	}
	r := gin.New()
	NewAuthHandler(svc).Register(r.Group("/api/v1/auth"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(`{"refresh_token":"rt_old"}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.refreshReq.RefreshToken != "rt_old" {
		t.Fatalf("unexpected refresh status=%d req=%+v body=%s", w.Code, svc.refreshReq, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/token/verify", nil)
	req.Header.Set("Authorization", "Bearer at")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.verifyAuth != "Bearer at" {
		t.Fatalf("unexpected verify status=%d auth=%q", w.Code, svc.verifyAuth)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer at")
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	req.Header.Set("X-Session-Id", "s_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.logoutAuth != "Bearer at" || svc.logoutContext.UserID != "u_1" || svc.logoutContext.SessionID != "s_1" {
		t.Fatalf("unexpected logout status=%d auth=%q ctx=%+v", w.Code, svc.logoutAuth, svc.logoutContext)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.currentUserCtx.UserID != "u_1" {
		t.Fatalf("unexpected me status=%d ctx=%+v body=%s", w.Code, svc.currentUserCtx, w.Body.String())
	}
}

func TestAuthHandlerIdentityCenterEndpointsUseExplicitService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &recordingAuthService{
		usersResp:  []map[string]any{{"id": "u_1", "username": "admin"}},
		total:      1,
		userResp:   map[string]any{"id": "u_1", "username": "admin"},
		orgsResp:   []map[string]any{{"id": "org_1", "code": "default"}},
		orgResp:    map[string]any{"id": "org_1", "code": "default"},
		configResp: map[string]any{"ldap_enabled": true},
	}
	r := gin.New()
	auth := NewAuthHandler(svc)
	auth.RegisterUserRoutes(r.Group("/api/v1/users"))
	auth.RegisterOrganizationRoutes(r.Group("/api/v1/organizations"))
	auth.RegisterSystemRoutes(r.Group("/api/v1/system"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users?page=2&page_size=10&keyword=adm", nil)
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.listUsersReq.Page != 2 || svc.listUsersReq.Keyword != "adm" || svc.userContext.OrgID != "org_1" {
		t.Fatalf("unexpected list users status=%d req=%+v ctx=%+v body=%s", w.Code, svc.listUsersReq, svc.userContext, w.Body.String())
	}
	var listBody struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list users: %v", err)
	}
	if listBody.Total != 1 {
		t.Fatalf("expected total 1, got %d", listBody.Total)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"org_id":"org_1","username":"ops","password":"secret"}`))
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated || svc.createUserReq.Username != "ops" {
		t.Fatalf("unexpected create user status=%d req=%+v body=%s", w.Code, svc.createUserReq, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/users/u_1/status", strings.NewReader(`{"status":"disabled"}`))
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.userID != "u_1" {
		t.Fatalf("unexpected update status=%d userID=%s", w.Code, svc.userID)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected list orgs status=%d body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/system/config", nil)
	req.Header.Set("X-User-Id", "actor_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.configResp["ldap_enabled"] != true {
		t.Fatalf("unexpected config status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestAuthHandlerAccountSecuritySessionAndOIDCEndpointsUseExplicitService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &recordingAuthService{redirect: "https://idp.example.com/login"}
	r := gin.New()
	NewAuthHandler(svc).Register(r.Group("/api/v1/auth"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/me/password", strings.NewReader(`{"old_password":"old","new_password":"new"}`))
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.changePassword.NewPassword != "new" || svc.userContext.UserID != "u_1" {
		t.Fatalf("unexpected change password status=%d req=%+v ctx=%+v", w.Code, svc.changePassword, svc.userContext)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/me/mfa/confirm", strings.NewReader(`{"code":"123456"}`))
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.mfaCode != "123456" {
		t.Fatalf("unexpected mfa confirm status=%d code=%s", w.Code, svc.mfaCode)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/sessions", nil)
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	req.Header.Set("X-Session-Id", "s_current")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.userContext.SessionID != "s_current" {
		t.Fatalf("unexpected sessions status=%d ctx=%+v", w.Code, svc.userContext)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/s_1", nil)
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || svc.sessionID != "s_1" {
		t.Fatalf("unexpected revoke session status=%d sessionID=%s", w.Code, svc.sessionID)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/login", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "https://idp.example.com/login" {
		t.Fatalf("unexpected oidc login status=%d location=%s", w.Code, w.Header().Get("Location"))
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/oidc/callback?code=c1&state=s1", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound || svc.oidcCallback.Code != "c1" || svc.oidcCallback.State != "s1" {
		t.Fatalf("unexpected oidc callback status=%d req=%+v", w.Code, svc.oidcCallback)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/oidc/exchange", strings.NewReader(`{"code":"login-code"}`))
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected oidc exchange status=%d body=%s", w.Code, w.Body.String())
	}
}
