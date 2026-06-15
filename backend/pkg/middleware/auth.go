package middleware

import (
	"strings"

	"mechat/pkg/jwt"
	"mechat/pkg/response"

	"github.com/gin-gonic/gin"
)

const ContextUserID = "userID"

// JWT 鉴权中间件
func Auth(jwtMgr *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Unauthorized(c, "缺少认证令牌")
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		blacklisted, err := jwtMgr.IsBlacklisted(c.Request.Context(), tokenStr)
		if err == nil && blacklisted { // Redis 不可用时放行
			response.Unauthorized(c, "令牌已失效")
			return
		}

		claims, err := jwtMgr.Parse(tokenStr)
		if err != nil {
			response.Unauthorized(c, "令牌无效或已过期")
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set("tokenStr", tokenStr)
		c.Set("tokenExpireAt", claims.ExpiresAt.Time)
		c.Next()
	}
}

// 取当前用户 ID
func CurrentUserID(c *gin.Context) uint64 {
	id, _ := c.Get(ContextUserID)
	return id.(uint64)
}
