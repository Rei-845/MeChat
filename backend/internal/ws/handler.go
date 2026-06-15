package ws

import (
	"context"
	"net/http"

	"mechat/pkg/jwt"
	"mechat/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Handler struct {
	hub           *Hub
	jwtMgr        *jwt.Manager
	rdb           *redis.Client
	msgSvc        MessageHandler
	logger        *zap.Logger
	allowedOrigin string
}

// 创建 WS 处理器
func NewHandler(hub *Hub, jwtMgr *jwt.Manager, rdb *redis.Client, msgSvc MessageHandler, logger *zap.Logger, allowedOrigin string) *Handler {
	return &Handler{hub: hub, jwtMgr: jwtMgr, rdb: rdb, msgSvc: msgSvc, logger: logger, allowedOrigin: allowedOrigin}
}

// websocket 升级器
func (h *Handler) upgrader() websocket.Upgrader {
	allowed := h.allowedOrigin
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if allowed == "*" {
				return true
			}
			return r.Header.Get("Origin") == allowed
		},
	}
}

// ServeWS GET /ws?token=xxx
func (h *Handler) ServeWS(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		response.Unauthorized(c, "缺少 token")
		return
	}

	blacklisted, err := h.jwtMgr.IsBlacklisted(c.Request.Context(), tokenStr)
	if err != nil || blacklisted {
		response.Unauthorized(c, "token 已失效")
		return
	}

	claims, err := h.jwtMgr.Parse(tokenStr)
	if err != nil {
		response.Unauthorized(c, "token 无效")
		return
	}

	up := h.upgrader()
	conn, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("ws upgrade", zap.Error(err))
		return
	}

	// 用新 context 防 HTTP 返回后被取消
	client := NewClient(claims.UserID, conn, h.hub, h.rdb, h.msgSvc, h.logger)
	go client.Start(context.Background())
}
