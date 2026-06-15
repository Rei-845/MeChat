package vip

import (
	"context"
	"errors"
	"time"

	"mechat/internal/user"
)

type Service struct {
	repo     *Repository
	userRepo *user.Repository
}

// 创建 VIP 服务
func NewService(repo *Repository, userRepo *user.Repository) *Service {
	return &Service{repo: repo, userRepo: userRepo}
}

// 套餐列表
func (s *Service) GetPlans() []*PlanInfo {
	return []*PlanInfo{Plans["monthly"], Plans["yearly"], Plans["lifetime"]}
}

// 创建订单
func (s *Service) CreateOrder(ctx context.Context, userID uint64, req *CreateOrderReq) (*VIPOrder, error) {
	plan, ok := Plans[req.Plan]
	if !ok {
		return nil, errors.New("无效的套餐")
	}

	order := &VIPOrder{
		UserID: userID,
		Plan:   req.Plan,
		Amount: plan.Price,
		Status: 0,
	}
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}
	return order, nil
}

// Pay 模拟支付立即生效
func (s *Service) Pay(ctx context.Context, userID, orderID uint64) (*VIPStatus, error) {
	order, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, errors.New("订单不存在")
	}
	if order.UserID != userID {
		return nil, errors.New("无权操作此订单")
	}
	if order.Status != 0 {
		return nil, errors.New("订单状态异常")
	}

	// 算到期时间
	now := time.Now()
	var expiredAt time.Time
	switch order.Plan {
	case "monthly":
		expiredAt = now.AddDate(0, 1, 0)
	case "yearly":
		expiredAt = now.AddDate(1, 0, 0)
	case "lifetime":
		expiredAt = now.AddDate(100, 0, 0) // 100 年视为永久
	}

	// 更新订单
	order.Status = 1
	order.PaidAt = &now
	order.ExpiredAt = &expiredAt
	if err := s.repo.UpdateOrder(ctx, order); err != nil {
		return nil, err
	}

	// 更新用户 VIP
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	// 已有 VIP 未过期则续期
	if u.VIPLevel > 0 && u.VIPExpiredAt != nil && u.VIPExpiredAt.After(now) && order.Plan != "lifetime" {
		switch order.Plan {
		case "monthly":
			expiredAt = u.VIPExpiredAt.AddDate(0, 1, 0)
		case "yearly":
			expiredAt = u.VIPExpiredAt.AddDate(1, 0, 0)
		}
	}
	u.VIPLevel = 1
	u.VIPExpiredAt = &expiredAt
	if err := s.userRepo.Update(ctx, u); err != nil {
		return nil, err
	}

	return &VIPStatus{
		VIPLevel:  1,
		ExpiredAt: &expiredAt,
		IsActive:  true,
	}, nil
}

// 查 VIP 状态
func (s *Service) GetStatus(ctx context.Context, userID uint64) (*VIPStatus, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	isActive := u.VIPLevel > 0 && (u.VIPExpiredAt == nil || u.VIPExpiredAt.After(time.Now()))
	isLifetime := isActive && u.VIPExpiredAt != nil && u.VIPExpiredAt.Year() > time.Now().Year()+50
	return &VIPStatus{
		VIPLevel:   u.VIPLevel,
		ExpiredAt:  u.VIPExpiredAt,
		IsActive:   isActive,
		IsLifetime: isLifetime,
	}, nil
}
