package user

import (
	"context"
	"errors"
	"math/rand"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// Repository 用户仓库
type Repository struct {
	db *gorm.DB
}

// 创建用户仓库
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// 创建用户
func (r *Repository) Create(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// 按 ID 查用户
func (r *Repository) GetByID(ctx context.Context, id uint64) (*User, error) {
	var u User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// 按邮箱查用户
func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// 更新用户
func (r *Repository) Update(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Save(u).Error
}

// 更新头像
func (r *Repository) UpdateAvatar(ctx context.Context, id uint64, url string) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Update("avatar_url", url).Error
}

// 加经验 gorm.Expr 原子操作
func (r *Repository) AddExperience(ctx context.Context, userID uint64, xp int) error {
	return r.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", userID).
		UpdateColumn("experience", gorm.Expr("experience + ?", xp)).Error
}

// 昵称是否被占用
func (r *Repository) NicknameExists(ctx context.Context, nickname string, excludeID uint64) (bool, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&User{}).Where("BINARY nickname = ?", nickname)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// 搜索用户 支持 MeChatID (即 id) 或昵称
func (r *Repository) Search(ctx context.Context, keyword string, limit int) ([]*User, error) {
	var users []*User
	q := r.db.WithContext(ctx).
		Where("nickname LIKE ? OR email = ?", "%"+keyword+"%", keyword)
	if id, err := strconv.ParseUint(keyword, 10, 64); err == nil {
		q = q.Or("id = ?", id)
	}
	err := q.Limit(limit).Find(&users).Error
	return users, err
}

// Recommend 随机推荐用户 用随机 ID 跳取避免全表扫描
func (r *Repository) Recommend(ctx context.Context, excludeIDs []uint64, limit int) ([]*User, error) {
	var maxID uint64
	if err := r.db.WithContext(ctx).Model(&User{}).Select("COALESCE(MAX(id), 0)").Scan(&maxID).Error; err != nil || maxID == 0 {
		return []*User{}, nil
	}
	pivot := uint64(rand.Int63n(int64(maxID) + 1))

	fetch := func(startID uint64, asc bool, n int) []*User {
		var us []*User
		order, cond := "id ASC", "id >= ?"
		if !asc {
			order, cond = "id DESC", "id < ?"
		}
		q := r.db.WithContext(ctx).Where(cond, startID).Order(order).Limit(n)
		if len(excludeIDs) > 0 {
			q = q.Where("id NOT IN ?", excludeIDs)
		}
		q.Find(&us)
		return us
	}

	users := fetch(pivot, true, limit)
	if len(users) < limit {
		users = append(users, fetch(pivot, false, limit-len(users))...)
	}
	return users, nil
}

// 批量查用户
func (r *Repository) GetByIDs(ctx context.Context, ids []uint64) ([]*User, error) {
	var users []*User
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	return users, err
}

// EmailCode methods

// 创建验证码 旧码失效
func (r *Repository) CreateCode(ctx context.Context, code *EmailVerifyCode) error {
	r.db.WithContext(ctx).Model(&EmailVerifyCode{}).
		Where("email = ? AND purpose = ? AND used = false", code.Email, code.Purpose).
		Update("used", true)
	return r.db.WithContext(ctx).Create(code).Error
}

// 取有效验证码
func (r *Repository) GetValidCode(ctx context.Context, email, purpose string) (*EmailVerifyCode, error) {
	var code EmailVerifyCode
	err := r.db.WithContext(ctx).
		Where("email = ? AND purpose = ? AND used = false AND expired_at > ?", email, purpose, time.Now()).
		Order("created_at DESC").
		First(&code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &code, err
}

// 标记验证码已用
func (r *Repository) MarkCodeUsed(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&EmailVerifyCode{}).Where("id = ?", id).Update("used", true).Error
}
