package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/model"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/service"
	iamv1 "github.com/Sakuya1998/ops-platform/pkg/proto/iam/v1"
)

type Server struct {
	iamv1.UnimplementedIAMServiceServer
	svc *service.IAMService
}

func NewServer(svc *service.IAMService) *Server {
	return &Server{svc: svc}
}

func (s *Server) CreateRole(ctx context.Context, req *iamv1.CreateRoleRequest) (*iamv1.Role, error) {
	role, err := s.svc.CreateRole(req.OrgId, req.Name, req.Code, req.Description)
	if err != nil {
		return nil, err
	}
	return mapRole(*role), nil
}

func (s *Server) GetRole(ctx context.Context, req *iamv1.GetRoleRequest) (*iamv1.Role, error) {
	role, err := s.svc.GetRole(req.Id)
	if err != nil {
		return nil, err
	}
	return mapRole(*role), nil
}

func (s *Server) ListRoles(ctx context.Context, req *iamv1.ListRolesRequest) (*iamv1.ListRolesResponse, error) {
	roles, err := s.svc.ListRoles(req.OrgId)
	if err != nil {
		return nil, err
	}
	resp := &iamv1.ListRolesResponse{Roles: make([]*iamv1.Role, 0, len(roles))}
	for _, role := range roles {
		resp.Roles = append(resp.Roles, mapRole(role))
	}
	return resp, nil
}

func (s *Server) UpdateRole(ctx context.Context, req *iamv1.UpdateRoleRequest) (*iamv1.Role, error) {
	role, err := s.svc.UpdateRole(req.Id, req.Name, req.Description)
	if err != nil {
		return nil, err
	}
	return mapRole(*role), nil
}

func (s *Server) DeleteRole(ctx context.Context, req *iamv1.DeleteRoleRequest) (*iamv1.Empty, error) {
	if err := s.svc.DeleteRole(req.Id); err != nil {
		return nil, err
	}
	return &iamv1.Empty{}, nil
}

func (s *Server) AssignPermissions(ctx context.Context, req *iamv1.AssignPermissionsRequest) (*iamv1.Empty, error) {
	if err := s.svc.AssignPermissions(req.RoleId, req.PermissionIds); err != nil {
		return nil, err
	}
	return &iamv1.Empty{}, nil
}

func (s *Server) GetPermissionTree(ctx context.Context, req *iamv1.Empty) (*iamv1.PermissionTreeResponse, error) {
	permissions, err := s.svc.GetPermissionTree()
	if err != nil {
		return nil, err
	}
	resp := &iamv1.PermissionTreeResponse{Permissions: make([]*iamv1.Permission, 0, len(permissions))}
	for _, permission := range permissions {
		resp.Permissions = append(resp.Permissions, mapPermission(permission))
	}
	return resp, nil
}

func (s *Server) AssignUserRoles(ctx context.Context, req *iamv1.AssignUserRolesRequest) (*iamv1.Empty, error) {
	if err := s.svc.AssignRoles(req.UserId, req.RoleIds); err != nil {
		return nil, err
	}
	return &iamv1.Empty{}, nil
}

func (s *Server) CheckPermission(ctx context.Context, req *iamv1.CheckPermissionRequest) (*iamv1.CheckPermissionResponse, error) {
	allowed, reason, userCtx := s.svc.CheckPermission(req.UserId, req.OrgId, req.Method, req.Path)
	resp := &iamv1.CheckPermissionResponse{Allowed: allowed, Reason: reason}
	if userCtx != nil {
		resp.Roles = userCtx.Roles
	}
	return resp, nil
}

func (s *Server) BatchCheckPermission(ctx context.Context, req *iamv1.BatchCheckPermissionRequest) (*iamv1.BatchCheckPermissionResponse, error) {
	checks := make([]service.PermissionCheck, 0, len(req.Checks))
	for _, check := range req.Checks {
		checks = append(checks, service.PermissionCheck{Method: check.Method, Path: check.Path})
	}
	results, err := s.svc.BatchCheckPermission(req.UserId, req.OrgId, checks)
	if err != nil {
		return nil, err
	}
	resp := &iamv1.BatchCheckPermissionResponse{Results: make([]*iamv1.PermissionCheckResult, 0, len(results))}
	for _, result := range results {
		resp.Results = append(resp.Results, &iamv1.PermissionCheckResult{
			Method:             result.Method,
			Path:               result.Path,
			Allowed:            result.Allowed,
			Reason:             result.Reason,
			RequiredPermission: result.RequiredPermission,
		})
	}
	return resp, nil
}

func (s *Server) GetUserRoles(ctx context.Context, req *iamv1.GetUserRolesRequest) (*iamv1.GetUserRolesResponse, error) {
	roles, err := s.svc.GetUserRoles(req.UserId)
	if err != nil {
		return nil, err
	}
	resp := &iamv1.GetUserRolesResponse{Roles: make([]*iamv1.Role, 0, len(roles))}
	for _, role := range roles {
		resp.Roles = append(resp.Roles, mapRole(role))
	}
	return resp, nil
}

func (s *Server) GetUserPermissions(ctx context.Context, req *iamv1.GetUserPermissionsRequest) (*iamv1.GetUserPermissionsResponse, error) {
	codes, err := s.svc.GetUserPermissions(req.UserId)
	if err != nil {
		return nil, err
	}
	return &iamv1.GetUserPermissionsResponse{Codes: codes}, nil
}

func (s *Server) ListAPIPermissions(ctx context.Context, req *iamv1.ListAPIPermissionsRequest) (*iamv1.ListAPIPermissionsResponse, error) {
	permissions, err := s.svc.ListAPIPermissions()
	if err != nil {
		return nil, err
	}
	resp := &iamv1.ListAPIPermissionsResponse{ApiPermissions: make([]*iamv1.APIPermission, 0, len(permissions))}
	for _, permission := range permissions {
		resp.ApiPermissions = append(resp.ApiPermissions, mapAPIPermission(permission))
	}
	return resp, nil
}

func (s *Server) CreateAPIPermission(ctx context.Context, req *iamv1.CreateAPIPermissionRequest) (*iamv1.APIPermission, error) {
	permission, err := s.svc.CreateAPIPermission(service.CreateAPIPermissionInput{
		Method: req.Method, PathPattern: req.PathPattern, PermissionCode: req.PermissionCode,
		Description: req.Description, Enabled: req.Enabled,
	})
	if err != nil {
		return nil, err
	}
	return mapAPIPermission(*permission), nil
}

func (s *Server) UpdateAPIPermission(ctx context.Context, req *iamv1.UpdateAPIPermissionRequest) (*iamv1.APIPermission, error) {
	permission, err := s.svc.UpdateAPIPermission(req.Id, service.UpdateAPIPermissionInput{
		Method: req.Method, PathPattern: req.PathPattern, PermissionCode: req.PermissionCode,
		Description: req.Description, Enabled: req.Enabled,
	})
	if err != nil {
		return nil, err
	}
	return mapAPIPermission(*permission), nil
}

func (s *Server) DeleteAPIPermission(ctx context.Context, req *iamv1.DeleteAPIPermissionRequest) (*iamv1.Empty, error) {
	if err := s.svc.DeleteAPIPermission(req.Id); err != nil {
		return nil, err
	}
	return &iamv1.Empty{}, nil
}

func mapRole(role model.Role) *iamv1.Role {
	return &iamv1.Role{
		Id:          role.ID.String(),
		OrgId:       role.OrgID.String(),
		Name:        role.Name,
		Code:        role.Code,
		Description: role.Description,
		IsSystem:    role.IsSystem,
	}
}

func mapPermission(permission *model.Permission) *iamv1.Permission {
	if permission == nil {
		return nil
	}
	parentID := ""
	if permission.ParentID != nil && *permission.ParentID != uuid.Nil {
		parentID = permission.ParentID.String()
	}
	out := &iamv1.Permission{
		Id:       permission.ID.String(),
		ParentId: parentID,
		Name:     permission.Name,
		Code:     permission.Code,
		Resource: permission.Resource,
		Action:   permission.Action,
		Type:     permission.Type,
		Sort:     int32(permission.Sort),
		Children: make([]*iamv1.Permission, 0, len(permission.Children)),
	}
	for _, child := range permission.Children {
		out.Children = append(out.Children, mapPermission(child))
	}
	return out
}

func mapAPIPermission(permission model.APIPermission) *iamv1.APIPermission {
	return &iamv1.APIPermission{
		Id:             permission.ID.String(),
		Method:         permission.Method,
		PathPattern:    permission.PathPattern,
		PermissionCode: permission.PermissionCode,
		Description:    permission.Description,
		Enabled:        permission.Enabled,
	}
}
