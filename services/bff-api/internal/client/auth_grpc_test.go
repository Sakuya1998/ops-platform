package client

import (
	"context"
	"net"
	"testing"
	"time"

	authv1 "github.com/ops-platform/pkg/proto/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAuthGRPCClientIdentityCenterMethods(t *testing.T) {
	conn := newAuthBufConn(t, &stubAuthServer{})
	defer conn.Close()
	c := NewAuthGRPCClient(conn, nil)
	userCtx := UserContext{UserID: "u_1", OrgID: "org_1", SessionID: "s_1"}

	login, err := c.Login(testContext(t), LoginRequest{Username: "admin", Password: "secret"}, RequestMeta{UserAgent: "agent", DeviceName: "dev", RequestID: "req_1"})
	if err != nil || login.AccessToken != "access" || login.User.UserID != "u_1" {
		t.Fatalf("login=%+v err=%v", login, err)
	}
	refresh, err := c.Refresh(testContext(t), RefreshRequest{RefreshToken: "refresh"})
	if err != nil || refresh.AccessToken != "access2" {
		t.Fatalf("refresh=%+v err=%v", refresh, err)
	}
	token, err := c.VerifyToken(testContext(t), "Bearer access")
	if err != nil || !token.Active || token.UserID != "u_1" || token.OrgID != "org_1" {
		t.Fatalf("token=%+v err=%v", token, err)
	}
	if err := c.Logout(testContext(t), "Bearer access", userCtx); err != nil {
		t.Fatalf("logout: %v", err)
	}
	current, err := c.GetCurrentUser(testContext(t), userCtx)
	if err != nil || current["id"] != "u_1" {
		t.Fatalf("current=%+v err=%v", current, err)
	}
	users, total, err := c.ListUsers(testContext(t), ListUsersRequest{Page: 1, PageSize: 20}, userCtx)
	if err != nil || total != 1 || len(users) != 1 || users[0]["username"] != "admin" {
		t.Fatalf("users=%+v total=%d err=%v", users, total, err)
	}
	created, err := c.CreateUser(testContext(t), CreateUserRequest{OrgID: "org_1", Username: "ops", Password: "Admin@2026"}, userCtx)
	if err != nil || created["username"] != "ops" {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	if _, err := c.UpdateUser(testContext(t), "u_1", UpdateUserRequest{DisplayName: "Admin Updated"}, userCtx); err != nil {
		t.Fatalf("update user: %v", err)
	}
	if _, err := c.UpdateUserStatus(testContext(t), "u_1", UpdateUserStatusRequest{Status: "disabled"}, userCtx); err != nil {
		t.Fatalf("update status: %v", err)
	}
	if _, err := c.ResetUserPassword(testContext(t), "u_1", ResetUserPasswordRequest{NewPassword: "Admin@2027", MustChangePassword: true}, userCtx); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if err := c.DeleteUser(testContext(t), "u_1", userCtx); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	orgs, err := c.ListOrganizations(testContext(t), userCtx)
	if err != nil || len(orgs) != 1 || orgs[0]["code"] != "default" {
		t.Fatalf("orgs=%+v err=%v", orgs, err)
	}
	if _, err := c.CreateOrganization(testContext(t), OrganizationRequest{Name: "Ops", Code: "ops"}, userCtx); err != nil {
		t.Fatalf("create org: %v", err)
	}
	if _, err := c.UpdateOrganization(testContext(t), "org_1", OrganizationRequest{Name: "Default Updated"}, userCtx); err != nil {
		t.Fatalf("update org: %v", err)
	}
	cfg, err := c.GetSystemConfig(testContext(t), userCtx)
	if err != nil || cfg["ldap"] == nil || cfg["oidc"] == nil {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
	if err := c.UpdateSystemConfig(testContext(t), cfg, userCtx); err != nil {
		t.Fatalf("update cfg: %v", err)
	}
	if err := c.ChangePassword(testContext(t), ChangePasswordRequest{OldPassword: "old", NewPassword: "new"}, userCtx); err != nil {
		t.Fatalf("change password: %v", err)
	}
	mfa, err := c.SetupMFA(testContext(t), userCtx)
	if err != nil || mfa["secret"] != "secret" {
		t.Fatalf("mfa=%+v err=%v", mfa, err)
	}
	confirmed, err := c.ConfirmMFA(testContext(t), MFAConfirmRequest{Code: "123456"}, userCtx)
	if err != nil || len(confirmed["recovery_codes"].([]string)) != 1 {
		t.Fatalf("confirmed=%+v err=%v", confirmed, err)
	}
	if err := c.DisableMFA(testContext(t), MFACodeRequest{Code: "123456"}, userCtx); err != nil {
		t.Fatalf("disable mfa: %v", err)
	}
	if _, err := c.RegenerateMFARecoveryCodes(testContext(t), MFACodeRequest{Code: "123456"}, userCtx); err != nil {
		t.Fatalf("regen codes: %v", err)
	}
	sessions, err := c.ListSessions(testContext(t), userCtx)
	if err != nil || len(sessions) != 1 || sessions[0]["id"] != "s_1" {
		t.Fatalf("sessions=%+v err=%v", sessions, err)
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
	exchanged, err := c.OIDCExchange(testContext(t), OIDCExchangeRequest{Code: "login-code"})
	if err != nil || exchanged.AccessToken != "oidc-access" {
		t.Fatalf("oidc exchange=%+v err=%v", exchanged, err)
	}
}

func TestAuthUserToMapIgnoresNilTimestamps(t *testing.T) {
	result := authUserToMap(&authv1.User{Id: "u_1", OrgId: "org_1", Username: "admin"})
	if _, ok := result["created_at"]; ok {
		t.Fatalf("nil created_at should be omitted: %+v", result)
	}
	if result["id"] != "u_1" {
		t.Fatalf("unexpected user mapping: %+v", result)
	}
}

type stubAuthServer struct {
	authv1.UnimplementedAuthServiceServer
}

func (s *stubAuthServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	return loginResp("access"), nil
}
func (s *stubAuthServer) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	return &authv1.RefreshTokenResponse{AccessToken: "access2", RefreshToken: "refresh2", ExpiresIn: 7200, TokenType: "Bearer", SessionId: "s_1", Jti: "jti_2", User: loginUser()}, nil
}
func (s *stubAuthServer) VerifyToken(ctx context.Context, req *authv1.VerifyTokenRequest) (*authv1.VerifyTokenResponse, error) {
	return &authv1.VerifyTokenResponse{Valid: true, UserId: "u_1", OrgId: "org_1", SessionId: "s_1", Jti: "jti_1"}, nil
}
func (s *stubAuthServer) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	return &authv1.LogoutResponse{}, nil
}
func (s *stubAuthServer) GetCurrentUser(ctx context.Context, req *authv1.GetCurrentUserRequest) (*authv1.User, error) {
	return user("u_1", "admin"), nil
}
func (s *stubAuthServer) ListUsers(ctx context.Context, req *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	return &authv1.ListUsersResponse{Users: []*authv1.User{user("u_1", "admin")}, Total: 1}, nil
}
func (s *stubAuthServer) CreateUser(ctx context.Context, req *authv1.CreateUserRequest) (*authv1.User, error) {
	return user("u_2", req.Username), nil
}
func (s *stubAuthServer) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.User, error) {
	return user(req.Id, "admin"), nil
}
func (s *stubAuthServer) UpdateUser(ctx context.Context, req *authv1.UpdateUserRequest) (*authv1.User, error) {
	return user(req.Id, "admin"), nil
}
func (s *stubAuthServer) UpdateUserStatus(ctx context.Context, req *authv1.UpdateUserStatusRequest) (*authv1.User, error) {
	u := user(req.Id, "admin")
	u.Status = req.Status
	return u, nil
}
func (s *stubAuthServer) ResetUserPassword(ctx context.Context, req *authv1.ResetUserPasswordRequest) (*authv1.User, error) {
	return user(req.Id, "admin"), nil
}
func (s *stubAuthServer) DeleteUser(ctx context.Context, req *authv1.DeleteUserRequest) (*authv1.Empty, error) {
	return &authv1.Empty{}, nil
}
func (s *stubAuthServer) ListOrganizations(ctx context.Context, req *authv1.ListOrganizationsRequest) (*authv1.ListOrganizationsResponse, error) {
	return &authv1.ListOrganizationsResponse{Organizations: []*authv1.Organization{org("org_1", "default")}}, nil
}
func (s *stubAuthServer) CreateOrganization(ctx context.Context, req *authv1.CreateOrganizationRequest) (*authv1.Organization, error) {
	return org("org_2", req.Code), nil
}
func (s *stubAuthServer) UpdateOrganization(ctx context.Context, req *authv1.UpdateOrganizationRequest) (*authv1.Organization, error) {
	return org(req.Id, "default"), nil
}
func (s *stubAuthServer) GetSystemConfig(ctx context.Context, req *authv1.GetSystemConfigRequest) (*authv1.SystemConfig, error) {
	return &authv1.SystemConfig{Providers: []*authv1.AuthProvider{
		{Provider: "ldap", ConfigJson: `{"enabled":true,"host":"ldap.local"}`, IsEnabled: true},
		{Provider: "oidc", ConfigJson: `{"enabled":true,"issuer":"https://issuer"}`, IsEnabled: true},
	}}, nil
}
func (s *stubAuthServer) UpdateSystemConfig(ctx context.Context, req *authv1.UpdateSystemConfigRequest) (*authv1.Empty, error) {
	return &authv1.Empty{}, nil
}
func (s *stubAuthServer) ChangePassword(ctx context.Context, req *authv1.ChangePasswordRequest) (*authv1.Empty, error) {
	return &authv1.Empty{}, nil
}
func (s *stubAuthServer) SetupMFA(ctx context.Context, req *authv1.MFASetupRequest) (*authv1.MFASetupResponse, error) {
	return &authv1.MFASetupResponse{Secret: "secret", OtpauthUrl: "otpauth://totp/x"}, nil
}
func (s *stubAuthServer) ConfirmMFA(ctx context.Context, req *authv1.ConfirmMFARequest) (*authv1.ConfirmMFAResponse, error) {
	return &authv1.ConfirmMFAResponse{RecoveryCodes: []string{"ABCD-2345"}}, nil
}
func (s *stubAuthServer) DisableMFA(ctx context.Context, req *authv1.DisableMFARequest) (*authv1.Empty, error) {
	return &authv1.Empty{}, nil
}
func (s *stubAuthServer) RegenerateMFARecoveryCodes(ctx context.Context, req *authv1.RegenerateMFARecoveryCodesRequest) (*authv1.RegenerateMFARecoveryCodesResponse, error) {
	return &authv1.RegenerateMFARecoveryCodesResponse{RecoveryCodes: []string{"WXYZ-2345"}}, nil
}
func (s *stubAuthServer) ListSessions(ctx context.Context, req *authv1.ListSessionsRequest) (*authv1.ListSessionsResponse, error) {
	return &authv1.ListSessionsResponse{Sessions: []*authv1.Session{{Id: "s_1", UserId: "u_1", OrgId: "org_1", Status: "active", CreatedAt: timestamppb.Now()}}}, nil
}
func (s *stubAuthServer) RevokeSession(ctx context.Context, req *authv1.RevokeSessionRequest) (*authv1.Empty, error) {
	return &authv1.Empty{}, nil
}
func (s *stubAuthServer) RevokeOtherSessions(ctx context.Context, req *authv1.RevokeOtherSessionsRequest) (*authv1.Empty, error) {
	return &authv1.Empty{}, nil
}
func (s *stubAuthServer) OIDCStatus(ctx context.Context, req *authv1.OIDCStatusRequest) (*authv1.OIDCStatusResponse, error) {
	return &authv1.OIDCStatusResponse{Enabled: true, ProviderName: "OIDC", LoginUrl: "/api/v1/auth/oidc/login"}, nil
}
func (s *stubAuthServer) OIDCExchange(ctx context.Context, req *authv1.OIDCExchangeRequest) (*authv1.LoginResponse, error) {
	return loginResp("oidc-access"), nil
}

func loginResp(token string) *authv1.LoginResponse {
	return &authv1.LoginResponse{AccessToken: token, RefreshToken: "refresh", ExpiresIn: 7200, TokenType: "Bearer", SessionId: "s_1", Jti: "jti_1", User: loginUser()}
}
func loginUser() *authv1.UserInfo {
	return &authv1.UserInfo{UserId: "u_1", OrgId: "org_1", Username: "admin", Roles: []string{"admin"}, MfaEnabled: true}
}
func user(id, username string) *authv1.User {
	return &authv1.User{Id: id, OrgId: "org_1", Username: username, Email: username + "@example.com", Status: "active", Source: "local", CreatedAt: timestamppb.New(time.Unix(100, 0))}
}
func org(id, code string) *authv1.Organization {
	return &authv1.Organization{Id: id, Name: code, Code: code, Status: "active", CreatedAt: timestamppb.Now()}
}
func newAuthBufConn(t *testing.T, server authv1.AuthServiceServer) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	authv1.RegisterAuthServiceServer(grpcServer, server)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			t.Logf("auth bufconn server stopped: %v", err)
		}
	}()
	t.Cleanup(grpcServer.Stop)
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	return conn
}
