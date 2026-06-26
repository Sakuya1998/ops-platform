package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/iam-svc/internal/service"
	"github.com/Sakuya1998/ops-platform/pkg/response"
)

type IAMHandler struct{ svc *service.IAMService }

func NewIAMHandler(svc *service.IAMService) *IAMHandler { return &IAMHandler{svc: svc} }

type permissionCheckRequest struct {
	UserID string `json:"user_id" binding:"required"`
	OrgID  string `json:"org_id" binding:"required"`
	Method string `json:"method" binding:"required"`
	Path   string `json:"path" binding:"required"`
}

type permissionCheckResponse struct {
	Allowed bool     `json:"allowed"`
	Reason  string   `json:"reason,omitempty"`
	Roles   []string `json:"roles,omitempty"`
}

type batchPermissionCheckRequest struct {
	UserID string                    `json:"user_id" binding:"required"`
	OrgID  string                    `json:"org_id" binding:"required"`
	Checks []service.PermissionCheck `json:"checks" binding:"required,min=1"`
}

type batchPermissionCheckResponse struct {
	Results []service.PermissionCheckResult `json:"results"`
}

type apiPermissionRequest struct {
	Method         string `json:"method" binding:"required"`
	PathPattern    string `json:"path_pattern" binding:"required"`
	PermissionCode string `json:"permission_code" binding:"required"`
	Description    string `json:"description"`
	Enabled        bool   `json:"enabled"`
}

func (h *IAMHandler) AssignUserRoles(c *gin.Context) {
	var req struct {
		RoleIDs []string `json:"role_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.AssignRoles(c.Param("id"), req.RoleIDs); err != nil {
		respondServiceError(c, err)
		return
	}
	response.OK(c)
}

func (h *IAMHandler) GetUserRoles(c *gin.Context) {
	roles, err := h.svc.GetUserRoles(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	response.Success(c, roles)
}

func (h *IAMHandler) GetUserPermissions(c *gin.Context) {
	permissions, err := h.svc.GetUserPermissions(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	response.Success(c, permissions)
}

func (h *IAMHandler) ListRoles(c *gin.Context) {
	orgID := c.GetHeader("X-Org-Id")
	roles, err := h.svc.ListRoles(orgID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	response.Success(c, roles)
}

func (h *IAMHandler) CreateRole(c *gin.Context) {
	var req struct {
		OrgID       string `json:"org_id" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	role, err := h.svc.CreateRole(req.OrgID, req.Name, req.Code, req.Description)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	response.Created(c, role)
}

func (h *IAMHandler) GetRole(c *gin.Context) {
	role, err := h.svc.GetRole(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	response.Success(c, role)
}

func (h *IAMHandler) UpdateRole(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	role, err := h.svc.UpdateRole(c.Param("id"), req.Name, req.Description)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	response.Success(c, role)
}

func (h *IAMHandler) DeleteRole(c *gin.Context) {
	if err := h.svc.DeleteRole(c.Param("id")); err != nil {
		respondServiceError(c, err)
		return
	}
	response.OK(c)
}

func (h *IAMHandler) AssignPermissions(c *gin.Context) {
	var req struct {
		PermissionIDs []string `json:"permission_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.AssignPermissions(c.Param("id"), req.PermissionIDs); err != nil {
		respondServiceError(c, err)
		return
	}
	response.OK(c)
}

func (h *IAMHandler) GetPermissionTree(c *gin.Context) {
	tree, err := h.svc.GetPermissionTree()
	if err != nil {
		respondServiceError(c, err)
		return
	}
	response.Success(c, tree)
}

func (h *IAMHandler) ListAPIPermissions(c *gin.Context) {
	permissions, err := h.svc.ListAPIPermissions()
	if err != nil {
		respondServiceError(c, err)
		return
	}
	response.Success(c, permissions)
}

func (h *IAMHandler) CreateAPIPermission(c *gin.Context) {
	var req apiPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	permission, err := h.svc.CreateAPIPermission(service.CreateAPIPermissionInput{
		Method: req.Method, PathPattern: req.PathPattern, PermissionCode: req.PermissionCode,
		Description: req.Description, Enabled: req.Enabled,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}
	response.Created(c, permission)
}

func (h *IAMHandler) UpdateAPIPermission(c *gin.Context) {
	var req apiPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	permission, err := h.svc.UpdateAPIPermission(c.Param("id"), service.UpdateAPIPermissionInput{
		Method: req.Method, PathPattern: req.PathPattern, PermissionCode: req.PermissionCode,
		Description: req.Description, Enabled: req.Enabled,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}
	response.Success(c, permission)
}

func (h *IAMHandler) DeleteAPIPermission(c *gin.Context) {
	if err := h.svc.DeleteAPIPermission(c.Param("id")); err != nil {
		respondServiceError(c, err)
		return
	}
	response.OK(c)
}

func (h *IAMHandler) CheckPermission(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	orgID := c.GetHeader("X-Org-Id")
	username := c.GetHeader("X-User-Name")
	roles := c.GetHeader("X-User-Roles")
	sessionID := c.GetHeader("X-Session-Id")
	if userID == "" {
		userCtx, err := h.svc.ResolveToken(c.GetHeader("Authorization"))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"allowed": false, "reason": "invalid token"})
			return
		}
		userID = userCtx.UserID
		orgID = userCtx.OrgID
		sessionID = userCtx.SessionID
		roles = strings.Join(userCtx.Roles, ",")
	}
	method := c.Request.Method
	path := c.Request.URL.Path
	if originalMethod := firstHeader(c, "X-Original-Method", "X-Forwarded-Method", "X-Request-Method"); originalMethod != "" {
		method = originalMethod
	}
	if originalPath := firstHeader(c, "X-Original-URI", "X-Original-Uri", "X-Forwarded-URI", "X-Forwarded-Uri", "X-Request-URI", "X-Request-Uri"); originalPath != "" {
		path = originalPath
	}
	allowed, reason, userCtx := h.svc.CheckPermission(userID, orgID, method, path)
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"allowed": false, "reason": reason})
		return
	}
	if userCtx != nil {
		if roles == "" {
			roles = strings.Join(userCtx.Roles, ",")
		}
		if sessionID == "" {
			sessionID = userCtx.SessionID
		}
	}
	c.Header("X-User-Id", userID)
	c.Header("X-Org-Id", orgID)
	c.Header("X-Session-Id", sessionID)
	c.Header("X-User-Name", username)
	c.Header("X-User-Roles", roles)
	c.JSON(http.StatusOK, gin.H{"allowed": true})
}

func (h *IAMHandler) CheckPermissionAPI(c *gin.Context) {
	var req permissionCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	allowed, reason, userCtx := h.svc.CheckPermission(req.UserID, req.OrgID, req.Method, req.Path)
	resp := permissionCheckResponse{Allowed: allowed, Reason: reason}
	if userCtx != nil {
		resp.Roles = userCtx.Roles
	}
	response.Success(c, resp)
}

func (h *IAMHandler) BatchCheckPermissionAPI(c *gin.Context) {
	var req batchPermissionCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	results, err := h.svc.BatchCheckPermission(req.UserID, req.OrgID, req.Checks)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	response.Success(c, batchPermissionCheckResponse{Results: results})
}

func respondServiceError(c *gin.Context, err error) {
	if kind, ok := service.ErrorKindOf(err); ok {
		switch kind {
		case service.ErrInvalidArgument:
			response.BadRequest(c, err.Error())
		case service.ErrNotFound:
			response.NotFound(c, err.Error())
		case service.ErrConflict:
			response.Error(c, http.StatusConflict, err.Error())
		case service.ErrForbidden:
			response.Forbidden(c, err.Error())
		default:
			response.InternalError(c, err.Error())
		}
		return
	}
	response.InternalError(c, err.Error())
}

func firstHeader(c *gin.Context, names ...string) string {
	for _, name := range names {
		if value := c.GetHeader(name); value != "" {
			return value
		}
	}
	return ""
}
