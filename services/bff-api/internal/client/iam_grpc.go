package client

import (
	"context"

	iamv1 "github.com/ops-platform/pkg/proto/iam/v1"
	"google.golang.org/grpc"
)

type IAMGRPCClient struct {
	client iamv1.IAMServiceClient
}

func NewIAMGRPCClient(conn grpc.ClientConnInterface) *IAMGRPCClient {
	return &IAMGRPCClient{client: iamv1.NewIAMServiceClient(conn)}
}

func (c *IAMGRPCClient) CheckPermission(ctx context.Context, req CheckPermissionRequest) (CheckPermissionResult, error) {
	resp, err := c.client.CheckPermission(ctx, &iamv1.CheckPermissionRequest{
		UserId: req.UserID, OrgId: req.OrgID, Method: req.Method, Path: req.Path,
	})
	if err != nil {
		return CheckPermissionResult{}, err
	}
	return CheckPermissionResult{Allowed: resp.Allowed, Reason: resp.Reason, Roles: resp.Roles}, nil
}

func (c *IAMGRPCClient) AssignUserRoles(ctx context.Context, userID string, req AssignRolesRequest, userCtx UserContext) error {
	_, err := c.client.AssignUserRoles(ctx, &iamv1.AssignUserRolesRequest{UserId: userID, RoleIds: req.RoleIDs})
	return err
}

func (c *IAMGRPCClient) GetUserRoles(ctx context.Context, userID string, userCtx UserContext) ([]map[string]any, error) {
	resp, err := c.client.GetUserRoles(ctx, &iamv1.GetUserRolesRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	return rolesToMaps(resp.Roles), nil
}

func (c *IAMGRPCClient) GetUserPermissions(ctx context.Context, userID string, userCtx UserContext) ([]any, error) {
	resp, err := c.client.GetUserPermissions(ctx, &iamv1.GetUserPermissionsRequest{UserId: userID})
	if err != nil {
		return nil, err
	}
	result := make([]any, 0, len(resp.Codes))
	for _, code := range resp.Codes {
		result = append(result, code)
	}
	return result, nil
}

func (c *IAMGRPCClient) ListRoles(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	resp, err := c.client.ListRoles(ctx, &iamv1.ListRolesRequest{OrgId: userCtx.OrgID})
	if err != nil {
		return nil, err
	}
	return rolesToMaps(resp.Roles), nil
}

func (c *IAMGRPCClient) CreateRole(ctx context.Context, req RoleRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.CreateRole(ctx, &iamv1.CreateRoleRequest{
		OrgId: firstNonEmpty(req.OrgID, userCtx.OrgID), Name: req.Name, Code: req.Code, Description: req.Description,
	})
	if err != nil {
		return nil, err
	}
	return roleToMap(resp), nil
}

func (c *IAMGRPCClient) GetRole(ctx context.Context, id string, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.GetRole(ctx, &iamv1.GetRoleRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return roleToMap(resp), nil
}

func (c *IAMGRPCClient) UpdateRole(ctx context.Context, id string, req RoleRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.UpdateRole(ctx, &iamv1.UpdateRoleRequest{Id: id, Name: req.Name, Description: req.Description})
	if err != nil {
		return nil, err
	}
	return roleToMap(resp), nil
}

func (c *IAMGRPCClient) DeleteRole(ctx context.Context, id string, userCtx UserContext) error {
	_, err := c.client.DeleteRole(ctx, &iamv1.DeleteRoleRequest{Id: id})
	return err
}

func (c *IAMGRPCClient) AssignRolePermissions(ctx context.Context, roleID string, req AssignPermissionsRequest, userCtx UserContext) error {
	_, err := c.client.AssignPermissions(ctx, &iamv1.AssignPermissionsRequest{RoleId: roleID, PermissionIds: req.PermissionIDs})
	return err
}

func (c *IAMGRPCClient) GetPermissionTree(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	resp, err := c.client.GetPermissionTree(ctx, &iamv1.Empty{})
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(resp.Permissions))
	for _, permission := range resp.Permissions {
		result = append(result, permissionToMap(permission))
	}
	return result, nil
}

func (c *IAMGRPCClient) ListAPIPermissions(ctx context.Context, userCtx UserContext) ([]map[string]any, error) {
	resp, err := c.client.ListAPIPermissions(ctx, &iamv1.ListAPIPermissionsRequest{})
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(resp.ApiPermissions))
	for _, permission := range resp.ApiPermissions {
		result = append(result, apiPermissionToMap(permission))
	}
	return result, nil
}

func (c *IAMGRPCClient) CreateAPIPermission(ctx context.Context, req APIPermissionRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.CreateAPIPermission(ctx, &iamv1.CreateAPIPermissionRequest{
		Method: req.Method, PathPattern: req.PathPattern, PermissionCode: req.PermissionCode,
		Description: req.Description, Enabled: req.Enabled,
	})
	if err != nil {
		return nil, err
	}
	return apiPermissionToMap(resp), nil
}

func (c *IAMGRPCClient) UpdateAPIPermission(ctx context.Context, id string, req APIPermissionRequest, userCtx UserContext) (map[string]any, error) {
	resp, err := c.client.UpdateAPIPermission(ctx, &iamv1.UpdateAPIPermissionRequest{
		Id: id, Method: req.Method, PathPattern: req.PathPattern, PermissionCode: req.PermissionCode,
		Description: req.Description, Enabled: req.Enabled,
	})
	if err != nil {
		return nil, err
	}
	return apiPermissionToMap(resp), nil
}

func (c *IAMGRPCClient) DeleteAPIPermission(ctx context.Context, id string, userCtx UserContext) error {
	_, err := c.client.DeleteAPIPermission(ctx, &iamv1.DeleteAPIPermissionRequest{Id: id})
	return err
}

func rolesToMaps(roles []*iamv1.Role) []map[string]any {
	result := make([]map[string]any, 0, len(roles))
	for _, role := range roles {
		result = append(result, roleToMap(role))
	}
	return result
}

func roleToMap(role *iamv1.Role) map[string]any {
	if role == nil {
		return map[string]any{}
	}
	result := map[string]any{
		"id":          role.Id,
		"org_id":      role.OrgId,
		"name":        role.Name,
		"code":        role.Code,
		"description": role.Description,
		"is_system":   role.IsSystem,
	}
	if role.CreatedAt != nil {
		result["created_at"] = role.CreatedAt.AsTime()
	}
	if role.UpdatedAt != nil {
		result["updated_at"] = role.UpdatedAt.AsTime()
	}
	return result
}

func permissionToMap(permission *iamv1.Permission) map[string]any {
	if permission == nil {
		return map[string]any{}
	}
	children := make([]map[string]any, 0, len(permission.Children))
	for _, child := range permission.Children {
		children = append(children, permissionToMap(child))
	}
	return map[string]any{
		"id":        permission.Id,
		"parent_id": permission.ParentId,
		"name":      permission.Name,
		"code":      permission.Code,
		"resource":  permission.Resource,
		"action":    permission.Action,
		"type":      permission.Type,
		"sort":      permission.Sort,
		"children":  children,
	}
}

func apiPermissionToMap(permission *iamv1.APIPermission) map[string]any {
	if permission == nil {
		return map[string]any{}
	}
	result := map[string]any{
		"id":              permission.Id,
		"method":          permission.Method,
		"path_pattern":    permission.PathPattern,
		"permission_code": permission.PermissionCode,
		"description":     permission.Description,
		"enabled":         permission.Enabled,
	}
	if permission.CreatedAt != nil {
		result["created_at"] = permission.CreatedAt.AsTime()
	}
	if permission.UpdatedAt != nil {
		result["updated_at"] = permission.UpdatedAt.AsTime()
	}
	return result
}
