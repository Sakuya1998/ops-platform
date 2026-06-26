package grpc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/model"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/service"
	authv1 "github.com/Sakuya1998/ops-platform/pkg/proto/auth/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	authv1.UnimplementedAuthServiceServer
	authSvc *service.AuthService
	oidcSvc *service.OIDCService
}

func NewServer(authSvc *service.AuthService, oidcSvc *service.OIDCService) *Server {
	return &Server{authSvc: authSvc, oidcSvc: oidcSvc}
}

func (s *Server) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	result, err := s.authSvc.Login(ctx, req.Username, req.Password, req.OrgCode, req.Provider, service.LoginContext{
		IP: req.Ip, UserAgent: req.UserAgent, DeviceName: req.DeviceName, MFACode: req.MfaCode, RequestID: req.RequestId,
	})
	if err != nil {
		return nil, err
	}
	return mapLoginResponse(result), nil
}

func (s *Server) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if req.UserId != "" {
		if err := s.authSvc.RevokeAllUserTokens(ctx, req.UserId); err != nil {
			return nil, err
		}
	}
	return &authv1.LogoutResponse{}, nil
}

func (s *Server) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	result, err := s.authSvc.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}
	login := mapLoginResponse(result)
	return &authv1.RefreshTokenResponse{
		AccessToken:  login.AccessToken,
		RefreshToken: login.RefreshToken,
		ExpiresIn:    login.ExpiresIn,
		TokenType:    login.TokenType,
		SessionId:    login.SessionId,
		Jti:          login.Jti,
		User:         login.User,
	}, nil
}

func (s *Server) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	claims, err := s.authSvc.ValidateToken(req.Token)
	if err != nil {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}
	resp := &authv1.ValidateTokenResponse{
		Valid: true, UserId: claims.UserID, OrgId: claims.OrgID, Username: claims.Subject,
		SessionId: claims.SessionID, Jti: claims.ID,
	}
	if claims.ExpiresAt != nil {
		resp.ExpiresAt = claims.ExpiresAt.Unix()
	}
	return resp, nil
}

func (s *Server) VerifyToken(ctx context.Context, req *authv1.VerifyTokenRequest) (*authv1.VerifyTokenResponse, error) {
	result := s.authSvc.VerifyToken(req.Token)
	return &authv1.VerifyTokenResponse{
		Valid: result.Active, UserId: result.UserID, OrgId: result.OrgID,
		SessionId: result.SessionID, Jti: result.JTI, ExpiresAt: result.ExpiresAt,
	}, nil
}

func (s *Server) GetCurrentUser(ctx context.Context, req *authv1.GetCurrentUserRequest) (*authv1.User, error) {
	user, err := s.authSvc.GetUser(req.UserId)
	if err != nil {
		return nil, err
	}
	return mapUser(*user), nil
}

func (s *Server) ChangePassword(ctx context.Context, req *authv1.ChangePasswordRequest) (*authv1.Empty, error) {
	if err := s.authSvc.ChangePassword(ctx, req.UserId, req.OldPassword, req.NewPassword); err != nil {
		return nil, err
	}
	return &authv1.Empty{}, nil
}

func (s *Server) SetupMFA(ctx context.Context, req *authv1.MFASetupRequest) (*authv1.MFASetupResponse, error) {
	result, err := s.authSvc.SetupMFA(req.UserId)
	if err != nil {
		return nil, err
	}
	return &authv1.MFASetupResponse{
		Secret: result.Secret, OtpauthUrl: result.OTPAuthURL,
	}, nil
}

func (s *Server) ConfirmMFA(ctx context.Context, req *authv1.ConfirmMFARequest) (*authv1.ConfirmMFAResponse, error) {
	result, err := s.authSvc.ConfirmMFA(req.UserId, req.Code)
	if err != nil {
		return nil, err
	}
	return &authv1.ConfirmMFAResponse{RecoveryCodes: result.RecoveryCodes}, nil
}

func (s *Server) DisableMFA(ctx context.Context, req *authv1.DisableMFARequest) (*authv1.Empty, error) {
	if err := s.authSvc.DisableMFA(req.UserId, req.Code); err != nil {
		return nil, err
	}
	return &authv1.Empty{}, nil
}

func (s *Server) RegenerateMFARecoveryCodes(ctx context.Context, req *authv1.RegenerateMFARecoveryCodesRequest) (*authv1.RegenerateMFARecoveryCodesResponse, error) {
	result, err := s.authSvc.RegenerateMFARecoveryCodes(req.UserId, req.Code)
	if err != nil {
		return nil, err
	}
	return &authv1.RegenerateMFARecoveryCodesResponse{RecoveryCodes: result.RecoveryCodes}, nil
}

func (s *Server) ListSessions(ctx context.Context, req *authv1.ListSessionsRequest) (*authv1.ListSessionsResponse, error) {
	sessions, err := s.authSvc.ListSessions(req.UserId)
	if err != nil {
		return nil, err
	}
	resp := &authv1.ListSessionsResponse{Sessions: make([]*authv1.Session, 0, len(sessions))}
	for _, session := range sessions {
		resp.Sessions = append(resp.Sessions, mapSession(session))
	}
	return resp, nil
}

func (s *Server) RevokeSession(ctx context.Context, req *authv1.RevokeSessionRequest) (*authv1.Empty, error) {
	if err := s.authSvc.RevokeSession(req.UserId, req.SessionId, req.Reason); err != nil {
		return nil, err
	}
	return &authv1.Empty{}, nil
}

func (s *Server) RevokeOtherSessions(ctx context.Context, req *authv1.RevokeOtherSessionsRequest) (*authv1.Empty, error) {
	if err := s.authSvc.RevokeOtherSessions(req.UserId, req.CurrentSessionId); err != nil {
		return nil, err
	}
	return &authv1.Empty{}, nil
}

func (s *Server) OIDCStatus(ctx context.Context, req *authv1.OIDCStatusRequest) (*authv1.OIDCStatusResponse, error) {
	if s.oidcSvc == nil || !s.oidcSvc.IsEnabled() {
		return &authv1.OIDCStatusResponse{Enabled: false}, nil
	}
	return &authv1.OIDCStatusResponse{Enabled: true, ProviderName: s.oidcSvc.ProviderName(), LoginUrl: "/api/v1/auth/oidc/login"}, nil
}

func (s *Server) OIDCExchange(ctx context.Context, req *authv1.OIDCExchangeRequest) (*authv1.LoginResponse, error) {
	result, err := s.oidcSvc.ExchangeLoginCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}
	return mapLoginResponse(result), nil
}

func (s *Server) ListUsers(ctx context.Context, req *authv1.ListUsersRequest) (*authv1.ListUsersResponse, error) {
	users, total, err := s.authSvc.ListUsers(req.OrgId, int(req.Page), int(req.PageSize), req.Keyword)
	if err != nil {
		return nil, err
	}
	resp := &authv1.ListUsersResponse{Users: make([]*authv1.User, 0, len(users)), Total: total}
	for _, user := range users {
		resp.Users = append(resp.Users, mapUser(user))
	}
	return resp, nil
}

func (s *Server) CreateUser(ctx context.Context, req *authv1.CreateUserRequest) (*authv1.User, error) {
	user, err := s.authSvc.CreateUser(req.OrgId, req.Username, req.Email, req.Phone, req.DisplayName, req.Password)
	if err != nil {
		return nil, err
	}
	return mapUser(*user), nil
}

func (s *Server) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.User, error) {
	user, err := s.authSvc.GetUser(req.Id)
	if err != nil {
		return nil, err
	}
	return mapUser(*user), nil
}

func (s *Server) UpdateUser(ctx context.Context, req *authv1.UpdateUserRequest) (*authv1.User, error) {
	user, err := s.authSvc.UpdateUser(req.Id, req.Email, req.Phone, req.DisplayName)
	if err != nil {
		return nil, err
	}
	return mapUser(*user), nil
}

func (s *Server) UpdateUserStatus(ctx context.Context, req *authv1.UpdateUserStatusRequest) (*authv1.User, error) {
	user, err := s.authSvc.UpdateUserStatus(req.Id, req.Status)
	if err != nil {
		return nil, err
	}
	return mapUser(*user), nil
}

func (s *Server) DeleteUser(ctx context.Context, req *authv1.DeleteUserRequest) (*authv1.Empty, error) {
	if err := s.authSvc.DeleteUser(req.Id); err != nil {
		return nil, err
	}
	return &authv1.Empty{}, nil
}

func (s *Server) ResetUserPassword(ctx context.Context, req *authv1.ResetUserPasswordRequest) (*authv1.User, error) {
	user, err := s.authSvc.ResetUserPassword(req.Id, req.NewPassword, req.MustChangePassword)
	if err != nil {
		return nil, err
	}
	return mapUser(*user), nil
}

func (s *Server) ListOrganizations(ctx context.Context, req *authv1.ListOrganizationsRequest) (*authv1.ListOrganizationsResponse, error) {
	orgs, err := s.authSvc.ListOrganizations()
	if err != nil {
		return nil, err
	}
	resp := &authv1.ListOrganizationsResponse{Organizations: make([]*authv1.Organization, 0, len(orgs))}
	for _, org := range orgs {
		resp.Organizations = append(resp.Organizations, mapOrganization(org))
	}
	return resp, nil
}

func (s *Server) CreateOrganization(ctx context.Context, req *authv1.CreateOrganizationRequest) (*authv1.Organization, error) {
	org, err := s.authSvc.CreateOrganization(req.Name, req.Code, req.Description)
	if err != nil {
		return nil, err
	}
	return mapOrganization(*org), nil
}

func (s *Server) UpdateOrganization(ctx context.Context, req *authv1.UpdateOrganizationRequest) (*authv1.Organization, error) {
	org, err := s.authSvc.UpdateOrganization(req.Id, req.Name, req.Description)
	if err != nil {
		return nil, err
	}
	return mapOrganization(*org), nil
}

func (s *Server) GetSystemConfig(ctx context.Context, req *authv1.GetSystemConfigRequest) (*authv1.SystemConfig, error) {
	cfg, err := s.authSvc.GetSystemConfig(req.OrgId)
	if err != nil {
		return nil, err
	}
	return mapSystemConfig(req.OrgId, cfg), nil
}

func (s *Server) UpdateSystemConfig(ctx context.Context, req *authv1.UpdateSystemConfigRequest) (*authv1.Empty, error) {
	if err := s.authSvc.UpdateSystemConfig(req.OrgId, unmapSystemConfig(req.Config)); err != nil {
		return nil, err
	}
	return &authv1.Empty{}, nil
}

func mapLoginResponse(result *service.LoginResult) *authv1.LoginResponse {
	return &authv1.LoginResponse{
		AccessToken: result.AccessToken, RefreshToken: result.RefreshToken, ExpiresIn: result.ExpiresIn,
		TokenType: "Bearer", SessionId: result.SessionID, Jti: result.JTI,
		User: &authv1.UserInfo{
			UserId: result.UserID, OrgId: result.OrgID, Username: result.Username,
			DisplayName: result.DisplayName, Email: result.Email, Roles: result.Roles,
			MustChangePassword: result.MustChangePassword, MfaEnabled: result.MFAEnabled,
		},
	}
}

func mapUser(user model.User) *authv1.User {
	return &authv1.User{
		Id: user.ID.String(), OrgId: user.OrgID.String(), Username: user.Username,
		Email: user.Email, Phone: user.Phone, DisplayName: user.DisplayName, Avatar: user.Avatar,
		Status: user.Status, Source: user.Source, FailedLoginAttempts: int32(user.FailedLoginAttempts),
		LockedUntil: toTimestamp(user.LockedUntil), PasswordChangedAt: toTimestamp(user.PasswordChangedAt),
		MustChangePassword: user.MustChangePassword, MfaEnabled: user.MFAEnabled,
		MfaConfirmedAt: toTimestamp(user.MFAConfirmedAt), LastLoginAt: toTimestamp(user.LastLoginAt),
		DeletedAt: toTimestamp(user.DeletedAt), CreatedAt: timestamppb.New(user.CreatedAt), UpdatedAt: timestamppb.New(user.UpdatedAt),
	}
}

func mapOrganization(org model.Organization) *authv1.Organization {
	return &authv1.Organization{
		Id: org.ID.String(), Name: org.Name, Code: org.Code, Description: org.Description,
		Logo: org.Logo, Status: org.Status, CreatedAt: timestamppb.New(org.CreatedAt), UpdatedAt: timestamppb.New(org.UpdatedAt),
	}
}

func mapSession(session model.Session) *authv1.Session {
	return &authv1.Session{
		Id: session.ID.String(), UserId: session.UserID.String(), OrgId: session.OrgID.String(),
		Status: session.Status, Ip: session.IP, UserAgent: session.UserAgent, DeviceName: session.DeviceName,
		LastSeenAt: timestamppb.New(session.LastSeenAt), ExpiresAt: timestamppb.New(session.ExpiresAt),
		RevokedAt: toTimestamp(session.RevokedAt), RevokedReason: session.RevokedReason,
		CreatedAt: timestamppb.New(session.CreatedAt), UpdatedAt: timestamppb.New(session.UpdatedAt),
	}
}

func mapSystemConfig(orgID string, cfg *service.SystemConfig) *authv1.SystemConfig {
	return &authv1.SystemConfig{Providers: []*authv1.AuthProvider{
		mapProvider(orgID, "ldap", "LDAP Authentication", cfg.LDAP.Enabled, cfg.LDAP),
		mapProvider(orgID, "oidc", "OIDC Authentication", cfg.OIDC.Enabled, cfg.OIDC),
	}}
}

func mapProvider(orgID, provider, name string, enabled bool, cfg any) *authv1.AuthProvider {
	data, _ := json.Marshal(cfg)
	return &authv1.AuthProvider{OrgId: orgID, Provider: provider, Name: name, ConfigJson: string(data), IsEnabled: enabled}
}

func unmapSystemConfig(cfg *authv1.SystemConfig) service.SystemConfig {
	out := service.SystemConfig{}
	if cfg == nil {
		return out
	}
	for _, provider := range cfg.Providers {
		switch provider.Provider {
		case "ldap":
			_ = json.Unmarshal([]byte(provider.ConfigJson), &out.LDAP)
			out.LDAP.Enabled = provider.IsEnabled
		case "oidc":
			_ = json.Unmarshal([]byte(provider.ConfigJson), &out.OIDC)
			out.OIDC.Enabled = provider.IsEnabled
		}
	}
	return out
}

func toTimestamp(value *time.Time) *timestamppb.Timestamp {
	if value == nil || value.IsZero() {
		return nil
	}
	return timestamppb.New(*value)
}
