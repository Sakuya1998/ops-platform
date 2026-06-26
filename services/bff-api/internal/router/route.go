package router

import (
	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/handler"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/middleware"
)

type Dependencies struct {
	Bootstrap     *handler.BootstrapHandler
	Auth          *handler.AuthHandler
	IAM           *handler.IAMHandler
	Audit         *handler.AuditHandler
	Notify        *handler.NotifyHandler
	Permission    middleware.PermissionChecker
	TokenVerifier middleware.TokenVerifier
}

func New(deps Dependencies) *gin.Engine {
	var r *gin.Engine
	if gin.Mode() == gin.TestMode {
		r = gin.New()
	} else {
		r = gin.Default()
	}
	Register(r, deps)
	return r
}

func Register(r gin.IRouter, deps Dependencies) {
	r.GET("/health", deps.Bootstrap.Health)
	r.GET("/api/v1/bootstrap", deps.Bootstrap.Bootstrap)
	userContext := middleware.RequireUserContext(deps.TokenVerifier)
	protected := middleware.RequirePermission(deps.Permission)

	auth := r.Group("/api/v1/auth")
	auth.POST("/login", deps.Auth.Login)
	auth.POST("/refresh", deps.Auth.Refresh)
	auth.POST("/token/verify", deps.Auth.VerifyToken)
	auth.GET("/oidc/login", deps.Auth.OIDCLogin)
	auth.GET("/oidc/callback", deps.Auth.OIDCCallback)
	auth.GET("/oidc/status", deps.Auth.OIDCStatus)
	auth.POST("/oidc/exchange", deps.Auth.OIDCExchange)

	protectedAuth := r.Group("/api/v1/auth")
	protectedAuth.Use(userContext, protected)
	protectedAuth.POST("/logout", deps.Auth.Logout)
	protectedAuth.GET("/me", deps.Auth.GetCurrentUser)
	protectedAuth.PUT("/me/password", deps.Auth.ChangePassword)
	protectedAuth.POST("/me/mfa/setup", deps.Auth.SetupMFA)
	protectedAuth.POST("/me/mfa/confirm", deps.Auth.ConfirmMFA)
	protectedAuth.POST("/me/mfa/recovery-codes", deps.Auth.RegenerateMFARecoveryCodes)
	protectedAuth.DELETE("/me/mfa", deps.Auth.DisableMFA)
	protectedAuth.GET("/sessions", deps.Auth.ListSessions)
	protectedAuth.DELETE("/sessions/:id", deps.Auth.RevokeSession)
	protectedAuth.DELETE("/sessions", deps.Auth.RevokeOtherSessions)

	users := r.Group("/api/v1/users")
	users.Use(userContext, protected)
	deps.IAM.RegisterUserRoutes(users)
	deps.Auth.RegisterUserRoutes(users)

	orgs := r.Group("/api/v1/organizations")
	orgs.Use(userContext, protected)
	deps.Auth.RegisterOrganizationRoutes(orgs)

	system := r.Group("/api/v1/system")
	system.Use(userContext, protected)
	deps.Auth.RegisterSystemRoutes(system)

	roles := r.Group("/api/v1/roles")
	roles.Use(userContext, protected)
	deps.IAM.RegisterRoleRoutes(roles)

	protectedV1 := r.Group("/api/v1")
	protectedV1.Use(userContext, protected)
	deps.IAM.RegisterPermissionRoutes(protectedV1)

	apiPermissions := r.Group("/api/v1/api-permissions")
	apiPermissions.Use(userContext, protected)
	deps.IAM.RegisterAPIPermissionRoutes(apiPermissions)

	audit := r.Group("/api/v1/audit-logs")
	audit.Use(userContext, protected)
	deps.Audit.Register(audit)

	notifications := r.Group("/api/v1/notifications")
	notifications.Use(userContext, protected)
	deps.Notify.RegisterChannels(notifications)

	notify := r.Group("/api/v1/notify")
	notify.Use(userContext, protected)
	deps.Notify.RegisterTemplates(notify)
}
