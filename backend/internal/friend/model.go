package friend

import "time"

// Friendship 好友关系
type Friendship struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	UserID    uint64 `gorm:"uniqueIndex:uk_user_friend;not null"`
	FriendID  uint64 `gorm:"uniqueIndex:uk_user_friend;not null;index"`
	Status    int8   `gorm:"default:1"` // 1=正常 2=拉黑
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FriendRequest 好友请求
type FriendRequest struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	FromUser  uint64 `gorm:"index;not null"`
	ToUser    uint64 `gorm:"index:idx_to_user;not null"`
	Message   string `gorm:"size:256"`
	Status    int8   `gorm:"index:idx_to_user;default:0"` // 0=待处理 1=同意 2=拒绝
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DTO

type SendRequestReq struct {
	ToUserID uint64 `json:"to_user_id" binding:"required"`
	Message  string `json:"message" binding:"max=100"`
}

type HandleRequestReq struct {
	Action string `json:"action" binding:"required,oneof=accept reject"`
}

type FriendInfo struct {
	UserID    uint64 `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	IsOnline  bool   `json:"is_online"`
	Level     int    `json:"level"`
	Tier      string `json:"tier"`
}

type RequestInfo struct {
	ID        uint64 `json:"id"`
	FromUser  uint64 `json:"from_user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Message   string `json:"message"`
	Status    int8   `json:"status"`
	CreatedAt string `json:"created_at"`
}
