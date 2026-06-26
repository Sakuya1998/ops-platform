package client

import (
	"context"
	"fmt"
	"strings"

	authv1 "github.com/Sakuya1998/ops-platform/pkg/proto/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthGRPCClient struct {
	client       authv1.AuthServiceClient
	redirectHTTP *AuthHTTPClient
}

func NewAuthGRPCClient(conn grpc.ClientConnInterface, redirectHTTP *AuthHTTPClient) *AuthGRPCClient {
	return &AuthGRPCClient{client: authv1.NewAuthServiceClient(conn), redirectHTTP: redirectHTTP}
}

func (c *AuthGRPCClient) Login(ctx context.Context, req LoginRequest, meta RequestMeta) (LoginResponse, error) {
	resp, err := c.client.Login(ctx, &authv1.LoginRequest{
		Username: req.Username, Password: req.Password, OrgCode: req.OrgCode, Provider: req.Provider, MfaCode: req.MFACode,
		UserAgent: meta.UserAgent, DeviceName: meta.DeviceName, RequestId: meta.RequestID,
	})
	if err != nil {
		return LoginResponse{}, err
	}
	return loginFromProto(resp), nil
}

func (c *AuthGRPCClient) Refresh(ctx context.Context, req RefreshRequest) (LoginResponse, error) {
	resp, err := c.client.RefreshToken(ctx, &authv1.RefreshTokenRequest{RefreshToken: req.RefreshToken})
	if err != nil {
		return LoginResponse{}, err
	}
	return loginFromRefreshProto(resp), nil
}

func (c *AuthGRPCClient) VerifyToken(ctx context.Context, authorization string) (TokenContext, error) {
	token := bearerToken(authorization)
	if token == "" {
		return TokenContext{}, fmt.Errorf("authorization header required")
	}
	resp, err := c.client.VerifyToken(ctx, &authv1.VerifyTokenRequest{Token: token})
	if err != nil {
		return TokenContext{}, err
	}
	if !resp.Valid {
		return TokenContext{}, fmt.Errorf("inactive token")
	}
	if resp.UserId == "" || resp.OrgId == "" {
		return TokenContext{}, fmt.Errorf("auth token verify returned incomplete context")
	}
	return TokenContext{Active: resp.Valid, UserID: resp.UserId, OrgID: resp.OrgId, SessionID: resp.SessionId, JTI: resp.Jti}, nil
}

func (c *AuthGRPCClient) Logout(ctx context.Context, authorization string, userCtx UserContext) error {
	_, err := c.client.Logout(ctx, &authv1.LogoutRequest{UserId: userCtx.UserID, AccessToken: bearerToken(authorization)})
	return err
}

func (c *AuthGRPCClient) GetCurrentUser(ctx context.Context, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.GetCurrentUser(ctx, &authv1.GetCurrentUserRequest{UserId: userCtx.UserID})
	if err != nil {
		return nil, err
	}
	return authUserToMap(resp), nil
}

func (c *AuthGRPCClient) ListUsers(ctx context.Context, req ListUsersRequest, userCtx UserContext) ([]map[string]any, int64, error) {
	resp, err := c.client.ListUsers(ctx, &authv1.ListUsersRequest{
		OrgId: firstNonEmpty(req.OrgID, userCtx.OrgID), Page: int32(req.Page), PageSize: int32(req.PageSize), Keyword: req.Keyword,
	})
	if err != nil {
		return nil, 0, err
	}
	users := make([]map[string]any, 0, len(resp.Users))
	for _, user := range resp.Users {
		users = append(users, authUserToMap(user))
	}
	return users, resp.Total, nil
}

func (c *AuthGRPCClient) CreateUser(ctx context.Context, req CreateUserRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.CreateUser(ctx, &authv1.CreateUserRequest{
		OrgId: firstNonEmpty(req.OrgID, userCtx.OrgID), Username: req.Username, Email: req.Email, Phone: req.Phone,
		DisplayName: req.DisplayName, Password: req.Password,
	})
	if err != nil {
		return nil, err
	}
	return authUserToMap(resp), nil
}

func (c *AuthGRPCClient) GetUser(ctx context.Context, id string, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.GetUser(ctx, &authv1.GetUserRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return authUserToMap(resp), nil
}

func (c *AuthGRPCClient) UpdateUser(ctx context.Context, id string, req UpdateUserRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.UpdateUser(ctx, &authv1.UpdateUserRequest{Id: id, Email: req.Email, Phone: req.Phone, DisplayName: req.DisplayName})
	if err != nil {
		return nil, err
	}
	return authUserToMap(resp), nil
}

func (c *AuthGRPCClient) DeleteUser(ctx context.Context, id string, userCtx UserContext) error {
	_, err := c.client.DeleteUser(ctx, &authv1.DeleteUserRequest{Id: id})
	return err
}

func (c *AuthGRPCClient) UpdateUserStatus(ctx context.Context, id string, req UpdateUserStatusRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.UpdateUserStatus(ctx, &authv1.UpdateUserStatusRequest{Id: id, Status: req.Status})
	if err != nil {
		return nil, err
	}
	return authUserToMap(resp), nil
}

func (c *AuthGRPCClient) ResetUserPassword(ctx context.Context, id string, req ResetUserPasswordRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.ResetUserPassword(ctx, &authv1.ResetUserPasswordRequest{Id: id, NewPassword: req.NewPassword, MustChangePassword: req.MustChangePassword})
	if err != nil {
		return nil, err
	}
	return authUserToMap(resp), nil
}

func (c *AuthGRPCClient) ListOrganizations(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	resp, err := c.client.ListOrganizations(ctx, &authv1.ListOrganizationsRequest{})
	if err != nil {
		return nil, err
	}
	orgs := make([]map[string]any, 0, len(resp.Organizations))
	for _, org := range resp.Organizations {
		orgs = append(orgs, organizationToMap(org))
	}
	return orgs, nil
}

func (c *AuthGRPCClient) CreateOrganization(ctx context.Context, req OrganizationRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.CreateOrganization(ctx, &authv1.CreateOrganizationRequest{Name: req.Name, Code: req.Code, Description: req.Description})
	if err != nil {
		return nil, err
	}
	return organizationToMap(resp), nil
}

func (c *AuthGRPCClient) UpdateOrganization(ctx context.Context, id string, req OrganizationRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.UpdateOrganization(ctx, &authv1.UpdateOrganizationRequest{Id: id, Name: req.Name, Description: req.Description})
	if err != nil {
		return nil, err
	}
	return organizationToMap(resp), nil
}

func (c *AuthGRPCClient) GetSystemConfig(ctx context.Context, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.GetSystemConfig(ctx, &authv1.GetSystemConfigRequest{OrgId: userCtx.OrgID})
	if err != nil {
		return nil, err
	}
	return systemConfigToMap(resp), nil
}

func (c *AuthGRPCClient) UpdateSystemConfig(ctx context.Context, req map[string]any, userCtx UserContext) error {
	_, err := c.client.UpdateSystemConfig(ctx, &authv1.UpdateSystemConfigRequest{OrgId: userCtx.OrgID, Config: systemConfigFromMap(req)})
	return err
}

func (c *AuthGRPCClient) ChangePassword(ctx context.Context, req ChangePasswordRequest, userCtx UserContext) error {
	_, err := c.client.ChangePassword(ctx, &authv1.ChangePasswordRequest{UserId: userCtx.UserID, OldPassword: req.OldPassword, NewPassword: req.NewPassword})
	return err
}

func (c *AuthGRPCClient) SetupMFA(ctx context.Context, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.SetupMFA(ctx, &authv1.MFASetupRequest{UserId: userCtx.UserID})
	if err != nil {
		return nil, err
	}
	return map[string]any{"secret": resp.Secret, "otpauth_url": resp.OtpauthUrl, "recovery_codes": resp.RecoveryCodes}, nil
}

func (c *AuthGRPCClient) ConfirmMFA(ctx context.Context, req MFAConfirmRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.ConfirmMFA(ctx, &authv1.ConfirmMFARequest{UserId: userCtx.UserID, Code: req.Code})
	if err != nil {
		return nil, err
	}
	return map[string]any{"recovery_codes": resp.RecoveryCodes}, nil
}

func (c *AuthGRPCClient) DisableMFA(ctx context.Context, req MFACodeRequest, userCtx UserContext) error {
	_, err := c.client.DisableMFA(ctx, &authv1.DisableMFARequest{UserId: userCtx.UserID, Code: req.Code})
	return err
}

func (c *AuthGRPCClient) RegenerateMFARecoveryCodes(ctx context.Context, req MFACodeRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.RegenerateMFARecoveryCodes(ctx, &authv1.RegenerateMFARecoveryCodesRequest{UserId: userCtx.UserID, Code: req.Code})
	if err != nil {
		return nil, err
	}
	return map[string]any{"recovery_codes": resp.RecoveryCodes}, nil
}

func (c *AuthGRPCClient) ListSessions(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	resp, err := c.client.ListSessions(ctx, &authv1.ListSessionsRequest{UserId: userCtx.UserID})
	if err != nil {
		return nil, err
	}
	sessions := make([]map[string]any, 0, len(resp.Sessions))
	for _, session := range resp.Sessions {
		sessions = append(sessions, sessionToMap(session))
	}
	return sessions, nil
}

func (c *AuthGRPCClient) RevokeSession(ctx context.Context, sessionID string, userCtx UserContext) error {
	_, err := c.client.RevokeSession(ctx, &authv1.RevokeSessionRequest{UserId: userCtx.UserID, SessionId: sessionID, Reason: "revoked_by_user"})
	return err
}

func (c *AuthGRPCClient) RevokeOtherSessions(ctx context.Context, userCtx UserContext) error {
	_, err := c.client.RevokeOtherSessions(ctx, &authv1.RevokeOtherSessionsRequest{UserId: userCtx.UserID, CurrentSessionId: userCtx.SessionID})
	return err
}

func (c *AuthGRPCClient) OIDCLogin(ctx context.Context) (string, error) {
	if c.redirectHTTP == nil {
		return "", fmt.Errorf("oidc login redirect requires http fallback")
	}
	return c.redirectHTTP.OIDCLogin(ctx)
}

func (c *AuthGRPCClient) OIDCCallback(ctx context.Context, req OIDCCallbackRequest) (string, error) {
	if c.redirectHTTP == nil {
		return "", fmt.Errorf("oidc callback redirect requires http fallback")
	}
	return c.redirectHTTP.OIDCCallback(ctx, req)
}

func (c *AuthGRPCClient) OIDCStatus(ctx context.Context) (map[string]any, error) {
	resp, err := c.client.OIDCStatus(ctx, &authv1.OIDCStatusRequest{})
	if err != nil {
		return nil, err
	}
	return map[string]any{"enabled": resp.Enabled, "provider_name": resp.ProviderName, "login_url": resp.LoginUrl}, nil
}

func (c *AuthGRPCClient) OIDCExchange(ctx context.Context, req OIDCExchangeRequest) (LoginResponse, error) {
	resp, err := c.client.OIDCExchange(ctx, &authv1.OIDCExchangeRequest{Code: req.Code})
	if err != nil {
		return LoginResponse{}, err
	}
	return loginFromProto(resp), nil
}

func loginFromProto(resp *authv1.LoginResponse) LoginResponse {
	if resp == nil {
		return LoginResponse{}
	}
	return LoginResponse{
		AccessToken: resp.AccessToken, RefreshToken: resp.RefreshToken, ExpiresIn: int(resp.ExpiresIn),
		TokenType: resp.TokenType, SessionID: resp.SessionId, JTI: resp.Jti, User: loginUserFromProto(resp.User),
	}
}

func loginFromRefreshProto(resp *authv1.RefreshTokenResponse) LoginResponse {
	if resp == nil {
		return LoginResponse{}
	}
	return LoginResponse{
		AccessToken: resp.AccessToken, RefreshToken: resp.RefreshToken, ExpiresIn: int(resp.ExpiresIn),
		TokenType: resp.TokenType, SessionID: resp.SessionId, JTI: resp.Jti, User: loginUserFromProto(resp.User),
	}
}

func loginUserFromProto(user *authv1.UserInfo) LoginUser {
	if user == nil {
		return LoginUser{}
	}
	return LoginUser{
		UserID: user.UserId, OrgID: user.OrgId, Username: user.Username, DisplayName: user.DisplayName,
		Email: user.Email, Roles: user.Roles, MustChangePassword: user.MustChangePassword, MFAEnabled: user.MfaEnabled,
	}
}

func authUserToMap(user *authv1.User) map[string]any {
	if user == nil {
		return map[string]any{}
	}
	result := map[string]any{
		"id":                    user.Id,
		"org_id":                user.OrgId,
		"username":              user.Username,
		"email":                 user.Email,
		"phone":                 user.Phone,
		"display_name":          user.DisplayName,
		"avatar":                user.Avatar,
		"status":                user.Status,
		"source":                user.Source,
		"failed_login_attempts": user.FailedLoginAttempts,
		"must_change_password":  user.MustChangePassword,
		"mfa_enabled":           user.MfaEnabled,
	}
	addTime(result, "locked_until", user.LockedUntil)
	addTime(result, "password_changed_at", user.PasswordChangedAt)
	addTime(result, "mfa_confirmed_at", user.MfaConfirmedAt)
	addTime(result, "last_login_at", user.LastLoginAt)
	addTime(result, "deleted_at", user.DeletedAt)
	addTime(result, "created_at", user.CreatedAt)
	addTime(result, "updated_at", user.UpdatedAt)
	return result
}

func organizationToMap(org *authv1.Organization) map[string]any {
	if org == nil {
		return map[string]any{}
	}
	result := map[string]any{"id": org.Id, "name": org.Name, "code": org.Code, "description": org.Description, "logo": org.Logo, "status": org.Status}
	addTime(result, "created_at", org.CreatedAt)
	addTime(result, "updated_at", org.UpdatedAt)
	return result
}

func sessionToMap(session *authv1.Session) map[string]any {
	if session == nil {
		return map[string]any{}
	}
	result := map[string]any{
		"id": session.Id, "user_id": session.UserId, "org_id": session.OrgId, "status": session.Status,
		"ip": session.Ip, "user_agent": session.UserAgent, "device_name": session.DeviceName,
		"revoked_reason": session.RevokedReason,
	}
	addTime(result, "last_seen_at", session.LastSeenAt)
	addTime(result, "expires_at", session.ExpiresAt)
	addTime(result, "revoked_at", session.RevokedAt)
	addTime(result, "created_at", session.CreatedAt)
	addTime(result, "updated_at", session.UpdatedAt)
	return result
}

func systemConfigToMap(cfg *authv1.SystemConfig) map[string]any {
	result := map[string]any{}
	if cfg == nil {
		return result
	}
	for _, provider := range cfg.Providers {
		data := jsonToMap(provider.ConfigJson)
		data["enabled"] = provider.IsEnabled
		result[provider.Provider] = data
	}
	return result
}

func systemConfigFromMap(input map[string]any) *authv1.SystemConfig {
	cfg := &authv1.SystemConfig{Providers: []*authv1.AuthProvider{}}
	for _, name := range []string{"ldap", "oidc"} {
		if raw, ok := input[name]; ok {
			asMap, _ := raw.(map[string]any)
			enabled, _ := asMap["enabled"].(bool)
			cfg.Providers = append(cfg.Providers, &authv1.AuthProvider{
				Provider: name, Name: strings.ToUpper(name) + " Authentication", ConfigJson: mustJSON(asMap), IsEnabled: enabled,
			})
		}
	}
	return cfg
}

func addTime(result map[string]any, key string, ts *timestamppb.Timestamp) {
	if ts != nil {
		result[key] = ts.AsTime()
	}
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
