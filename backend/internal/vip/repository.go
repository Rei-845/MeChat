package vip

import (
	"context"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

// 创建 VIP 仓库
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// 创建订单
func (r *Repository) CreateOrder(ctx context.Context, order *VIPOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

// 查订单
func (r *Repository) GetOrder(ctx context.Context, id uint64) (*VIPOrder, error) {
	var order VIPOrder
	if err := r.db.WithContext(ctx).First(&order, id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// 更新订单
func (r *Repository) UpdateOrder(ctx context.Context, order *VIPOrder) error {
	return r.db.WithContext(ctx).Save(order).Error
}
