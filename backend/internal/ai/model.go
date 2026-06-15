package ai

import "time"

// AIMessage AI 对话消息 落库
type AIMessage struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"index;not null" json:"-"`
	Role      string    `gorm:"size:16;not null" json:"role"` // user / assistant
	Content   string    `gorm:"type:longtext" json:"content"`
	Note      string    `gorm:"type:text" json:"-"`                                              // 工具执行注记 仅喂模型 不展示给用户
	Pending   string    `gorm:"type:text;column:pending_action" json:"pending_action,omitempty"` // 待确认写操作 JSON 空表示无
	Status    string    `gorm:"size:16;default:done" json:"status"`                              // pending / done / error
	CreatedAt time.Time `json:"created_at"`
}

// DTO

type SummarizeReq struct {
	ConversationID uint64 `json:"conversation_id" binding:"required"`
	MessageCount   int    `json:"message_count" binding:"required,min=1,max=200"`
}

type DraftMessageReq struct {
	Draft   string `json:"draft" binding:"required,max=500"`
	Context string `json:"context" binding:"max=200"`
}

type DraftPostReq struct {
	Keywords string `json:"keywords" binding:"required,max=200"`
}

type ChatReq struct {
	Text string `json:"text" binding:"required,max=8000"`
}

// ConfirmActionReq 确认执行写操作
type ConfirmActionReq struct {
	Tool string         `json:"tool" binding:"required"`
	Args map[string]any `json:"args"`
}

type ConfirmActionResp struct {
	Result string `json:"result"`
}

type QuotaInfo struct {
	VIPUser bool `json:"vip_user"` // 仅区分 VIP
}
