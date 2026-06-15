package friend

import (
	"context"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

// 创建好友仓库
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// 是否好友
func (r *Repository) AreFriends(ctx context.Context, userA, userB uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Friendship{}).
		Where("user_id = ? AND friend_id = ? AND status = 1", userA, userB).
		Count(&count).Error
	return count > 0, err
}

// 好友 ID 列表
func (r *Repository) GetFriendIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	var ids []uint64
	err := r.db.WithContext(ctx).Model(&Friendship{}).
		Where("user_id = ? AND status = 1", userID).
		Pluck("friend_id", &ids).Error
	return ids, err
}

// 写入好友关系 事务
func (r *Repository) CreateFriendship(ctx context.Context, userA, userB uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 双向写入
		if err := tx.Create(&Friendship{UserID: userA, FriendID: userB}).Error; err != nil {
			return err
		}
		return tx.Create(&Friendship{UserID: userB, FriendID: userA}).Error
	})
}

// 删除好友关系 事务
func (r *Repository) DeleteFriendship(ctx context.Context, userA, userB uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx.Where("user_id = ? AND friend_id = ?", userA, userB).Delete(&Friendship{})
		tx.Where("user_id = ? AND friend_id = ?", userB, userA).Delete(&Friendship{})
		return nil
	})
}

// 创建好友请求
func (r *Repository) CreateRequest(ctx context.Context, req *FriendRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

// 查请求
func (r *Repository) GetRequest(ctx context.Context, id uint64) (*FriendRequest, error) {
	var req FriendRequest
	if err := r.db.WithContext(ctx).First(&req, id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

// 待处理请求
func (r *Repository) GetPendingRequests(ctx context.Context, toUser uint64) ([]*FriendRequest, error) {
	var reqs []*FriendRequest
	err := r.db.WithContext(ctx).
		Where("to_user = ? AND status = 0", toUser).
		Order("created_at DESC").
		Find(&reqs).Error
	return reqs, err
}

// 更新请求状态
func (r *Repository) UpdateRequestStatus(ctx context.Context, id uint64, status int8) error {
	return r.db.WithContext(ctx).Model(&FriendRequest{}).Where("id = ?", id).Update("status", status).Error
}

// 是否有待处理请求
func (r *Repository) HasPendingRequest(ctx context.Context, fromUser, toUser uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&FriendRequest{}).
		Where("from_user = ? AND to_user = ? AND status = 0", fromUser, toUser).
		Count(&count).Error
	return count > 0, err
}
