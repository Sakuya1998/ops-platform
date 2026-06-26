package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ops-platform/auth-svc/internal/service"
	"github.com/ops-platform/pkg/response"
)

type AuthHandler struct {
	authSvc *service.AuthService
	oidcSvc *service.OIDCService
}

func NewAuthHandler(authSvc *service.AuthService, oidcSvc *service.OIDCService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, oidcSvc: oidcSvc}
}

func (h *AuthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		OrgCode  string `json:"org_code"`
		Provider string `json:"provider"`
		MFACode  string `json:"mfa_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.authSvc.Login(c.Request.Context(), req.Username, req.Password, req.OrgCode, req.Provider, service.LoginContext{
		IP:         c.ClientIP(),
		UserAgent:  c.GetHeader("User-Agent"),
		DeviceName: c.GetHeader("X-Device-Name"),
		MFACode:    req.MFACode,
		RequestID:  requestID(c),
	})
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	response.Success(c, loginResponse(result))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		if token := bearerToken(c.GetHeader("Authorization")); token != "" && h.authSvc != nil {
			if claims, err := h.authSvc.ValidateToken(token); err == nil {
				userID = claims.UserID
			}
		}
	}
	if userID != "" && h.authSvc != nil {
		_ = h.authSvc.RevokeAllUserTokens(c.Request.Context(), userID)
	}
	response.OK(c)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.authSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, "Refresh token expired or invalid")
		return
	}
	response.Success(c, loginResponse(result))
}

func (h *AuthHandler) VerifyToken(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}
	_ = c.ShouldBindJSON(&req)
	token := strings.TrimSpace(req.Token)
	if token == "" {
		token = bearerToken(c.GetHeader("Authorization"))
	}
	if token == "" {
		response.BadRequest(c, "token is required")
		return
	}
	response.Success(c, h.authSvc.VerifyToken(token))
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		response.Unauthorized(c, "user not identified")
		return
	}
	user, err := h.authSvc.GetUser(userID)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.Success(c, user)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		response.Unauthorized(c, "user not identified")
		return
	}
	if err := h.authSvc.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) SetupMFA(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		response.Unauthorized(c, "user not identified")
		return
	}
	result, err := h.authSvc.SetupMFA(userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) ConfirmMFA(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		response.Unauthorized(c, "user not identified")
		return
	}
	result, err := h.authSvc.ConfirmMFA(userID, req.Code)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) DisableMFA(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
	}
	_ = c.ShouldBindJSON(&req)
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		response.Unauthorized(c, "user not identified")
		return
	}
	if err := h.authSvc.DisableMFA(userID, req.Code); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) RegenerateMFARecoveryCodes(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		response.Unauthorized(c, "user not identified")
		return
	}
	result, err := h.authSvc.RegenerateMFARecoveryCodes(userID, req.Code)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, result)
}

func (h *AuthHandler) ResetUserPassword(c *gin.Context) {
	var req struct {
		NewPassword        string `json:"new_password" binding:"required"`
		MustChangePassword bool   `json:"must_change_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := h.authSvc.ResetUserPassword(c.Param("id"), req.NewPassword, req.MustChangePassword)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, user)
}

func (h *AuthHandler) ListSessions(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		response.Unauthorized(c, "user not identified")
		return
	}
	sessions, err := h.authSvc.ListSessions(userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, sessions)
}

func (h *AuthHandler) RevokeSession(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		response.Unauthorized(c, "user not identified")
		return
	}
	if err := h.authSvc.RevokeSession(userID, c.Param("id"), "revoked_by_user"); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) RevokeOtherSessions(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	sessionID := c.GetHeader("X-Session-Id")
	if userID == "" {
		response.Unauthorized(c, "user not identified")
		return
	}
	if sessionID == "" {
		response.BadRequest(c, "session not identified")
		return
	}
	if err := h.authSvc.RevokeOtherSessions(userID, sessionID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) OIDCLogin(c *gin.Context) {
	if h.oidcSvc == nil || !h.oidcSvc.IsEnabled() {
		response.NotFound(c, "OIDC not configured")
		return
	}
	authURL, _, err := h.oidcSvc.LoginURL()
	if err != nil {
		response.InternalError(c, fmt.Sprintf("OIDC error: %v", err))
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) OIDCCallback(c *gin.Context) {
	if h.oidcSvc == nil || !h.oidcSvc.IsEnabled() {
		response.NotFound(c, "OIDC not configured")
		return
	}
	code, state := c.Query("code"), c.Query("state")
	if code == "" || state == "" {
		response.BadRequest(c, "Missing code or state")
		return
	}
	result, err := h.oidcSvc.HandleCallbackCode(c.Request.Context(), code, state)
	if err != nil {
		response.Unauthorized(c, fmt.Sprintf("OIDC failed: %v", err))
		return
	}
	c.Redirect(http.StatusFound, result.Redirect)
}

func (h *AuthHandler) OIDCStatus(c *gin.Context) {
	if h.oidcSvc != nil && h.oidcSvc.IsEnabled() {
		response.Success(c, gin.H{"enabled": true, "provider_name": h.oidcSvc.ProviderName(), "login_url": "/api/v1/auth/oidc/login"})
	} else {
		response.Success(c, gin.H{"enabled": false})
	}
}

func (h *AuthHandler) OIDCExchange(c *gin.Context) {
	if h.oidcSvc == nil {
		response.NotFound(c, "OIDC not configured")
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.oidcSvc.ExchangeLoginCode(c.Request.Context(), req.Code)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	response.Success(c, loginResponse(result))
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	orgID := c.GetHeader("X-Org-Id")
	if orgID == "" {
		orgID = c.DefaultQuery("org_id", "00000000-0000-0000-0000-000000000001")
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	users, total, err := h.authSvc.ListUsers(orgID, page, pageSize, c.Query("keyword"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.SuccessWithTotal(c, users, total)
}

func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req struct {
		OrgID       string `json:"org_id" binding:"required"`
		Username    string `json:"username" binding:"required"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := h.authSvc.CreateUser(req.OrgID, req.Username, req.Email, req.Phone, req.DisplayName, req.Password)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, user)
}

func (h *AuthHandler) GetUser(c *gin.Context) {
	user, err := h.authSvc.GetUser(c.Param("id"))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}
	response.Success(c, user)
}

func (h *AuthHandler) UpdateUser(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := h.authSvc.UpdateUser(c.Param("id"), req.Email, req.Phone, req.DisplayName)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, user)
}

func (h *AuthHandler) UpdateUserStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required,oneof=active disabled locked"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := h.authSvc.UpdateUserStatus(c.Param("id"), req.Status)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, user)
}

func (h *AuthHandler) DeleteUser(c *gin.Context) {
	if err := h.authSvc.DeleteUser(c.Param("id")); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func (h *AuthHandler) ListOrganizations(c *gin.Context) {
	orgs, err := h.authSvc.ListOrganizations()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, orgs)
}

func (h *AuthHandler) CreateOrganization(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	org, err := h.authSvc.CreateOrganization(req.Name, req.Code, req.Description)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, org)
}

func (h *AuthHandler) UpdateOrganization(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	org, err := h.authSvc.UpdateOrganization(c.Param("id"), req.Name, req.Description)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, org)
}

func (h *AuthHandler) GetSystemConfig(c *gin.Context) {
	orgID := c.GetHeader("X-Org-Id")
	if orgID == "" {
		orgID = c.DefaultQuery("org_id", "00000000-0000-0000-0000-000000000001")
	}
	cfg, err := h.authSvc.GetSystemConfig(orgID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.Success(c, cfg)
}

func (h *AuthHandler) UpdateSystemConfig(c *gin.Context) {
	orgID := c.GetHeader("X-Org-Id")
	if orgID == "" {
		orgID = c.DefaultQuery("org_id", "00000000-0000-0000-0000-000000000001")
	}
	var req service.SystemConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.authSvc.UpdateSystemConfig(orgID, req); err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c)
}

func bearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func requestID(c *gin.Context) string {
	if id := c.GetHeader("X-Request-Id"); id != "" {
		return id
	}
	return c.GetHeader("X-Request-ID")
}

func loginResponse(result *service.LoginResult) gin.H {
	return gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
		"token_type":    "Bearer",
		"session_id":    result.SessionID,
		"jti":           result.JTI,
		"user": gin.H{
			"user_id":              result.UserID,
			"org_id":               result.OrgID,
			"username":             result.Username,
			"display_name":         result.DisplayName,
			"email":                result.Email,
			"roles":                result.Roles,
			"must_change_password": result.MustChangePassword,
			"mfa_enabled":          result.MFAEnabled,
		},
	}
}
