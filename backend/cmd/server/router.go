package main

import (
	"mechat/internal/ai"
	"mechat/internal/chat"
	"mechat/internal/feed"
	"mechat/internal/friend"
	"mechat/internal/level"
	"mechat/internal/user"
	"mechat/internal/vip"
	"mechat/internal/ws"
	"mechat/pkg/middleware"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	user   *user.Handler
	friend *friend.Handler
	chat   *chat.Handler
	feed   *feed.Handler
	ai     *ai.Handler
	vip    *vip.Handler
	level  *level.Handler
	ws     *ws.Handler
}

// 注册路由
func setupRouter(h *Handlers, authMW gin.HandlerFunc, allowedOrigin string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.CORS(allowedOrigin))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 本地上传静态服务
	r.Static("/uploads", "./uploads")

	// WebSocket
	r.GET("/ws", h.ws.ServeWS)

	// API v1
	v1 := r.Group("/api/v1")
	h.user.RegisterRoutes(v1, authMW)
	h.friend.RegisterRoutes(v1, authMW)
	h.chat.RegisterRoutes(v1, authMW)
	h.feed.RegisterRoutes(v1, authMW)
	h.ai.RegisterRoutes(v1, authMW)
	h.vip.RegisterRoutes(v1, authMW)
	h.level.RegisterRoutes(v1, authMW)

	return r
}
