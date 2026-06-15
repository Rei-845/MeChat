package user

import (
	"time"

	"gorm.io/gorm"
)

// User 用户
type User struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Email        string         `gorm:"uniqueIndex;size:128;not null" json:"email"`
	Password     string         `gorm:"size:256;not null" json:"-"`
	Nickname     string         `gorm:"size:64;not null" json:"nickname"`
	AvatarURL    string         `gorm:"size:512" json:"avatar_url"`
	Bio          string         `gorm:"size:256" json:"bio"`
	VIPLevel     int8           `gorm:"default:0" json:"vip_level"`
	VIPExpiredAt *time.Time     `json:"vip_expired_at"` // nil 表示永久
	Experience   int            `gorm:"default:0" json:"experience"`
	Status       int8           `gorm:"default:1" json:"status"` // 账号状态
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// EmailVerifyCode 邮箱验证码
type EmailVerifyCode struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement"`
	Email     string    `gorm:"index:idx_email_purpose;size:128;not null"`
	Code      string    `gorm:"size:8;not null"`
	Purpose   string    `gorm:"index:idx_email_purpose;size:32;not null"` // register/login/reset
	ExpiredAt time.Time `gorm:"not null"`
	Used      bool      `gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}

// DTO

// RegisterReq 注册请求
type RegisterReq struct {
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required,len=6"`
	Nickname string `json:"nickname" binding:"required,min=2,max=20"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginReq 登录请求
type LoginReq struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"omitempty,min=6"`
	Code     string `json:"code"     binding:"omitempty,len=6"`
}

// SendCodeReq 发送验证码
type SendCodeReq struct {
	Email   string `json:"email" binding:"required,email"`
	Purpose string `json:"purpose" binding:"required,oneof=register login reset"`
}

// UpdateProfileReq 更新资料
type UpdateProfileReq struct {
	Nickname string `json:"nickname" binding:"omitempty,min=2,max=20"`
	Bio      string `json:"bio" binding:"omitempty,max=256"`
}

// UserInfo 自己可见信息
type UserInfo struct {
	ID           uint64     `json:"id"`
	Email        string     `json:"email"`
	Nickname     string     `json:"nickname"`
	AvatarURL    string     `json:"avatar_url"`
	Bio          string     `json:"bio"`
	VIPLevel     int8       `json:"vip_level"`
	VIPExpiredAt *time.Time `json:"vip_expired_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// 转 UserInfo
func (u *User) ToInfo() *UserInfo {
	return &UserInfo{
		ID:           u.ID,
		Email:        u.Email,
		Nickname:     u.Nickname,
		AvatarURL:    u.AvatarURL,
		Bio:          u.Bio,
		VIPLevel:     u.VIPLevel,
		VIPExpiredAt: u.VIPExpiredAt,
		CreatedAt:    u.CreatedAt,
	}
}

// 是否有效 VIP
func (u *User) IsVIP() bool {
	return u.VIPLevel > 0 && (u.VIPExpiredAt == nil || u.VIPExpiredAt.After(time.Now()))
}

// PublicInfo 公开信息
type PublicInfo struct {
	ID        uint64 `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
	VIP       bool   `json:"vip"`
	Level     int    `json:"level"`
	Tier      string `json:"tier"` // gray / blue / yellow / orange
}

func (u *User) Level() int   { return levelFromXP(u.Experience) }
func (u *User) Tier() string { return tierOf(u.Level()) }

// level 包逻辑的本地副本 避免循环导入
func levelFromXP(xp int) int {
	thresholds := [10]int{0, 10, 20, 30, 50, 70, 90, 120, 150, 190}
	lv := 1
	for i := 9; i >= 0; i-- {
		if xp >= thresholds[i] {
			lv = i + 1
			break
		}
	}
	return lv
}
func tierOf(lv int) string {
	switch {
	case lv <= 3:
		return "gray"
	case lv <= 6:
		return "blue"
	case lv <= 8:
		return "yellow"
	default:
		return "orange"
	}
}

// 转 PublicInfo
func (u *User) ToPublicInfo() *PublicInfo {
	return &PublicInfo{
		ID:        u.ID,
		Nickname:  u.Nickname,
		AvatarURL: u.AvatarURL,
		Bio:       u.Bio,
		VIP:       u.IsVIP(),
		Level:     u.Level(),
		Tier:      u.Tier(),
	}
}
