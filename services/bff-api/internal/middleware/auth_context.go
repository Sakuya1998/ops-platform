package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/Sakuya1998/ops-platform/services/bff-api/internal/client"
	"github.com/Sakuya1998/ops-platform/pkg/response"
)

type TokenVerifier interface {
	VerifyToken(ctx context.Context, authorization string) (client.TokenContext, error)
}

func RequireUserContext(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-User-Id") != "" && c.GetHeader("X-Org-Id") != "" {
			c.Next()
			return
		}
		authorization := c.GetHeader("Authorization")
		if authorization == "" {
			response.Unauthorized(c, "missing authorization")
			c.Abort()
			return
		}
		tokenCtx, err := verifier.VerifyToken(c.Request.Context(), authorization)
		if err != nil {
			response.Unauthorized(c, "invalid token")
			c.Abort()
			return
		}
		c.Request.Header.Set("X-User-Id", tokenCtx.UserID)
		c.Request.Header.Set("X-Org-Id", tokenCtx.OrgID)
		if tokenCtx.SessionID != "" {
			c.Request.Header.Set("X-Session-Id", tokenCtx.SessionID)
		}
		c.Next()
	}
}
