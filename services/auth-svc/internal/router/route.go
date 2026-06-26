package router

import (
	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/auth-svc/internal/handler"
)

func New(h *handler.AuthHandler) *gin.Engine {
	r := gin.Default()
	Register(r, h)
	return r
}

func Register(r gin.IRouter, h *handler.AuthHandler) {
	r.GET("/health", h.Health)

	auth := r.Group("/api/v1/auth")
	auth.POST("/login", h.Login)
	auth.POST("/logout", h.Logout)
	auth.POST("/refresh", h.RefreshToken)
	auth.POST("/token/verify", h.VerifyToken)
	auth.GET("/me", h.GetCurrentUser)
	auth.PUT("/me/password", h.ChangePassword)
	auth.POST("/me/mfa/setup", h.SetupMFA)
	auth.POST("/me/mfa/confirm", h.ConfirmMFA)
	auth.POST("/me/mfa/recovery-codes", h.RegenerateMFARecoveryCodes)
	auth.DELETE("/me/mfa", h.DisableMFA)
	auth.GET("/sessions", h.ListSessions)
	auth.DELETE("/sessions/:id", h.RevokeSession)
	auth.DELETE("/sessions", h.RevokeOtherSessions)
	auth.GET("/oidc/login", h.OIDCLogin)
	auth.GET("/oidc/callback", h.OIDCCallback)
	auth.GET("/oidc/status", h.OIDCStatus)
	auth.POST("/oidc/exchange", h.OIDCExchange)

	users := r.Group("/api/v1/users")
	users.GET("", h.ListUsers)
	users.POST("", h.CreateUser)
	users.GET("/:id", h.GetUser)
	users.PUT("/:id", h.UpdateUser)
	users.DELETE("/:id", h.DeleteUser)
	users.PUT("/:id/status", h.UpdateUserStatus)
	users.PUT("/:id/password/reset", h.ResetUserPassword)

	orgs := r.Group("/api/v1/organizations")
	orgs.GET("", h.ListOrganizations)
	orgs.POST("", h.CreateOrganization)
	orgs.PUT("/:id", h.UpdateOrganization)

	system := r.Group("/api/v1/system")
	system.GET("/config", h.GetSystemConfig)
	system.PUT("/config", h.UpdateSystemConfig)
}
