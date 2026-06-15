package vip

import "time"

// VIPOrder VIP 订单
type VIPOrder struct {
	ID        uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint64     `gorm:"not null;index" json:"user_id"`
	Plan      string     `gorm:"size:32;not null" json:"plan"`
	Amount    float64    `gorm:"type:decimal(10,2);not null" json:"amount"` // 用 decimal 避免精度问题
	Status    int8       `gorm:"default:0" json:"status"`                   // 0=待支付 1=已支付 2=已取消
	ExpiredAt *time.Time `json:"expired_at"`                                // 永久为 NULL
	PaidAt    *time.Time `json:"paid_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// DTO

type CreateOrderReq struct {
	Plan string `json:"plan" binding:"required,oneof=monthly yearly lifetime"`
}

type PlanInfo struct {
	Plan        string  `json:"plan"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Duration    string  `json:"duration"`
	AIQuotaDay  int     `json:"ai_quota_day"`
	Description string  `json:"description"`
	Badge       string  `json:"badge,omitempty"`
}

type VIPStatus struct {
	VIPLevel   int8       `json:"vip_level"`
	ExpiredAt  *time.Time `json:"expired_at"`
	IsActive   bool       `json:"is_active"`
	IsLifetime bool       `json:"is_lifetime"`
}

// VIP 套餐
var Plans = map[string]*PlanInfo{
	"monthly": {
		Plan:        "monthly",
		Name:        "月度 VIP",
		Price:       1,
		Duration:    "30天",
		AIQuotaDay:  50,
		Description: "每日 50 次 AI 调用，解锁全部功能",
	},
	"yearly": {
		Plan:        "yearly",
		Name:        "年度 VIP",
		Price:       10,
		Duration:    "365天",
		AIQuotaDay:  50,
		Description: "全年权益，折合每月不到 1 元",
		Badge:       "省17%",
	},
	"lifetime": {
		Plan:        "lifetime",
		Name:        "永久 VIP",
		Price:       50,
		Duration:    "永久",
		AIQuotaDay:  50,
		Description: "一次买断，用到跑路为止",
		Badge:       "最划算",
	},
}
