package ai

import (
	"context"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

// 创建 AI 消息仓库
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// 追加一条消息
func (r *Repository) Add(ctx context.Context, m *AIMessage) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// 最近 limit 条 正序返回 用作模型上下文与前端展示
func (r *Repository) Recent(ctx context.Context, userID uint64, limit int) ([]*AIMessage, error) {
	var msgs []*AIMessage
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").Limit(limit).
		Find(&msgs).Error
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, err
}

// 完成生成 写入内容 状态 注记 与待确认写操作
func (r *Repository) Finish(ctx context.Context, id uint64, content, status, note, pending string) error {
	return r.db.WithContext(ctx).Model(&AIMessage{}).
		Where("id = ?", id).
		Updates(map[string]any{"content": content, "status": status, "note": note, "pending_action": pending}).Error
}

// 最近一条还挂着待确认写操作的 assistant id
func (r *Repository) latestPending(ctx context.Context, userID uint64) (uint64, error) {
	var m AIMessage
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND role = ? AND pending_action <> ''", userID, "assistant").
		Order("id DESC").First(&m).Error
	return m.ID, err
}

// ResolvePending 写操作已执行 清掉待确认并写最终注记
func (r *Repository) ResolvePending(ctx context.Context, userID uint64, note string) error {
	id, err := r.latestPending(ctx, userID)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&AIMessage{}).
		Where("id = ?", id).
		Updates(map[string]any{"note": note, "pending_action": ""}).Error
}

// ClearPending 用户取消 只清掉待确认
func (r *Repository) ClearPending(ctx context.Context, userID uint64) error {
	id, err := r.latestPending(ctx, userID)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&AIMessage{}).
		Where("id = ?", id).Update("pending_action", "").Error
}

// 清空某用户全部 AI 消息
func (r *Repository) Clear(ctx context.Context, userID uint64) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&AIMessage{}).Error
}
