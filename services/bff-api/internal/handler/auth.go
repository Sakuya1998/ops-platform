package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/client"
	"github.com/Sakuya1998/ops-platform/pkg/response"
)

type AuthService interface {
	Login(ctx context.Context, req client.LoginRequest, meta client.RequestMeta) (client.LoginResponse, error)
	Refresh(ctx context.Context, req client.RefreshRequest) (client.LoginResponse, error)
	VerifyToken(ctx context.Context, authorization string) (client.TokenContext, error)
	Logout(ctx context.Context, authorization string, userCtx client.UserContext) error
	GetCurrentUser(ctx context.Context, userCtx client.UserContext) (map[string]any, error)
	ListUsers(ctx context.Context, req client.ListUsersRequest, userCtx client.UserContext) ([]map[string]any, int64, error)
	CreateUser(ctx context.Context, req client.CreateUserRequest, userCtx client.UserContext) (map[string]any, error)
	GetUser(ctx context.Context, id string, userCtx client.UserContext) (map[string]any, error)
	UpdateUser(ctx context.Context, id string, req client.UpdateUserRequest, userCtx client.UserContext) (map[string]any, error)
	DeleteUser(ctx context.Context, id string, userCtx client.UserContext) error
	UpdateUserStatus(ctx context.Context, id string, req client.UpdateUserStatusRequest, userCtx client.UserContext) (map[string]any, error)
	ResetUserPassword(ctx context.Context, id string, req client.ResetUserPasswordRequest, userCtx client.UserContext) (map[string]any, error)
	ListOrganizations(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error)
	CreateOrganization(ctx context.Context, req client.OrganizationRequest, userCtx client.UserContext) (map[string]any, error)
	UpdateOrganization(ctx context.Context, id string, req client.OrganizationRequest, userCtx client.UserContext) (map[string]any, error)
	GetSystemConfig(ctx context.Context, userCtx client.UserContext) (map[string]any, error)
	UpdateSystemConfig(ctx context.Context, req map[string]any, userCtx client.UserContext) error
	ChangePassword(ctx context.Context, req client.ChangePasswordRequest, userCtx client.UserContext) error
	SetupMFA(ctx context.Context, userCtx client.UserContext) (map[string]any, error)
	ConfirmMFA(ctx context.Context, req client.MFAConfirmRequest, userCtx client.UserContext) (map[string]any, error)
	DisableMFA(ctx context.Context, req client.MFACodeRequest, userCtx client.UserContext) error
	RegenerateMFARecoveryCodes(ctx context.Context, req client.MFACodeRequest, userCtx client.UserContext) (map[string]any, error)
	ListSessions(ctx context.Context, userCtx client.UserContext) ([]map[string]any, error)
	RevokeSession(ctx context.Context, sessionID string, userCtx client.UserContext) error
	RevokeOtherSessions(ctx context.Context, userCtx client.UserContext) error
	OIDCLogin(ctx context.Context) (string, error)
	OIDCCallback(ctx context.Context, req client.OIDCCallbackRequest) (string, error)
	OIDCStatus(ctx context.Context) (map[string]any, error)
	OIDCExchange(ctx context.Context, req client.OIDCExchangeRequest) (client.LoginResponse, error)
}

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(r gin.IRouter) {
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/token/verify", h.VerifyToken)
	r.POST("/logout", h.Logout)
	r.GET("/me", h.GetCurrentUser)
	r.PUT("/me/password", h.ChangePassword)
	r.POST("/me/mfa/setup", h.SetupMFA)
	r.POST("/me/mfa/confirm", h.ConfirmMFA)
	r.POST("/me/mfa/recovery-codes", h.RegenerateMFARecoveryCodes)
	r.DELETE("/me/mfa", h.DisableMFA)
	r.GET("/sessions", h.ListSessions)
	r.DELETE("/sessions/:id", h.RevokeSession)
	r.DELETE("/sessions", h.RevokeOtherSessions)
	r.GET("/oidc/login", h.OIDCLogin)
	r.GET("/oidc/callback", h.OIDCCallback)
	r.GET("/oidc/status", h.OIDCStatus)
	r.POST("/oidc/exchange", h.OIDCExchange)
}

func (h *AuthHandler) RegisterUserRoutes(r gin.IRouter) {
	r.GET("", h.ListUsers)
	r.POST("", h.CreateUser)
	r.GET("/:id", h.GetUser)
	r.PUT("/:id", h.UpdateUser)
	r.DELETE("/:id", h.DeleteUser)
	r.PUT("/:id/status", h.UpdateUserStatus)
	r.PUT("/:id/password/reset", h.ResetUserPassword)
}

func (h *AuthHandler) RegisterOrganizationRoutes(r gin.IRouter) {
	r.GET("", h.ListOrganizations)
	r.POST("", h.CreateOrganization)
	r.PUT("/:id", h.UpdateOrganization)
}

func (h *AuthHandler) RegisterSystemRoutes(r gin.IRouter) {
	r.GET("/config", h.GetSystemConfig)
	r.PUT("/config", h.UpdateSystemConfig)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req client.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.Login(c.Request.Context(), req, client.RequestMeta{
		UserAgent:  c.GetHeader("User-Agent"),
		DeviceName: c.GetHeader("X-Device-Name"),
		RequestID:  requestID(c),
	})
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req client.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.Refresh(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) VerifyToken(c *gin.Context) {
	result, err := h.service.VerifyToken(c.Request.Context(), c.GetHeader("Authorization"))
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if err := h.service.Logout(c.Request.Context(), c.GetHeader("Authorization"), userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	result, err := h.service.GetCurrentUser(c.Request.Context(), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	users, total, err := h.service.ListUsers(c.Request.Context(), client.ListUsersRequest{
		OrgID:    c.Query("org_id"),
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 20),
		Keyword:  c.Query("keyword"),
	}, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.SuccessWithTotal(c, users, total)
}

func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req client.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.CreateUser(c.Request.Context(), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, result)
}

func (h *AuthHandler) GetUser(c *gin.Context) {
	result, err := h.service.GetUser(c.Request.Context(), c.Param("id"), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) UpdateUser(c *gin.Context) {
	var req client.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.UpdateUser(c.Request.Context(), c.Param("id"), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) DeleteUser(c *gin.Context) {
	if err := h.service.DeleteUser(c.Request.Context(), c.Param("id"), userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) UpdateUserStatus(c *gin.Context) {
	var req client.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.UpdateUserStatus(c.Request.Context(), c.Param("id"), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) ResetUserPassword(c *gin.Context) {
	var req client.ResetUserPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.ResetUserPassword(c.Request.Context(), c.Param("id"), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) ListOrganizations(c *gin.Context) {
	result, err := h.service.ListOrganizations(c.Request.Context(), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) CreateOrganization(c *gin.Context) {
	var req client.OrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.CreateOrganization(c.Request.Context(), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, result)
}

func (h *AuthHandler) UpdateOrganization(c *gin.Context) {
	var req client.OrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.UpdateOrganization(c.Request.Context(), c.Param("id"), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) GetSystemConfig(c *gin.Context) {
	result, err := h.service.GetSystemConfig(c.Request.Context(), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) UpdateSystemConfig(c *gin.Context) {
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.service.UpdateSystemConfig(c.Request.Context(), req, userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req client.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), req, userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) SetupMFA(c *gin.Context) {
	result, err := h.service.SetupMFA(c.Request.Context(), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) ConfirmMFA(c *gin.Context) {
	var req client.MFAConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.ConfirmMFA(c.Request.Context(), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) DisableMFA(c *gin.Context) {
	var req client.MFACodeRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.service.DisableMFA(c.Request.Context(), req, userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) RegenerateMFARecoveryCodes(c *gin.Context) {
	var req client.MFACodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.RegenerateMFARecoveryCodes(c.Request.Context(), req, userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) ListSessions(c *gin.Context) {
	result, err := h.service.ListSessions(c.Request.Context(), userContext(c))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) RevokeSession(c *gin.Context) {
	if err := h.service.RevokeSession(c.Request.Context(), c.Param("id"), userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) RevokeOtherSessions(c *gin.Context) {
	if err := h.service.RevokeOtherSessions(c.Request.Context(), userContext(c)); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) OIDCLogin(c *gin.Context) {
	location, err := h.service.OIDCLogin(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.Redirect(http.StatusFound, location)
}

func (h *AuthHandler) OIDCCallback(c *gin.Context) {
	location, err := h.service.OIDCCallback(c.Request.Context(), client.OIDCCallbackRequest{
		Code:  c.Query("code"),
		State: c.Query("state"),
	})
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.Redirect(http.StatusFound, location)
}

func (h *AuthHandler) OIDCStatus(c *gin.Context) {
	result, err := h.service.OIDCStatus(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) OIDCExchange(c *gin.Context) {
	var req client.OIDCExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.OIDCExchange(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

func requestID(c *gin.Context) string {
	if id := c.GetHeader("X-Request-Id"); id != "" {
		return id
	}
	return c.GetHeader("X-Request-ID")
}

func userContext(c *gin.Context) client.UserContext {
	return client.UserContext{
		UserID:    c.GetHeader("X-User-Id"),
		OrgID:     c.GetHeader("X-Org-Id"),
		SessionID: c.GetHeader("X-Session-Id"),
	}
}

func queryInt(c *gin.Context, key string, fallback int) int {
	var value int
	if _, err := fmt.Sscanf(c.Query(key), "%d", &value); err != nil || value <= 0 {
		return fallback
	}
	return value
}
