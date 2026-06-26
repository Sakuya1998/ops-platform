package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ops-platform/iam-svc/internal/handler"
)

func New(h *handler.IAMHandler) *gin.Engine {
	r := gin.Default()
	Register(r, h)
	return r
}

func Register(r gin.IRouter, h *handler.IAMHandler) {
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	internal := r.Group("/internal/v1")
	internal.POST("/check-permission", h.CheckPermission)
	internal.POST("/permissions/check", h.CheckPermissionAPI)
	internal.POST("/permissions/batch-check", h.BatchCheckPermissionAPI)
	internal.GET("/users/:id/roles", h.GetUserRoles)

	v1 := r.Group("/api/v1")
	v1.PUT("/users/:id/roles", h.AssignUserRoles)
	v1.GET("/users/:id/roles", h.GetUserRoles)
	v1.GET("/users/:id/permissions", h.GetUserPermissions)

	v1.GET("/roles", h.ListRoles)
	v1.POST("/roles", h.CreateRole)
	v1.GET("/roles/:id", h.GetRole)
	v1.PUT("/roles/:id", h.UpdateRole)
	v1.DELETE("/roles/:id", h.DeleteRole)
	v1.PUT("/roles/:id/permissions", h.AssignPermissions)

	v1.GET("/permissions", h.GetPermissionTree)

	v1.GET("/api-permissions", h.ListAPIPermissions)
	v1.POST("/api-permissions", h.CreateAPIPermission)
	v1.PUT("/api-permissions/:id", h.UpdateAPIPermission)
	v1.DELETE("/api-permissions/:id", h.DeleteAPIPermission)
}
