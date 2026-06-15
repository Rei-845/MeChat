package chat

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Conversation 会话
type Conversation struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Type      int8       `gorm:"not null" json:"type"`            // 1=单聊 2=群聊
	GroupID   uint64     `gorm:"default:0;index" json:"group_id"` // 单聊为 0
	LastMsgID uint64     `gorm:"default:0" json:"last_msg_id"`
	LastMsgAt *time.Time `json:"last_msg_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ConversationMember 会话成员
type ConversationMember struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`
	ConversationID uint64    `gorm:"uniqueIndex:uk_conv_user;not null;index"`
	UserID         uint64    `gorm:"uniqueIndex:uk_conv_user;not null;index"`
	Role           int8      `gorm:"default:0"` // 0=普通 1=群主
	ReadSeq        uint64    `gorm:"default:0"` // 已读位点
	JoinedAt       time.Time `gorm:"not null;autoCreateTime"`
}

// Group 群聊
type Group struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name       string     `gorm:"size:64;not null" json:"name"`
	AvatarURL  string     `gorm:"size:512" json:"avatar_url"`
	OwnerID    uint64     `gorm:"not null;index" json:"owner_id"`
	MaxMembers int        `gorm:"default:500" json:"max_members"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `gorm:"index" json:"-"`
}

// JSONMap Message.Content 的 JSON 类型
type JSONMap map[string]any

// 转 JSON 存库
func (j JSONMap) Value() (driver.Value, error) {
	b, err := json.Marshal(j)
	return string(b), err
}

// 从 JSON 读出
func (j *JSONMap) Scan(v any) error {
	var bs []byte
	switch val := v.(type) {
	case string:
		bs = []byte(val)
	case []byte:
		bs = val
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
	return json.Unmarshal(bs, j)
}

// Message 消息
type Message struct {
	ID             uint64     `gorm:"primaryKey" json:"msg_id"` // 雪花 ID
	ConversationID uint64     `gorm:"not null;index:idx_conv_id_created" json:"conversation_id"`
	SenderID       uint64     `gorm:"not null;index" json:"sender_id"`
	MsgType        int8       `gorm:"not null" json:"msg_type"` // 1=文本 2=图片 3=文件 4=AI生成
	Content        JSONMap    `gorm:"type:json;not null" json:"content"`
	IsRecalled     bool       `gorm:"default:false" json:"is_recalled"`
	RecalledAt     *time.Time `json:"recalled_at,omitempty"`
	CreatedAt      time.Time  `gorm:"index:idx_conv_id_created" json:"created_at"`
}

// DTO

type CreatePrivateConvReq struct {
	TargetUserID uint64 `json:"target_user_id" binding:"required"`
}

type CreateGroupReq struct {
	Name      string   `json:"name" binding:"required,min=2,max=30"`
	MemberIDs []uint64 `json:"member_ids" binding:"required,min=1"`
}

type GetMessagesReq struct {
	BeforeID uint64 `form:"before_id"`
	Limit    int    `form:"limit,default=20"`
}

type ConversationInfo struct {
	ID          uint64       `json:"id"`
	Type        int8         `json:"type"`
	GroupID     uint64       `json:"group_id,omitempty"`
	LastMsg     *MessageInfo `json:"last_msg,omitempty"`
	UnreadCount int64        `json:"unread_count"`
	UpdatedAt   time.Time    `json:"updated_at"`
	// 单聊填对方
	TargetUser *TargetUserInfo `json:"target_user,omitempty"`
	// 群聊填群信息
	GroupInfo *GroupInfo `json:"group_info,omitempty"`
}

type TargetUserInfo struct {
	UserID    uint64 `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	IsOnline  bool   `json:"is_online"`
	IsVIP     bool   `json:"is_vip"`
}

type GroupInfo struct {
	GroupID   uint64 `json:"group_id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Members   int    `json:"members"`
}

type MessageInfo struct {
	MsgID          string         `json:"msg_id"` // 雪花 ID 字符串防精度丢失
	ConversationID uint64         `json:"conversation_id"`
	SenderID       uint64         `json:"sender_id"`
	SenderNickname string         `json:"sender_nickname,omitempty"`
	SenderAvatar   string         `json:"sender_avatar,omitempty"`
	SenderVIP      bool           `json:"sender_vip,omitempty"`
	MsgType        int8           `json:"msg_type"`
	Content        map[string]any `json:"content"`
	IsRecalled     bool           `json:"is_recalled"`
	CreatedAt      time.Time      `json:"created_at"`
}
