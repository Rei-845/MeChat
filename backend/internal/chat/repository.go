package chat

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

// 创建会话仓库
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Conversation

// 创建会话
func (r *Repository) CreateConversation(ctx context.Context, conv *Conversation) error {
	return r.db.WithContext(ctx).Create(conv).Error
}

// 查会话
func (r *Repository) GetConversation(ctx context.Context, id uint64) (*Conversation, error) {
	var conv Conversation
	if err := r.db.WithContext(ctx).First(&conv, id).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// 查私聊会话
func (r *Repository) GetPrivateConversation(ctx context.Context, userA, userB uint64) (*Conversation, error) {
	var conv Conversation
	err := r.db.WithContext(ctx).Raw(`
		SELECT c.* FROM conversations c
		JOIN conversation_members m1 ON m1.conversation_id = c.id AND m1.user_id = ?
		JOIN conversation_members m2 ON m2.conversation_id = c.id AND m2.user_id = ?
		WHERE c.type = 1
		LIMIT 1
	`, userA, userB).Scan(&conv).Error
	if err != nil {
		return nil, err
	}
	if conv.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &conv, nil
}

// 用户会话列表
func (r *Repository) GetUserConversations(ctx context.Context, userID uint64) ([]*Conversation, error) {
	var convs []*Conversation
	err := r.db.WithContext(ctx).Raw(`
		SELECT c.* FROM conversations c
		JOIN conversation_members m ON m.conversation_id = c.id AND m.user_id = ?
		ORDER BY (c.last_msg_at IS NULL), c.last_msg_at DESC, c.id DESC
		LIMIT 50
	`, userID).Scan(&convs).Error
	return convs, err
}

// 加成员
func (r *Repository) AddMembers(ctx context.Context, convID uint64, userIDs []uint64, roles map[uint64]int8) error {
	members := make([]*ConversationMember, 0, len(userIDs))
	for _, uid := range userIDs {
		role := int8(0)
		if r, ok := roles[uid]; ok {
			role = r
		}
		members = append(members, &ConversationMember{
			ConversationID: convID,
			UserID:         uid,
			Role:           role,
		})
	}
	return r.db.WithContext(ctx).Create(&members).Error
}

// 会话成员
func (r *Repository) GetMembers(ctx context.Context, convID uint64) ([]*ConversationMember, error) {
	var members []*ConversationMember
	err := r.db.WithContext(ctx).Where("conversation_id = ?", convID).Find(&members).Error
	return members, err
}

// 成员 ID
func (r *Repository) GetMemberIDs(ctx context.Context, convID uint64) ([]uint64, error) {
	var ids []uint64
	err := r.db.WithContext(ctx).Model(&ConversationMember{}).
		Where("conversation_id = ?", convID).
		Pluck("user_id", &ids).Error
	return ids, err
}

// 批量取多会话成员 消除 N+1
func (r *Repository) GetMembersByConvIDs(ctx context.Context, convIDs []uint64) ([]*ConversationMember, error) {
	var members []*ConversationMember
	if len(convIDs) == 0 {
		return members, nil
	}
	err := r.db.WithContext(ctx).
		Where("conversation_id IN ?", convIDs).
		Find(&members).Error
	return members, err
}

// 是否成员
func (r *Repository) IsMember(ctx context.Context, convID, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Count(&count).Error
	return count > 0, err
}

// 更新 last_msg
func (r *Repository) UpdateLastMsg(ctx context.Context, convID, msgID uint64, at time.Time) error {
	return r.db.WithContext(ctx).Model(&Conversation{}).Where("id = ?", convID).
		Updates(map[string]any{"last_msg_id": msgID, "last_msg_at": at}).Error
}

// 推进已读位点 read_seq < ? 保证单调
func (r *Repository) UpdateReadSeq(ctx context.Context, convID, userID, msgID uint64) error {
	return r.db.WithContext(ctx).Model(&ConversationMember{}).
		Where("conversation_id = ? AND user_id = ? AND read_seq < ?", convID, userID, msgID).
		Update("read_seq", msgID).Error
}

// 批量取已读位点 返回 convID->readSeq
func (r *Repository) GetReadSeqs(ctx context.Context, userID uint64, convIDs []uint64) (map[uint64]uint64, error) {
	res := make(map[uint64]uint64, len(convIDs))
	if len(convIDs) == 0 {
		return res, nil
	}
	var members []*ConversationMember
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND conversation_id IN ?", userID, convIDs).
		Find(&members).Error
	if err != nil {
		return res, err
	}
	for _, m := range members {
		res[m.ConversationID] = m.ReadSeq
	}
	return res, nil
}

// 取已读位点
func (r *Repository) GetReadSeq(ctx context.Context, convID, userID uint64) (uint64, error) {
	var member ConversationMember
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return member.ReadSeq, err
}

// Message

// 创建消息
func (r *Repository) CreateMessage(ctx context.Context, msg *Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// 事务内写消息并更新 last_msg
func (r *Repository) CreateMessageAndUpdateConv(ctx context.Context, msg *Message, at time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		return tx.Model(&Conversation{}).Where("id = ?", msg.ConversationID).
			Updates(map[string]any{"last_msg_id": msg.ID, "last_msg_at": at}).Error
	})
}

// 会话消息分页
func (r *Repository) GetMessages(ctx context.Context, convID, beforeID uint64, limit int) ([]*Message, error) {
	var msgs []*Message
	query := r.db.WithContext(ctx).Where("conversation_id = ? AND is_recalled = false", convID)
	if beforeID > 0 {
		query = query.Where("id < ?", beforeID)
	}
	err := query.Order("id DESC").Limit(limit).Find(&msgs).Error
	return msgs, err
}

// 查消息
func (r *Repository) GetMessage(ctx context.Context, msgID uint64) (*Message, error) {
	var msg Message
	if err := r.db.WithContext(ctx).First(&msg, msgID).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

// 批量按 ID 取消息
func (r *Repository) GetMessagesByIDs(ctx context.Context, ids []uint64) ([]*Message, error) {
	var msgs []*Message
	if len(ids) == 0 {
		return msgs, nil
	}
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&msgs).Error
	return msgs, err
}

// 取 afterID 之后的消息 用于上线补推
func (r *Repository) GetMessagesAfter(ctx context.Context, convID, afterID uint64, limit int) ([]*Message, error) {
	var msgs []*Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND id > ? AND is_recalled = false", convID, afterID).
		Order("id ASC").
		Limit(limit).
		Find(&msgs).Error
	return msgs, err
}

// 撤回消息
func (r *Repository) RecallMessage(ctx context.Context, msgID uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&Message{}).Where("id = ?", msgID).
		Updates(map[string]any{"is_recalled": true, "recalled_at": &now}).Error
}

// Group

// 创建群
func (r *Repository) CreateGroup(ctx context.Context, group *Group) error {
	return r.db.WithContext(ctx).Create(group).Error
}

// 查群
func (r *Repository) GetGroup(ctx context.Context, id uint64) (*Group, error) {
	var group Group
	if err := r.db.WithContext(ctx).First(&group, id).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// 批量按 ID 取群
func (r *Repository) GetGroupsByIDs(ctx context.Context, ids []uint64) ([]*Group, error) {
	var groups []*Group
	if len(ids) == 0 {
		return groups, nil
	}
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&groups).Error
	return groups, err
}

// 更新群
func (r *Repository) UpdateGroup(ctx context.Context, group *Group) error {
	return r.db.WithContext(ctx).Save(group).Error
}

// 移除成员
func (r *Repository) RemoveMember(ctx context.Context, convID, userID uint64) error {
	return r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Delete(&ConversationMember{}).Error
}

// 更新群头像
func (r *Repository) UpdateGroupAvatar(ctx context.Context, groupID uint64, url string) error {
	return r.db.WithContext(ctx).Model(&Group{}).Where("id = ?", groupID).Update("avatar_url", url).Error
}

// 成员角色
func (r *Repository) GetMemberRole(ctx context.Context, convID, userID uint64) (int8, error) {
	var m ConversationMember
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		First(&m).Error
	return m.Role, err
}

// 按群 ID 查会话
func (r *Repository) GetConvByGroupID(ctx context.Context, groupID uint64) (*Conversation, error) {
	var conv Conversation
	err := r.db.WithContext(ctx).Where("group_id = ? AND type = 2", groupID).First(&conv).Error
	return &conv, err
}

// 解散会话 删成员与会话
func (r *Repository) DisbandConversation(ctx context.Context, convID uint64) error {
	r.db.WithContext(ctx).Where("conversation_id = ?", convID).Delete(&ConversationMember{})
	return r.db.WithContext(ctx).Delete(&Conversation{}, convID).Error
}

// 级联删会话 消息 成员
func (r *Repository) DeleteConversationCascade(ctx context.Context, convID uint64) error {
	db := r.db.WithContext(ctx)
	db.Where("conversation_id = ?", convID).Delete(&Message{})
	db.Where("conversation_id = ?", convID).Delete(&ConversationMember{})
	return db.Delete(&Conversation{}, convID).Error
}
