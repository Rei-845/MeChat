package feed

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// StringSlice Post.Images 的 JSON 类型
type StringSlice []string

// 转 JSON 存库
func (s StringSlice) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	return string(b), err
}

// 从 JSON 读出
func (s *StringSlice) Scan(v any) error {
	var bs []byte
	switch val := v.(type) {
	case string:
		bs = []byte(val)
	case []byte:
		bs = val
	default:
		if v == nil {
			return nil
		}
		return fmt.Errorf("unsupported type: %T", v)
	}
	return json.Unmarshal(bs, s)
}

// Post 帖子
type Post struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"post_id"`
	UserID       uint64         `gorm:"not null;index:idx_user_created" json:"user_id"`
	Title        string         `gorm:"size:128" json:"title"`
	Content      string         `gorm:"type:text" json:"content"`
	Images       StringSlice    `gorm:"type:json" json:"images"`
	IP           string         `gorm:"size:64" json:"ip"` // 客户端 IP 属地展示
	LikeCount    int            `gorm:"default:0" json:"like_count"`
	CommentCount int            `gorm:"default:0" json:"comment_count"`
	CreatedAt    time.Time      `gorm:"index:idx_user_created" json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// PostLike 帖子点赞
type PostLike struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	PostID    uint64 `gorm:"uniqueIndex:uk_post_user;not null"`
	UserID    uint64 `gorm:"uniqueIndex:uk_post_user;not null;index"`
	CreatedAt time.Time
}

// PostComment 帖子评论
type PostComment struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	PostID    uint64         `gorm:"not null;index:idx_post_created" json:"post_id"`
	UserID    uint64         `gorm:"not null" json:"user_id"`
	ParentID  uint64         `gorm:"default:0;index" json:"parent_id"` // 0 表示一级评论
	Content   string         `gorm:"size:512;not null" json:"content"`
	LikeCount int            `gorm:"default:0" json:"like_count"`
	CreatedAt time.Time      `gorm:"index:idx_post_created" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// CommentLike 评论点赞
type CommentLike struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	CommentID uint64 `gorm:"uniqueIndex:uk_comment_user;not null"`
	UserID    uint64 `gorm:"uniqueIndex:uk_comment_user;not null;index"`
	CreatedAt time.Time
}

// DTO

type CreatePostReq struct {
	Title   string   `json:"title" binding:"required,min=1,max=60"`
	Content string   `json:"content" binding:"omitempty,max=2000"`
	Images  []string `json:"images"`
}

type CreateCommentReq struct {
	Content  string `json:"content" binding:"required,min=1,max=500"`
	ParentID uint64 `json:"parent_id"`
}

type PostInfo struct {
	PostID       uint64     `json:"post_id"`
	User         AuthorInfo `json:"user"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	Images       []string   `json:"images"`
	IP           string     `json:"ip"`
	LikeCount    int        `json:"like_count"`
	CommentCount int        `json:"comment_count"`
	IsLiked      bool       `json:"is_liked"`
	IsFriend     bool       `json:"is_friend"`
	CreatedAt    time.Time  `json:"created_at"`
}

type AuthorInfo struct {
	ID        uint64 `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	VIP       bool   `json:"vip"`
	Level     int    `json:"level"`
	Tier      string `json:"tier"`
}

type CommentInfo struct {
	ID             uint64        `json:"id"`
	PostID         uint64        `json:"post_id"`
	ParentID       uint64        `json:"parent_id"`
	User           AuthorInfo    `json:"user"`
	Content        string        `json:"content"`
	LikeCount      int           `json:"like_count"`
	IsLiked        bool          `json:"is_liked"`
	Replies        []CommentInfo `json:"replies,omitempty"`
	HasMoreReplies bool          `json:"has_more_replies"`
	CreatedAt      time.Time     `json:"created_at"`
}
