package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/ops-platform/bff-api/internal/client"
	"github.com/ops-platform/pkg/response"
)

type PermissionChecker interface {
	CheckPermission(ctx context.Context, req client.CheckPermissionRequest) (client.CheckPermissionResult, error)
}

func RequirePermission(checker PermissionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-Id")
		orgID := c.GetHeader("X-Org-Id")
		if userID == "" || orgID == "" {
			response.Unauthorized(c, "missing user context")
			c.Abort()
			return
		}

		result, err := checker.CheckPermission(c.Request.Context(), client.CheckPermissionRequest{
			UserID: userID,
			OrgID:  orgID,
			Method: c.Request.Method,
			Path:   c.Request.URL.Path,
		})
		if err != nil {
			response.InternalError(c, "permission check failed")
			c.Abort()
			return
		}
		if !result.Allowed {
			message := result.Reason
			if message == "" {
				message = "permission denied"
			}
			response.Forbidden(c, message)
			c.Abort()
			return
		}
		c.Set("user_roles", result.Roles)
		c.Next()
	}
}
