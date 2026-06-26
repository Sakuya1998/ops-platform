package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuthHTTPClientLoginForwardsRequestMetaAndDecodesEnvelope(t *testing.T) {
	var gotUserAgent, gotDeviceName, gotRequestID string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotUserAgent = r.Header.Get("User-Agent")
		gotDeviceName = r.Header.Get("X-Device-Name")
		gotRequestID = r.Header.Get("X-Request-Id")

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Username != "admin" || req.Password != "secret" || req.OrgCode != "default" || req.Provider != "local" || req.MFACode != "123456" {
			t.Fatalf("unexpected login request: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"at","refresh_token":"rt","expires_in":7200,"token_type":"Bearer","session_id":"s_1","jti":"j_1","user":{"user_id":"u_1","org_id":"org_1","username":"admin","display_name":"Admin","email":"admin@example.com","roles":["admin"],"must_change_password":false,"mfa_enabled":true}}}`))
	}))
	defer backend.Close()

	c, err := NewAuthHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	result, err := c.Login(testContext(t), LoginRequest{
		Username: "admin",
		Password: "secret",
		OrgCode:  "default",
		Provider: "local",
		MFACode:  "123456",
	}, RequestMeta{
		UserAgent:  "Codex Test",
		DeviceName: "browser",
		RequestID:  "req_1",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if gotUserAgent != "Codex Test" || gotDeviceName != "browser" || gotRequestID != "req_1" {
		t.Fatalf("headers not forwarded userAgent=%q device=%q requestID=%q", gotUserAgent, gotDeviceName, gotRequestID)
	}
	if result.AccessToken != "at" || result.RefreshToken != "rt" || result.User.UserID != "u_1" || !result.User.MFAEnabled {
		t.Fatalf("unexpected login response: %+v", result)
	}
}

func TestAuthHTTPClientLoginReturnsEnvelopeError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":401,"message":"invalid credentials"}`))
	}))
	defer backend.Close()

	c, err := NewAuthHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = c.Login(testContext(t), LoginRequest{Username: "admin", Password: "bad"}, RequestMeta{})
	if err == nil || !strings.Contains(err.Error(), "invalid credentials") {
		t.Fatalf("expected envelope error, got %v", err)
	}
}

func TestAuthHTTPClientRefreshLogoutAndCurrentUser(t *testing.T) {
	var logoutAuth, currentUserID string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/refresh":
			var req RefreshRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode refresh: %v", err)
			}
			if req.RefreshToken != "rt_old" {
				t.Fatalf("unexpected refresh token: %s", req.RefreshToken)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"at_new","refresh_token":"rt_new","expires_in":7200,"token_type":"Bearer","user":{"user_id":"u_1","org_id":"org_1","username":"admin"}}}`))
		case "/api/v1/auth/logout":
			logoutAuth = r.Header.Get("Authorization")
			_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
		case "/api/v1/auth/me":
			currentUserID = r.Header.Get("X-User-Id")
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"id":"u_1","username":"admin","display_name":"Admin"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer backend.Close()

	c, err := NewAuthHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	refreshed, err := c.Refresh(testContext(t), RefreshRequest{RefreshToken: "rt_old"})
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.AccessToken != "at_new" || refreshed.RefreshToken != "rt_new" {
		t.Fatalf("unexpected refresh response: %+v", refreshed)
	}
	if err := c.Logout(testContext(t), "Bearer at_new", UserContext{UserID: "u_1", OrgID: "org_1", SessionID: "s_1"}); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if logoutAuth != "Bearer at_new" {
		t.Fatalf("expected logout authorization forwarded, got %q", logoutAuth)
	}
	user, err := c.GetCurrentUser(testContext(t), UserContext{UserID: "u_1", OrgID: "org_1"})
	if err != nil {
		t.Fatalf("current user: %v", err)
	}
	if currentUserID != "u_1" || user["username"] != "admin" {
		t.Fatalf("unexpected current user header=%q body=%+v", currentUserID, user)
	}
}

func TestAuthHTTPClientUserOrganizationAndSystemConfigMethods(t *testing.T) {
	var gotListOrg, gotUserHeader, gotCreateBody, gotSystemOrg string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/users":
			if r.Method == http.MethodGet {
				gotListOrg = r.URL.Query().Get("org_id")
				_, _ = w.Write([]byte(`{"code":0,"message":"success","total":1,"data":[{"id":"u_1","username":"admin"}]}`))
				return
			}
			gotCreateBody = mustReadBody(t, r)
			_, _ = w.Write([]byte(`{"code":0,"message":"created","data":{"id":"u_2","username":"ops"}}`))
		case "/api/v1/users/u_1":
			gotUserHeader = r.Header.Get("X-User-Id")
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"id":"u_1","username":"admin"}}`))
		case "/api/v1/users/u_1/status":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"id":"u_1","status":"disabled"}}`))
		case "/api/v1/users/u_1/password/reset":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"id":"u_1","must_change_password":true}}`))
		case "/api/v1/organizations":
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"code":0,"message":"success","data":[{"id":"org_1","code":"default"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"created","data":{"id":"org_2","code":"ops"}}`))
		case "/api/v1/organizations/org_1":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"id":"org_1","name":"Default"}}`))
		case "/api/v1/system/config":
			gotSystemOrg = r.Header.Get("X-Org-Id")
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"ldap_enabled":true}}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer backend.Close()

	c, err := NewAuthHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	userCtx := UserContext{UserID: "actor_1", OrgID: "org_1", SessionID: "s_1"}
	users, total, err := c.ListUsers(testContext(t), ListUsersRequest{OrgID: "org_1", Page: 2, PageSize: 10, Keyword: "adm"}, userCtx)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if total != 1 || gotListOrg != "org_1" || len(users) != 1 || users[0]["username"] != "admin" {
		t.Fatalf("unexpected list users total=%d org=%s users=%+v", total, gotListOrg, users)
	}
	created, err := c.CreateUser(testContext(t), CreateUserRequest{OrgID: "org_1", Username: "ops", Password: "secret"}, userCtx)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created["username"] != "ops" || !strings.Contains(gotCreateBody, `"username":"ops"`) {
		t.Fatalf("unexpected created=%+v body=%s", created, gotCreateBody)
	}
	user, err := c.GetUser(testContext(t), "u_1", userCtx)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if gotUserHeader != "actor_1" || user["id"] != "u_1" {
		t.Fatalf("unexpected get user header=%s user=%+v", gotUserHeader, user)
	}
	if _, err := c.UpdateUserStatus(testContext(t), "u_1", UpdateUserStatusRequest{Status: "disabled"}, userCtx); err != nil {
		t.Fatalf("update user status: %v", err)
	}
	if _, err := c.ResetUserPassword(testContext(t), "u_1", ResetUserPasswordRequest{NewPassword: "new", MustChangePassword: true}, userCtx); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if orgs, err := c.ListOrganizations(testContext(t), userCtx); err != nil || len(orgs) != 1 {
		t.Fatalf("list orgs orgs=%+v err=%v", orgs, err)
	}
	if _, err := c.CreateOrganization(testContext(t), OrganizationRequest{Name: "Ops", Code: "ops"}, userCtx); err != nil {
		t.Fatalf("create org: %v", err)
	}
	if _, err := c.UpdateOrganization(testContext(t), "org_1", OrganizationRequest{Name: "Default"}, userCtx); err != nil {
		t.Fatalf("update org: %v", err)
	}
	cfg, err := c.GetSystemConfig(testContext(t), userCtx)
	if err != nil {
		t.Fatalf("get system config: %v", err)
	}
	if gotSystemOrg != "org_1" || cfg["ldap_enabled"] != true {
		t.Fatalf("unexpected system config org=%s cfg=%+v", gotSystemOrg, cfg)
	}
	if err := c.UpdateSystemConfig(testContext(t), map[string]any{"ldap_enabled": false}, userCtx); err != nil {
		t.Fatalf("update system config: %v", err)
	}
}

func TestAuthHTTPClientAccountSecuritySessionAndOIDCMethods(t *testing.T) {
	var gotPaths []string
	var gotSessionHeader string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/me/password":
			_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
		case "/api/v1/auth/me/mfa/setup":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"secret":"otp-secret","qr_code":"otpauth://totp"}}`))
		case "/api/v1/auth/me/mfa/confirm":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"recovery_codes":["r1","r2"]}}`))
		case "/api/v1/auth/me/mfa/recovery-codes":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"recovery_codes":["r3"]}}`))
		case "/api/v1/auth/me/mfa":
			_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
		case "/api/v1/auth/sessions":
			gotSessionHeader = r.Header.Get("X-Session-Id")
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"code":0,"message":"success","data":[{"id":"s_1","device_name":"browser"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
		case "/api/v1/auth/sessions/s_1":
			_, _ = w.Write([]byte(`{"code":0,"message":"success"}`))
		case "/api/v1/auth/oidc/status":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"enabled":true,"login_url":"/api/v1/auth/oidc/login"}}`))
		case "/api/v1/auth/oidc/exchange":
			_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"access_token":"at","refresh_token":"rt","user":{"user_id":"u_1","org_id":"org_1"}}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer backend.Close()

	c, err := NewAuthHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	userCtx := UserContext{UserID: "u_1", OrgID: "org_1", SessionID: "s_current"}
	if err := c.ChangePassword(testContext(t), ChangePasswordRequest{OldPassword: "old", NewPassword: "new"}, userCtx); err != nil {
		t.Fatalf("change password: %v", err)
	}
	setup, err := c.SetupMFA(testContext(t), userCtx)
	if err != nil || setup["secret"] != "otp-secret" {
		t.Fatalf("setup mfa result=%+v err=%v", setup, err)
	}
	if _, err := c.ConfirmMFA(testContext(t), MFAConfirmRequest{Code: "123456"}, userCtx); err != nil {
		t.Fatalf("confirm mfa: %v", err)
	}
	if err := c.DisableMFA(testContext(t), MFACodeRequest{Code: "123456"}, userCtx); err != nil {
		t.Fatalf("disable mfa: %v", err)
	}
	if _, err := c.RegenerateMFARecoveryCodes(testContext(t), MFACodeRequest{Code: "123456"}, userCtx); err != nil {
		t.Fatalf("regenerate recovery codes: %v", err)
	}
	sessions, err := c.ListSessions(testContext(t), userCtx)
	if err != nil || len(sessions) != 1 {
		t.Fatalf("list sessions sessions=%+v err=%v", sessions, err)
	}
	if gotSessionHeader != "s_current" {
		t.Fatalf("expected session header forwarded, got %q", gotSessionHeader)
	}
	if err := c.RevokeSession(testContext(t), "s_1", userCtx); err != nil {
		t.Fatalf("revoke session: %v", err)
	}
	if err := c.RevokeOtherSessions(testContext(t), userCtx); err != nil {
		t.Fatalf("revoke other sessions: %v", err)
	}
	status, err := c.OIDCStatus(testContext(t))
	if err != nil || status["enabled"] != true {
		t.Fatalf("oidc status=%+v err=%v", status, err)
	}
	login, err := c.OIDCExchange(testContext(t), OIDCExchangeRequest{Code: "login-code"})
	if err != nil || login.AccessToken != "at" {
		t.Fatalf("oidc exchange=%+v err=%v", login, err)
	}
	if len(gotPaths) < 10 {
		t.Fatalf("expected account/session/oidc paths to be called, got %+v", gotPaths)
	}
}

func TestAuthHTTPClientOIDCRedirectsReturnLocation(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/oidc/login":
			http.Redirect(w, r, "https://idp.example.com/login", http.StatusFound)
		case "/api/v1/auth/oidc/callback":
			if r.URL.Query().Get("code") != "c1" || r.URL.Query().Get("state") != "s1" {
				t.Fatalf("unexpected callback query: %s", r.URL.RawQuery)
			}
			http.Redirect(w, r, "/login/callback?code=local", http.StatusFound)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer backend.Close()

	c, err := NewAuthHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	location, err := c.OIDCLogin(testContext(t))
	if err != nil || location != "https://idp.example.com/login" {
		t.Fatalf("oidc login location=%q err=%v", location, err)
	}
	location, err = c.OIDCCallback(testContext(t), OIDCCallbackRequest{Code: "c1", State: "s1"})
	if err != nil || location != "/login/callback?code=local" {
		t.Fatalf("oidc callback location=%q err=%v", location, err)
	}
}

func TestAuthHTTPClientVerifyTokenUsesAuthorizationHeader(t *testing.T) {
	var gotAuth string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/token/verify" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"message":"success","data":{"active":true,"user_id":"u_1","org_id":"org_1","session_id":"s_1","jti":"j_1"}}`))
	}))
	defer backend.Close()

	c, err := NewAuthHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	result, err := c.VerifyToken(testContext(t), "Bearer access-token")
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}

	if gotAuth != "Bearer access-token" {
		t.Fatalf("expected auth header forwarded, got %q", gotAuth)
	}
	if !result.Active || result.UserID != "u_1" || result.OrgID != "org_1" || result.SessionID != "s_1" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func mustReadBody(t *testing.T, r *http.Request) string {
	t.Helper()
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return string(raw)
}

func TestAuthHTTPClientVerifyTokenRejectsInactiveToken(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    0,
			"message": "success",
			"data": map[string]any{
				"active": false,
				"reason": "token expired",
			},
		})
	}))
	defer backend.Close()

	c, err := NewAuthHTTPClient(backend.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	_, err = c.VerifyToken(testContext(t), "Bearer expired")
	if err == nil {
		t.Fatal("expected inactive token error")
	}
}
