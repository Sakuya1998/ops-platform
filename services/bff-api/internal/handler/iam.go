package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/client"
	"github.com/Sakuya1998/ops-platform/pkg/response"
)

type IAMService interface {
	AssignUserRoles(ctx context.Context, userID string, req client.AssignRolesRequest, userCtx client.UserContext) error
	GetUserRoles(ctx context.Context, userID string, userCtx client.UserContext) ([]map[string]any, error)
	GetUserPermissions(ctx context.Context, userID string, userCtx client.UserContext) ([]any, error)
	ListRoles(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error)
	CreateRole(ctx context.Context, req client.RoleRequest, userCtx client.UserContext) (map[string]any, error)
	GetRole(ctx context.Context, id string, userCtx client.UserContext) (map[string]any, error)
	UpdateRole(ctx context.Context, id string, req client.RoleRequest, userCtx client.UserContext) (map[string]any, error)
	DeleteRole(ctx context.Context, id string, userCtx client.UserContext) error
	AssignRolePermissions(ctx context.Context, roleID string, req client.AssignPermissionsRequest, userCtx client.UserContext) error
	GetPermissionTree(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error)
	ListAPIPermissions(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error)
	CreateAPIPermission(ctx context.Context, req client.APIPermissionRequest, userCtx client.UserContext) (map[string]any, error)
	UpdateAPIPermission(ctx context.Context, id string, req client.APIPermissionRequest, userCtx client.UserContext) (map[string]any, error)
	DeleteAPIPermission(ctx context.Context, id string, userCtx client.UserContext) error
}

type IAMHandler struct {
	service IAMService
}

func NewIAMHandler(service IAMService) *IAMHandler {
	return &IAMHandler{service: service}
}

func (h *IAMHandler) RegisterUserRoutes(r gin.IRouter) {
	r.GET("/:id/roles", h.GetUserRoles)
	r.PUT("/:id/roles", h.AssignUserRoles)
	r.GET("/:id/permissions", h.GetUserPermissions)
}

func (h *IAMHandler) RegisterRoleRoutes(r gin.IRouter) {
	r.GET("", h.ListRoles)
	r.POST("", h.CreateRole)
	r.GET("/:id", h.GetRole)
	r.PUT("/:id", h.UpdateRole)
	r.DELETE("/:id", h.DeleteRole)
	r.PUT("/:id/permissions", h.AssignRolePermissions)
}

func (h *IAMHandler) RegisterPermissionRoutes(r gin.IRouter) {
	r.GET("/permissions", h.GetPermissionTree)
}

func (h *IAMHandler) RegisterAPIPermissionRoutes(r gin.IRouter) {
	r.GET("", h.ListAPIPermissions)
	r.POST("", h.CreateAPIPermission)
	r.PUT("/:id", h.UpdateAPIPermission)
	r.DELETE("/:id", h.DeleteAPIPermission)
}

func (h *IAMHandler) AssignUserRoles(c *gin.Context) {
	var req client.AssignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.service.AssignUserRoles(c.Request.Context(), c.Param("id"), req, userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *IAMHandler) GetUserRoles(c *gin.Context) {
	result, err := h.service.GetUserRoles(c.Request.Context(), c.Param("id"), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *IAMHandler) GetUserPermissions(c *gin.Context) {
	result, err := h.service.GetUserPermissions(c.Request.Context(), c.Param("id"), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *IAMHandler) ListRoles(c *gin.Context) {
	result, err := h.service.ListRoles(c.Request.Context(), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *IAMHandler) CreateRole(c *gin.Context) {
	var req client.RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.CreateRole(c.Request.Context(), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, result)
}

func (h *IAMHandler) GetRole(c *gin.Context) {
	result, err := h.service.GetRole(c.Request.Context(), c.Param("id"), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *IAMHandler) UpdateRole(c *gin.Context) {
	var req client.RoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.UpdateRole(c.Request.Context(), c.Param("id"), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *IAMHandler) DeleteRole(c *gin.Context) {
	if err := h.service.DeleteRole(c.Request.Context(), c.Param("id"), userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *IAMHandler) AssignRolePermissions(c *gin.Context) {
	var req client.AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.service.AssignRolePermissions(c.Request.Context(), c.Param("id"), req, userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *IAMHandler) GetPermissionTree(c *gin.Context) {
	result, err := h.service.GetPermissionTree(c.Request.Context(), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *IAMHandler) ListAPIPermissions(c *gin.Context) {
	result, err := h.service.ListAPIPermissions(c.Request.Context(), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *IAMHandler) CreateAPIPermission(c *gin.Context) {
	var req client.APIPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.CreateAPIPermission(c.Request.Context(), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, result)
}

func (h *IAMHandler) UpdateAPIPermission(c *gin.Context) {
	var req client.APIPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.UpdateAPIPermission(c.Request.Context(), c.Param("id"), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *IAMHandler) DeleteAPIPermission(c *gin.Context) {
	if err := h.service.DeleteAPIPermission(c.Request.Context(), c.Param("id"), userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}
