package router

import (
	"context"

	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/client"
)

type recordingIAMRouterService struct {
	userID   string
	userCtx  client.UserContext
	roleReq  client.RoleRequest
	apiReq   client.APIPermissionRequest
	assigned client.AssignRolesRequest
}

func (s *recordingIAMRouterService) AssignUserRoles(ctx context.Context, userID string, req client.AssignRolesRequest, userCtx client.UserContext) error {
	s.userID = userID
	s.assigned = req
	s.userCtx = userCtx
	return nil
}

func (s *recordingIAMRouterService) GetUserRoles(ctx context.Context, userID string, userCtx client.UserContext) ([]map[string]any, error) {
	s.userID = userID
	s.userCtx = userCtx
	return []map[string]any{{"id": "r_1", "code": "admin"}}, nil
}

func (s *recordingIAMRouterService) GetUserPermissions(ctx context.Context, userID string, userCtx client.UserContext) ([]any, error) {
	s.userID = userID
	s.userCtx = userCtx
	return []any{"user:read"}, nil
}

func (s *recordingIAMRouterService) ListRoles(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userCtx = userCtx
	return []map[string]any{{"id": "r_1", "code": "admin"}}, nil
}

func (s *recordingIAMRouterService) CreateRole(ctx context.Context, req client.RoleRequest, userCtx client.UserContext) (map[string]any, error) {
	s.roleReq = req
	s.userCtx = userCtx
	return map[string]any{"id": "r_2", "code": req.Code}, nil
}

func (s *recordingIAMRouterService) GetRole(ctx context.Context, id string, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.userCtx = userCtx
	return map[string]any{"id": id, "code": "admin"}, nil
}

func (s *recordingIAMRouterService) UpdateRole(ctx context.Context, id string, req client.RoleRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.roleReq = req
	s.userCtx = userCtx
	return map[string]any{"id": id, "name": req.Name}, nil
}

func (s *recordingIAMRouterService) DeleteRole(ctx context.Context, id string, userCtx client.UserContext) error {
	s.userID = id
	s.userCtx = userCtx
	return nil
}

func (s *recordingIAMRouterService) AssignRolePermissions(ctx context.Context, roleID string, req client.AssignPermissionsRequest, userCtx client.UserContext) error {
	s.userID = roleID
	s.userCtx = userCtx
	return nil
}

func (s *recordingIAMRouterService) GetPermissionTree(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userCtx = userCtx
	return []map[string]any{{"code": "user:read"}}, nil
}

func (s *recordingIAMRouterService) ListAPIPermissions(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error) {
	s.userCtx = userCtx
	return []map[string]any{{"id": "api_1", "method": "GET"}}, nil
}

func (s *recordingIAMRouterService) CreateAPIPermission(ctx context.Context, req client.APIPermissionRequest, userCtx client.UserContext) (map[string]any, error) {
	s.apiReq = req
	s.userCtx = userCtx
	return map[string]any{"id": "api_2", "method": req.Method}, nil
}

func (s *recordingIAMRouterService) UpdateAPIPermission(ctx context.Context, id string, req client.APIPermissionRequest, userCtx client.UserContext) (map[string]any, error) {
	s.userID = id
	s.apiReq = req
	s.userCtx = userCtx
	return map[string]any{"id": id, "method": req.Method}, nil
}

func (s *recordingIAMRouterService) DeleteAPIPermission(ctx context.Context, id string, userCtx client.UserContext) error {
	s.userID = id
	s.userCtx = userCtx
	return nil
}
