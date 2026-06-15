package ws

import (
	"context"
	"encoding/json"
	"time"

	redispkg "mechat/pkg/redis"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 90 * time.Second
	pingPeriod     = 30 * time.Second
	maxMessageSize = 8192
)

// Client 单个 WebSocket 连接
type Client struct {
	userID uint64
	conn   *websocket.Conn
	send   chan []byte // 发送队列
	hub    *Hub
	rdb    *redis.Client
	logger *zap.Logger
	msgSvc MessageHandler
}

// MessageHandler 由 chat.Service 实现
type MessageHandler interface {
	HandleWSMessage(ctx context.Context, userID uint64, env *Envelope) error
	DeliverOfflineMessages(ctx context.Context, userID uint64) error
}

// 创建客户端
func NewClient(userID uint64, conn *websocket.Conn, hub *Hub, rdb *redis.Client, msgSvc MessageHandler, logger *zap.Logger) *Client {
	return &Client{
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, 256),
		hub:    hub,
		rdb:    rdb,
		logger: logger,
		msgSvc: msgSvc,
	}
}

// 启动读写循环
func (c *Client) Start(ctx context.Context) {
	// 标记上线
	redispkg.SetOnline(ctx, c.rdb, c.userID)

	// 注册到 Hub
	c.hub.register <- c

	// 推送离线消息
	go func() {
		if err := c.msgSvc.DeliverOfflineMessages(ctx, c.userID); err != nil {
			c.logger.Error("deliver offline messages", zap.Error(err))
		}
	}()

	// 写循环 goroutine 读循环阻塞
	go c.writePump()
	c.readPump(ctx)

	// 下线清理
	c.hub.unregister <- c
	redispkg.SetOffline(context.Background(), c.rdb, c.userID)
	c.conn.Close()
}

// 读循环
func (c *Client) readPump(ctx context.Context) {
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		redispkg.SetOnline(ctx, c.rdb, c.userID) // 心跳续期
		return nil
	})

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			continue
		}

		// 协议层 ping
		if env.Type == TypePing {
			redispkg.SetOnline(ctx, c.rdb, c.userID)
			pong, _ := BuildEnvelope(TypePong, env.Seq, nil)
			c.send <- pong
			continue
		}

		// 交业务层处理
		if err := c.msgSvc.HandleWSMessage(ctx, c.userID, &env); err != nil {
			c.logger.Error("handle ws message", zap.Error(err))
		}
	}
}

// 写循环
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// 通道关闭 Hub 要求断开
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
