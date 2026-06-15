package ws

import "encoding/json"

// 消息类型常量
const (
	TypePing          = "ping"                // 心跳
	TypePong          = "pong"                // 心跳响应
	TypeSendMsg       = "send_msg"            // 客户端发送消息请求
	TypeMsgAck        = "msg_ack"             // 服务端发送消息确认
	TypeNewMsg        = "new_msg"             // 服务端推送新消息
	TypeMsgRecall     = "msg_recall"          // 消息撤回通知
	TypeFriendReq     = "friend_req"          // 好友请求通知
	TypeFriendAccept  = "friend_accept"       // 好友请求被接受通知
	TypeSystem        = "system"              // 系统消息
	TypeGroupDissolve = "group_dissolve"      // 群聊解散通知
	TypeConvRemove    = "conversation_remove" // 会话移除通知
	TypeAIResult      = "ai_result"           // 异步 AI 任务结果
)

// Envelope 统一 WS 消息信封
type Envelope struct {
	Type string          `json:"type"`
	Seq  int64           `json:"seq"`
	Data json.RawMessage `json:"data"` // 原始 JSON 按 Type 解析
}

// GroupDissolveData 群聊解散推送
type GroupDissolveData struct {
	ConversationID uint64 `json:"conversation_id"`
}

// ConvRemoveData 会话移除推送
type ConvRemoveData struct {
	ConversationID uint64 `json:"conversation_id"`
}

// SendMsgData 发送消息请求
type SendMsgData struct {
	ConversationID uint64         `json:"conversation_id"`
	MsgType        int            `json:"msg_type"`
	Content        map[string]any `json:"content"`
}

// MsgAckData 发送确认
type MsgAckData struct {
	MsgID          string `json:"msg_id"`
	ConversationID uint64 `json:"conversation_id"`
	CreatedAt      string `json:"created_at"`
}

// NewMsgData 新消息推送
type NewMsgData struct {
	MsgID          string         `json:"msg_id"`
	ConversationID uint64         `json:"conversation_id"`
	SenderID       uint64         `json:"sender_id"`
	SenderNickname string         `json:"sender_nickname,omitempty"`
	SenderAvatar   string         `json:"sender_avatar,omitempty"`
	SenderVIP      bool           `json:"sender_vip,omitempty"`
	MsgType        int            `json:"msg_type"`
	Content        map[string]any `json:"content"`
	CreatedAt      string         `json:"created_at"`
	// Sync 补推历史消息 前端不再本地计未读
	Sync bool `json:"sync,omitempty"`
}

// AIResultData 异步 AI 任务结果
type AIResultData struct {
	Kind           string `json:"kind"`
	ConversationID uint64 `json:"conversation_id,omitempty"`
	Result         string `json:"result,omitempty"`
	Error          string `json:"error,omitempty"`
}

// 构造 WS 信封
func BuildEnvelope(msgType string, seq int64, data any) ([]byte, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return json.Marshal(Envelope{Type: msgType, Seq: seq, Data: raw})
}
