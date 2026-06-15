package ws

import (
	"context"
	"encoding/json"
	"sync"

	redispkg "mechat/pkg/redis"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// fanoutChannel 跨节点投递的 Pub/Sub 频道
const fanoutChannel = "ws:fanout"

// fanoutMsg 跨节点投递信封 UserIDs 让群消息一次发布
type fanoutMsg struct {
	UserIDs []uint64 `json:"user_ids"`
	Payload []byte   `json:"payload"`
}

// Hub 本节点在线连接注册表
type Hub struct {
	clients    map[uint64]*Client // userID -> Client
	mu         sync.RWMutex
	register   chan *Client // 注册
	unregister chan *Client // 注销
	rdb        *redis.Client
	logger     *zap.Logger
}

// 创建 Hub
func NewHub(rdb *redis.Client, logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[uint64]*Client),
		register:   make(chan *Client, 128),
		unregister: make(chan *Client, 128),
		rdb:        rdb,
		logger:     logger,
	}
}

// Run 事件循环 需在 goroutine 启动
func (h *Hub) Run() {
	// 启动跨节点订阅
	if h.rdb != nil {
		go h.subscribe()
	}
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// 同用户重连关旧连接
			if old, ok := h.clients[client.userID]; ok {
				close(old.send)
			}
			h.clients[client.userID] = client
			h.mu.Unlock()
			h.logger.Info("ws client registered", zap.Uint64("user_id", client.userID))

		case client := <-h.unregister:
			h.mu.Lock()
			if cur, ok := h.clients[client.userID]; ok && cur == client {
				delete(h.clients, client.userID)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Info("ws client unregistered", zap.Uint64("user_id", client.userID))
		}
	}
}

// 订阅跨节点频道
func (h *Hub) subscribe() {
	sub := h.rdb.Subscribe(context.Background(), fanoutChannel)
	defer sub.Close()
	for msg := range sub.Channel() {
		var fm fanoutMsg
		if err := json.Unmarshal([]byte(msg.Payload), &fm); err != nil {
			continue
		}
		// 逐个本地投递
		for _, uid := range fm.UserIDs {
			h.deliverLocal(uid, fm.Payload)
		}
	}
}

// Deliver 向单个用户投递 在线返回 true
func (h *Hub) Deliver(ctx context.Context, userID uint64, data []byte) bool {
	// 本节点直达
	if h.deliverLocal(userID, data) {
		return true
	}
	// 无 Redis 按离线
	if h.rdb == nil {
		return false
	}
	// 跨节点发布
	online, err := redispkg.IsOnline(ctx, h.rdb, userID)
	if err != nil || !online {
		return false
	}
	body, _ := json.Marshal(fanoutMsg{UserIDs: []uint64{userID}, Payload: data})
	if err := h.rdb.Publish(ctx, fanoutChannel, body).Err(); err != nil {
		h.logger.Warn("ws fanout publish failed", zap.Uint64("user_id", userID), zap.Error(err))
		return false
	}
	return true
}

// DeliverMulti 群消息扇出 返回离线用户
func (h *Hub) DeliverMulti(ctx context.Context, userIDs []uint64, data []byte) []uint64 {
	if len(userIDs) == 0 {
		return nil
	}

	var offline []uint64 // 确认离线
	var remote []uint64  // 待 Redis 判断

	// 本节点直投
	h.mu.RLock()
	local := make(map[uint64]*Client, len(userIDs))
	for _, uid := range userIDs {
		if c, ok := h.clients[uid]; ok {
			local[uid] = c
		}
	}
	h.mu.RUnlock()

	for _, uid := range userIDs {
		c, ok := local[uid]
		if !ok {
			remote = append(remote, uid)
			continue
		}
		select {
		case c.send <- data:
		default:
			// 缓冲满 踢连接按离线
			h.unregister <- c
			offline = append(offline, uid)
		}
	}

	// 无 Redis remote 按离线
	if h.rdb == nil {
		return append(offline, remote...)
	}
	if len(remote) == 0 {
		return offline
	}

	// 批量查在线
	onlineMap, err := redispkg.MultiIsOnline(ctx, h.rdb, remote)
	if err != nil {
		// 查询失败全部按离线
		return append(offline, remote...)
	}
	var crossNode []uint64
	for _, uid := range remote {
		if onlineMap[uid] {
			crossNode = append(crossNode, uid)
		} else {
			offline = append(offline, uid)
		}
	}

	// 合并一次 PUBLISH
	if len(crossNode) > 0 {
		body, _ := json.Marshal(fanoutMsg{UserIDs: crossNode, Payload: data})
		if err := h.rdb.Publish(ctx, fanoutChannel, body).Err(); err != nil {
			h.logger.Warn("ws fanout publish failed", zap.Int("count", len(crossNode)), zap.Error(err))
			offline = append(offline, crossNode...)
		}
	}
	return offline
}

// 投递本节点连接
func (h *Hub) deliverLocal(userID uint64, data []byte) bool {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	select {
	case client.send <- data:
		return true
	default:
		// 缓冲满 视为掉线
		h.unregister <- client
		return false
	}
}

// OnlineCount 本节点在线连接数
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
