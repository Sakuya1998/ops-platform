package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ops-platform/bff-api/internal/client"
	"github.com/ops-platform/bff-api/internal/handler"
)

type recordingAuthHandlerService struct {
	loginReq       client.LoginRequest
	listUsersReq   client.ListUsersRequest
	userID         string
	userContext    client.UserContext
	logoutContext  client.UserContext
	changePassword client.ChangePasswordRequest
	mfaCode        string
	sessionID      string
	redirect       string
}

func (s *recordingAuthHandlerService) Login(ctx context.Context, req client.LoginRequest, meta client.RequestMeta) (client.LoginResponse, error) {
	s.loginReq = req
	return client.LoginResponse{AccessToken: "at", RefreshToken: "rt", User: client.LoginUser{UserID: "u_1", OrgID: "org_1"}}, nil
}

func (s *recordingAuthHandlerService) Refresh(ctx context.Context, req client.RefreshRequest) (client.LoginResponse, error) {
	return client.LoginResponse{AccessToken: "at_new", RefreshToken: "rt_new"}, nil
}

func (s *recordingAuthHandlerService) VerifyToken(ctx context.Context, authorization string) (client.TokenContext, error) {
	return client.TokenContext{Active: true, UserID: "u_1", OrgID: "org_1"}, nil
}

func (s *recordingAuthHandlerService) Logout(ctx context.Context, authorization string, userCtx client.UserContext) error {
	s.logoutContext = userCtx
	return nil
}

func (s *recordingAuthHandlerService) GetCurrentUser(ctx context.Context, userCtx client.UserContext) (map[string]any, error) {
	return map[string]any{"id": userCtx.UserID, "username": "admin"}, nil
}

func (s *recordingAuthHandlerService) ListUsers(ctx context.Context, req client.ListUsersRequest, userCtx client.UserContext) ([]map[string]any, int64, error) {
	s.listUsersReq = req
	s.userContext = userCtx
	return []map[string]any{{"id": "u_1", "username": "admin"}}, 1, nil
}

func (s *recordingAuthHandlerService) CreateUser(ctx context.Context, req client.CreateUserRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userContext = userCtx
	return map[string]any{"id": "u_2", "username": req.Username}, nil
}

func (s *recordingAuthHandlerService) GetUser(ctx context.Context, id string, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userContext = userCtx
	return map[string]any{"id": id, "username": "admin"}, nil
}

func (s *recordingAuthHandlerService) UpdateUser(ctx context.Context, id string, req client.UpdateUserRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userContext = userCtx
	return map[string]any{"id": id, "display_name": req.DisplayName}, nil
}

func (s *recordingAuthHandlerService) DeleteUser(ctx context.Context, id string, userCtx client.UserContext) error {
	s.userID = id
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthHandlerService) UpdateUserStatus(ctx context.Context, id string, req client.UpdateUserStatusRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userContext = userCtx
	return map[string]any{"id": id, "status": req.Status}, nil
}

func (s *recordingAuthHandlerService) ResetUserPassword(ctx context.Context, id string, req client.ResetUserPasswordRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userContext = userCtx
	return map[string]any{"id": id, "must_change_password": req.MustChangePassword}, nil
}

func (s *recordingAuthHandlerService) ListOrganizations(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userContext = userCtx
	return []map[string]any{{"id": "org_1", "code": "default"}}, nil
}

func (s *recordingAuthHandlerService) CreateOrganization(ctx context.Context, req client.OrganizationRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userContext = userCtx
	return map[string]any{"id": "org_2", "code": req.Code}, nil
}

func (s *recordingAuthHandlerService) UpdateOrganization(ctx context.Context, id string, req client.OrganizationRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userContext = userCtx
	return map[string]any{"id": id, "name": req.Name}, nil
}

func (s *recordingAuthHandlerService) GetSystemConfig(ctx context.Context, userCtx client.UserContext) (map[string]any, error) {
	s.userContext = userCtx
	return map[string]any{"ldap_enabled": true}, nil
}

func (s *recordingAuthHandlerService) UpdateSystemConfig(ctx context.Context, req map[string]any, userCtx client.UserContext) error {
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthHandlerService) ChangePassword(ctx context.Context, req client.ChangePasswordRequest, userCtx client.UserContext) error {
	s.changePassword = req
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthHandlerService) SetupMFA(ctx context.Context, userCtx client.UserContext) (map[string]any, error) {
	s.userContext = userCtx
	return map[string]any{"secret": "otp-secret"}, nil
}

func (s *recordingAuthHandlerService) ConfirmMFA(ctx context.Context, req client.MFAConfirmRequest, userCtx client.UserContext) (map[string]any, error) {
	s.mfaCode = req.Code
	s.userContext = userCtx
	return map[string]any{"recovery_codes": []string{"r1"}}, nil
}

func (s *recordingAuthHandlerService) DisableMFA(ctx context.Context, req client.MFACodeRequest, userCtx client.UserContext) error {
	s.mfaCode = req.Code
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthHandlerService) RegenerateMFARecoveryCodes(ctx context.Context, req client.MFACodeRequest, userCtx client.UserContext) (map[string]any, error) {
	s.mfaCode = req.Code
	s.userContext = userCtx
	return map[string]any{"recovery_codes": []string{"r2"}}, nil
}

func (s *recordingAuthHandlerService) ListSessions(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userContext = userCtx
	return []map[string]any{{"id": "s_1"}}, nil
}

func (s *recordingAuthHandlerService) RevokeSession(ctx context.Context, sessionID string, userCtx client.UserContext) error {
	s.sessionID = sessionID
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthHandlerService) RevokeOtherSessions(ctx context.Context, userCtx client.UserContext) error {
	s.userContext = userCtx
	return nil
}

func (s *recordingAuthHandlerService) OIDCLogin(ctx context.Context) (string, error) {
	if s.redirect == "" {
		return "https://idp.example.com/login", nil
	}
	return s.redirect, nil
}

func (s *recordingAuthHandlerService) OIDCCallback(ctx context.Context, req client.OIDCCallbackRequest) (string, error) {
	if s.redirect == "" {
		return "/login/callback?code=local", nil
	}
	return s.redirect, nil
}

func (s *recordingAuthHandlerService) OIDCStatus(ctx context.Context) (map[string]any, error) {
	return map[string]any{"enabled": true}, nil
}

func (s *recordingAuthHandlerService) OIDCExchange(ctx context.Context, req client.OIDCExchangeRequest) (client.LoginResponse, error) {
	return client.LoginResponse{AccessToken: "at", RefreshToken: "rt"}, nil
}

func TestAuthOwnedRoutesUseExplicitHandlers(t *testing.T) {
	authSvc := &recordingAuthHandlerService{}
	r := New(Dependencies{
		Bootstrap:  handler.NewBootstrapHandler("bff-api"),
		Auth:       handler.NewAuthHandler(authSvc),
		IAM:        handler.NewIAMHandler(&recordingIAMRouterService{}),
		Audit:      handler.NewAuditHandler(&recordingAuditRouterService{}),
		Notify:     handler.NewNotifyHandler(&recordingNotifyRouterService{}),
		Permission: &recordingPermissionChecker{result: client.CheckPermissionResult{Allowed: true}},
		TokenVerifier: &recordingTokenVerifier{result: client.TokenContext{
			Active: true, UserID: "u_1", OrgID: "org_1", SessionID: "s_1",
		}},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin"}`))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if authSvc.loginReq.Username != "admin" {
		t.Fatalf("expected explicit handler login, got %+v", authSvc.loginReq)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/api/v1/auth/me/password", strings.NewReader(`{"old_password":"a","new_password":"b"}`))
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if authSvc.changePassword.NewPassword != "b" || authSvc.userContext.UserID != "u_1" {
		t.Fatalf("password route should use explicit handler req=%+v ctx=%+v", authSvc.changePassword, authSvc.userContext)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users?page=2&page_size=10&keyword=adm", nil)
	req.Header.Set("X-User-Id", "u_1")
	req.Header.Set("X-Org-Id", "org_1")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if authSvc.listUsersReq.Page != 2 || authSvc.listUsersReq.Keyword != "adm" || authSvc.userContext.OrgID != "org_1" {
		t.Fatalf("users route should use explicit handler req=%+v ctx=%+v", authSvc.listUsersReq, authSvc.userContext)
	}
}
