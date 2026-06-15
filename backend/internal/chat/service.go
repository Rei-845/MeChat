package chat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"time"

	"mechat/internal/user"
	"mechat/internal/ws"
	"mechat/pkg/oss"
	"mechat/pkg/snowflake"

	redisPkg "mechat/pkg/redis"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNotMember    = errors.New("你不在该会话中")
	ErrNotOwner     = errors.New("只有群主才能执行此操作")
	ErrCannotRecall = errors.New("只能撤回自己发送的消息")
)

type Service struct {
	repo     *Repository
	users    *user.Cache // 带缓存的按 ID 读用户
	hub      *ws.Hub
	rdb      *redis.Client
	uploader oss.Uploader
	logger   *zap.Logger
}

// 创建会话服务
func NewService(repo *Repository, userRepo *user.Repository, hub *ws.Hub, rdb *redis.Client, uploader oss.Uploader, logger *zap.Logger) *Service {
	return &Service{repo: repo, users: user.NewCache(userRepo, rdb), hub: hub, rdb: rdb, uploader: uploader, logger: logger}
}

// 每会话上线补推上限
const offlineSyncPerConv = 30

// GetOrCreatePrivateConv 取或建单聊
func (s *Service) GetOrCreatePrivateConv(ctx context.Context, userA, userB uint64) (*Conversation, error) {
	conv, err := s.repo.GetPrivateConversation(ctx, userA, userB)
	if err == nil {
		return conv, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 创建新会话
	conv = &Conversation{Type: 1}
	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}
	roles := map[uint64]int8{userA: 0, userB: 0}
	if err := s.repo.AddMembers(ctx, conv.ID, []uint64{userA, userB}, roles); err != nil {
		return nil, err
	}
	return conv, nil
}

// CreateGroup 建群
func (s *Service) CreateGroup(ctx context.Context, ownerID uint64, req *CreateGroupReq) (*Conversation, error) {
	group := &Group{Name: req.Name, OwnerID: ownerID}
	if err := s.repo.CreateGroup(ctx, group); err != nil {
		return nil, err
	}

	conv := &Conversation{Type: 2, GroupID: group.ID}
	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}

	allMembers := append([]uint64{ownerID}, req.MemberIDs...)
	roles := map[uint64]int8{ownerID: 1} // 群主
	if err := s.repo.AddMembers(ctx, conv.ID, allMembers, roles); err != nil {
		return nil, err
	}
	return conv, nil
}

// CreateGroupInfo 建群并返回 DTO
func (s *Service) CreateGroupInfo(ctx context.Context, ownerID uint64, req *CreateGroupReq) (*ConversationInfo, error) {
	conv, err := s.CreateGroup(ctx, ownerID, req)
	if err != nil {
		return nil, err
	}
	return s.enrichConversation(ctx, ownerID, conv), nil
}

// HandleWSMessage 处理 WS 消息
func (s *Service) HandleWSMessage(ctx context.Context, senderID uint64, env *ws.Envelope) error {
	switch env.Type {
	case ws.TypeSendMsg:
		var data ws.SendMsgData
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return err
		}
		msg, err := s.SendMessage(ctx, senderID, data.ConversationID, int8(data.MsgType), data.Content)
		if err != nil {
			return err
		}
		// 回 ACK
		ack, _ := ws.BuildEnvelope(ws.TypeMsgAck, env.Seq, ws.MsgAckData{
			MsgID:          strconv.FormatUint(msg.ID, 10),
			ConversationID: msg.ConversationID,
			CreatedAt:      msg.CreatedAt.Format(time.RFC3339),
		})
		s.hub.Deliver(ctx, senderID, ack)

	case ws.TypeMsgRecall:
		var data struct {
			MsgID uint64 `json:"msg_id"`
		}
		if err := json.Unmarshal(env.Data, &data); err != nil {
			return err
		}
		return s.RecallMessage(ctx, senderID, data.MsgID)
	}
	return nil
}

// SendMessage 发消息
func (s *Service) SendMessage(ctx context.Context, senderID, convID uint64, msgType int8, content map[string]any) (*Message, error) {
	// 校验成员身份
	isMember, err := s.repo.IsMember(ctx, convID, senderID)
	if err != nil || !isMember {
		return nil, ErrNotMember
	}

	msg := &Message{
		ID:             snowflake.NextID(),
		ConversationID: convID,
		SenderID:       senderID,
		MsgType:        msgType,
		Content:        JSONMap(content),
	}
	now := time.Now()
	if err := s.repo.CreateMessageAndUpdateConv(ctx, msg, now); err != nil { // 先落库
		return nil, err
	}

	// 取成员推送
	memberIDs, err := s.repo.GetMemberIDs(ctx, convID)
	if err != nil {
		return msg, nil
	}

	payload := ws.NewMsgData{
		MsgID:          strconv.FormatUint(msg.ID, 10),
		ConversationID: convID,
		SenderID:       senderID,
		MsgType:        int(msgType),
		Content:        content,
		CreatedAt:      msg.CreatedAt.Format(time.RFC3339),
	}
	if sender, err := s.users.GetByID(ctx, senderID); err == nil {
		payload.SenderNickname = sender.Nickname
		payload.SenderAvatar = sender.AvatarURL
		payload.SenderVIP = sender.IsVIP()
	}
	pushData, _ := ws.BuildEnvelope(ws.TypeNewMsg, 0, payload)

	// 收件人 排除自己
	recipients := make([]uint64, 0, len(memberIDs))
	for _, uid := range memberIDs {
		if uid != senderID {
			recipients = append(recipients, uid)
		}
	}

	// 批量未读 +1 + 批量投递 离线靠上线补推
	redisPkg.MultiIncrConvUnread(ctx, s.rdb, convID, recipients)
	s.hub.DeliverMulti(ctx, recipients, pushData)

	return msg, nil
}

// DeliverOfflineMessages 上线补推未读
func (s *Service) DeliverOfflineMessages(ctx context.Context, userID uint64) error {
	convs, err := s.repo.GetUserConversations(ctx, userID)
	if err != nil {
		return err
	}
	if len(convs) == 0 {
		return nil
	}

	// 批量取已读位点
	convIDs := make([]uint64, 0, len(convs))
	for _, c := range convs {
		convIDs = append(convIDs, c.ID)
	}
	readSeqs, err := s.repo.GetReadSeqs(ctx, userID, convIDs)
	if err != nil {
		return err
	}

	for _, conv := range convs {
		readSeq := readSeqs[conv.ID]
		// 无新消息跳过
		if conv.LastMsgID == 0 || conv.LastMsgID <= readSeq {
			continue
		}
		msgs, err := s.repo.GetMessagesAfter(ctx, conv.ID, readSeq, offlineSyncPerConv)
		if err != nil || len(msgs) == 0 {
			continue
		}

		// 批量取发送者
		senderIDs := make([]uint64, 0, len(msgs))
		for _, m := range msgs {
			senderIDs = append(senderIDs, m.SenderID)
		}
		senders, _ := s.users.GetByIDs(ctx, senderIDs)

		for _, m := range msgs {
			// 跳过自己发的
			if m.SenderID == userID {
				continue
			}
			payload := ws.NewMsgData{
				MsgID:          strconv.FormatUint(m.ID, 10),
				ConversationID: m.ConversationID,
				SenderID:       m.SenderID,
				MsgType:        int(m.MsgType),
				Content:        m.Content,
				CreatedAt:      m.CreatedAt.Format(time.RFC3339),
				Sync:           true, // 补推历史 前端不再计未读
			}
			if u := senders[m.SenderID]; u != nil {
				payload.SenderNickname = u.Nickname
				payload.SenderAvatar = u.AvatarURL
				payload.SenderVIP = u.IsVIP()
			}
			if data, e := ws.BuildEnvelope(ws.TypeNewMsg, 0, payload); e == nil {
				s.hub.Deliver(ctx, userID, data)
			}
		}
	}
	return nil
}

// 撤回消息
func (s *Service) RecallMessage(ctx context.Context, senderID, msgID uint64) error {
	msg, err := s.repo.GetMessage(ctx, msgID)
	if err != nil {
		return err
	}
	if msg.SenderID != senderID {
		return ErrCannotRecall
	}
	// 只能撤回 2 分钟内
	if time.Since(msg.CreatedAt) > 2*time.Minute {
		return errors.New("只能撤回 2 分钟内的消息")
	}
	if err := s.repo.RecallMessage(ctx, msgID); err != nil {
		return err
	}

	// 通知会话成员
	memberIDs, _ := s.repo.GetMemberIDs(ctx, msg.ConversationID)
	notif, _ := ws.BuildEnvelope(ws.TypeMsgRecall, 0, map[string]any{
		"msg_id":          strconv.FormatUint(msgID, 10),
		"conversation_id": msg.ConversationID,
	})
	for _, uid := range memberIDs {
		s.hub.Deliver(ctx, uid, notif)
	}
	return nil
}

// GetConversations 会话列表 批量组装
func (s *Service) GetConversations(ctx context.Context, userID uint64) ([]*ConversationInfo, error) {
	convs, err := s.repo.GetUserConversations(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(convs) == 0 {
		return []*ConversationInfo{}, nil
	}

	// 收集各类 ID
	convIDs := make([]uint64, 0, len(convs))
	lastMsgIDs := make([]uint64, 0, len(convs))
	groupIDs := make([]uint64, 0)
	for _, c := range convs {
		convIDs = append(convIDs, c.ID)
		if c.LastMsgID > 0 {
			lastMsgIDs = append(lastMsgIDs, c.LastMsgID)
		}
		if c.Type == 2 && c.GroupID > 0 {
			groupIDs = append(groupIDs, c.GroupID)
		}
	}

	// 批量取成员分桶
	members, _ := s.repo.GetMembersByConvIDs(ctx, convIDs)
	membersByConv := make(map[uint64][]uint64, len(convs))
	for _, m := range members {
		membersByConv[m.ConversationID] = append(membersByConv[m.ConversationID], m.UserID)
	}

	// 批量取最后消息
	lastMsgs, _ := s.repo.GetMessagesByIDs(ctx, lastMsgIDs)
	msgByID := make(map[uint64]*Message, len(lastMsgs))
	for _, m := range lastMsgs {
		msgByID[m.ID] = m
	}

	// 批量取群信息
	groupByID := make(map[uint64]*Group, len(groupIDs))
	if len(groupIDs) > 0 {
		groups, _ := s.repo.GetGroupsByIDs(ctx, groupIDs)
		for _, g := range groups {
			groupByID[g.ID] = g
		}
	}

	// 取单聊对方 + 在线
	otherIDs := make([]uint64, 0)
	for _, c := range convs {
		if c.Type != 1 {
			continue
		}
		for _, uid := range membersByConv[c.ID] {
			if uid != userID {
				otherIDs = append(otherIDs, uid)
				break
			}
		}
	}
	userByID, _ := s.users.GetByIDs(ctx, otherIDs)
	onlineMap, _ := redisPkg.MultiIsOnline(ctx, s.rdb, otherIDs)

	// 批量取未读
	unreadMap, _ := redisPkg.MultiConvUnread(ctx, s.rdb, convIDs, userID)

	// 组装 全程查 map
	result := make([]*ConversationInfo, 0, len(convs))
	for _, conv := range convs {
		info := &ConversationInfo{
			ID:          conv.ID,
			Type:        conv.Type,
			GroupID:     conv.GroupID,
			UpdatedAt:   conv.UpdatedAt,
			UnreadCount: unreadMap[conv.ID],
		}
		if m := msgByID[conv.LastMsgID]; m != nil {
			info.LastMsg = toMessageInfo(m)
		}
		switch conv.Type {
		case 1:
			for _, uid := range membersByConv[conv.ID] {
				if uid == userID {
					continue
				}
				info.TargetUser = targetUserInfo(uid, userByID[uid], onlineMap[uid])
				break
			}
		case 2:
			info.GroupInfo = groupInfo(groupByID[conv.GroupID], len(membersByConv[conv.ID]))
		}
		result = append(result, info)
	}
	return result, nil
}

// 会话 DTO 叶子构造器 批量与单条共用
func targetUserInfo(uid uint64, u *user.User, online bool) *TargetUserInfo {
	if u == nil {
		return nil
	}
	return &TargetUserInfo{
		UserID:    uid,
		Nickname:  u.Nickname,
		AvatarURL: u.AvatarURL,
		IsOnline:  online,
		IsVIP:     u.IsVIP(),
	}
}

func groupInfo(g *Group, members int) *GroupInfo {
	if g == nil {
		return nil
	}
	return &GroupInfo{
		GroupID:   g.ID,
		Name:      g.Name,
		AvatarURL: g.AvatarURL,
		Members:   members,
	}
}

// GetOrCreatePrivateConvInfo 取或建单聊并返回 DTO
func (s *Service) GetOrCreatePrivateConvInfo(ctx context.Context, userID, targetUserID uint64) (*ConversationInfo, error) {
	conv, err := s.GetOrCreatePrivateConv(ctx, userID, targetUserID)
	if err != nil {
		return nil, err
	}
	return s.enrichConversation(ctx, userID, conv), nil
}

// enrichConversation 补全为 ConversationInfo
func (s *Service) enrichConversation(ctx context.Context, userID uint64, conv *Conversation) *ConversationInfo {
	info := &ConversationInfo{
		ID:        conv.ID,
		Type:      conv.Type,
		GroupID:   conv.GroupID,
		UpdatedAt: conv.UpdatedAt,
	}

	// 未读数
	unread, _ := s.rdb.Get(ctx, redisPkg.ConvUnreadKey(conv.ID, userID)).Int64()
	info.UnreadCount = unread

	// 最后一条消息
	if conv.LastMsgID > 0 {
		msg, err := s.repo.GetMessage(ctx, conv.LastMsgID)
		if err == nil {
			info.LastMsg = toMessageInfo(msg)
		}
	}

	if conv.Type == 1 {
		// 单聊找对方
		memberIDs, _ := s.repo.GetMemberIDs(ctx, conv.ID)
		for _, uid := range memberIDs {
			if uid != userID {
				u, _ := s.users.GetByID(ctx, uid)
				online, _ := redisPkg.IsOnline(ctx, s.rdb, uid)
				info.TargetUser = targetUserInfo(uid, u, online)
				break
			}
		}
	} else if conv.Type == 2 {
		g, _ := s.repo.GetGroup(ctx, conv.GroupID)
		memberIDs, _ := s.repo.GetMemberIDs(ctx, conv.ID)
		info.GroupInfo = groupInfo(g, len(memberIDs))
	}

	return info
}

// 消息列表
func (s *Service) GetMessages(ctx context.Context, userID, convID, beforeID uint64, limit int) ([]*MessageInfo, bool, error) {
	isMember, err := s.repo.IsMember(ctx, convID, userID)
	if err != nil || !isMember {
		return nil, false, ErrNotMember
	}

	if limit <= 0 || limit > 50 {
		limit = 20
	}

	msgs, err := s.repo.GetMessages(ctx, convID, beforeID, limit+1)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}

	// 批量取发送者
	senderIDs := make([]uint64, 0, len(msgs))
	for _, msg := range msgs {
		senderIDs = append(senderIDs, msg.SenderID)
	}
	senders, _ := s.users.GetByIDs(ctx, senderIDs)

	result := make([]*MessageInfo, 0, len(msgs))
	for _, msg := range msgs {
		info := toMessageInfo(msg)
		if u := senders[msg.SenderID]; u != nil {
			info.SenderNickname = u.Nickname
			info.SenderAvatar = u.AvatarURL
			info.SenderVIP = u.IsVIP()
		}
		result = append(result, info)
	}
	return result, hasMore, nil
}

// MarkRead 清未读并推进已读位点 msgID=0 回退为最后一条
func (s *Service) MarkRead(ctx context.Context, userID, convID, msgID uint64) error {
	s.rdb.Del(ctx, redisPkg.ConvUnreadKey(convID, userID))
	if msgID == 0 {
		if conv, err := s.repo.GetConversation(ctx, convID); err == nil {
			msgID = conv.LastMsgID
		}
	}
	if msgID == 0 {
		return nil // 无消息不推进
	}
	return s.repo.UpdateReadSeq(ctx, convID, userID, msgID)
}

// 查群
func (s *Service) GetGroup(ctx context.Context, groupID uint64) (*Group, error) {
	return s.repo.GetGroup(ctx, groupID)
}

// 加群成员
func (s *Service) AddGroupMembers(ctx context.Context, operatorID, convID uint64, memberIDs []uint64) error {
	member, err := s.repo.GetMembers(ctx, convID)
	if err != nil {
		return err
	}
	var operatorRole int8
	for _, m := range member {
		if m.UserID == operatorID {
			operatorRole = m.Role
			break
		}
	}
	if operatorRole < 1 {
		return ErrNotOwner
	}
	return s.repo.AddMembers(ctx, convID, memberIDs, nil)
}

// 移除群成员
func (s *Service) RemoveGroupMember(ctx context.Context, operatorID, convID, targetID uint64) error {
	members, err := s.repo.GetMembers(ctx, convID)
	if err != nil {
		return err
	}
	var operatorRole int8
	for _, m := range members {
		if m.UserID == operatorID {
			operatorRole = m.Role
			break
		}
	}
	if operatorRole < 1 && operatorID != targetID {
		return ErrNotOwner
	}
	return s.repo.RemoveMember(ctx, convID, targetID)
}

// 更新群头像
func (s *Service) UpdateGroupAvatar(ctx context.Context, operatorID, groupID uint64, reader io.Reader, size int64, filename string) (string, error) {
	g, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return "", errors.New("群不存在")
	}
	if g.OwnerID != operatorID {
		return "", ErrNotOwner
	}
	if s.uploader == nil {
		return "", errors.New("存储服务未配置")
	}
	ext := ""
	if i := len(filename) - 1; i >= 0 {
		for j := i; j >= 0; j-- {
			if filename[j] == '.' {
				ext = filename[j:]
				break
			}
		}
	}
	url, err := s.uploader.PutObject("group-avatars", ext, reader, size)
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdateGroupAvatar(ctx, groupID, url); err != nil {
		return "", err
	}
	return url, nil
}

type GroupMemberInfo struct {
	UserID    uint64 `json:"user_id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
	Role      int8   `json:"role"` // 0=普通 1=群主
	IsOnline  bool   `json:"is_online"`
}

// 群成员信息
func (s *Service) GetGroupMembersInfo(ctx context.Context, callerID, groupID uint64) ([]*GroupMemberInfo, error) {
	conv, err := s.repo.GetConvByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if ok, _ := s.repo.IsMember(ctx, conv.ID, callerID); !ok {
		return nil, ErrNotMember
	}
	members, err := s.repo.GetMembers(ctx, conv.ID)
	if err != nil {
		return nil, err
	}
	// 批量取成员 + 在线
	memberIDs := make([]uint64, 0, len(members))
	for _, m := range members {
		memberIDs = append(memberIDs, m.UserID)
	}
	users, _ := s.users.GetByIDs(ctx, memberIDs)
	onlineMap, _ := redisPkg.MultiIsOnline(ctx, s.rdb, memberIDs)

	result := make([]*GroupMemberInfo, 0, len(members))
	for _, m := range members {
		info := &GroupMemberInfo{UserID: m.UserID, Role: m.Role, IsOnline: onlineMap[m.UserID]}
		if u := users[m.UserID]; u != nil {
			info.Nickname = u.Nickname
			info.AvatarURL = u.AvatarURL
		}
		result = append(result, info)
	}
	return result, nil
}

// 按群 ID 查会话
func (s *Service) GetConvByGroupID(ctx context.Context, groupID uint64) (*Conversation, error) {
	return s.repo.GetConvByGroupID(ctx, groupID)
}

// 转 MessageInfo
func toMessageInfo(msg *Message) *MessageInfo {
	return &MessageInfo{
		MsgID:          strconv.FormatUint(msg.ID, 10),
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		MsgType:        msg.MsgType,
		Content:        msg.Content,
		IsRecalled:     msg.IsRecalled,
		CreatedAt:      msg.CreatedAt,
	}
}

// NotifySenderMessage 推消息给发送方本人 Agent 代发用
func (s *Service) NotifySenderMessage(ctx context.Context, senderID uint64, msg *Message) {
	payload := ws.NewMsgData{
		MsgID:          strconv.FormatUint(msg.ID, 10),
		ConversationID: msg.ConversationID,
		SenderID:       senderID,
		MsgType:        int(msg.MsgType),
		Content:        msg.Content,
		CreatedAt:      msg.CreatedAt.Format(time.RFC3339),
	}
	if u, err := s.users.GetByID(ctx, senderID); err == nil {
		payload.SenderNickname = u.Nickname
		payload.SenderAvatar = u.AvatarURL
		payload.SenderVIP = u.IsVIP()
	}
	if data, err := ws.BuildEnvelope(ws.TypeNewMsg, 0, payload); err == nil {
		s.hub.Deliver(ctx, senderID, data)
	}
}

// DeletePrivateConversation 删私聊及记录 返回会话 ID
func (s *Service) DeletePrivateConversation(ctx context.Context, userA, userB uint64) (uint64, error) {
	conv, err := s.repo.GetPrivateConversation(ctx, userA, userB)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	if err := s.repo.DeleteConversationCascade(ctx, conv.ID); err != nil {
		return conv.ID, err
	}
	s.rdb.Del(ctx, redisPkg.ConvUnreadKey(conv.ID, userA), redisPkg.ConvUnreadKey(conv.ID, userB))
	return conv.ID, nil
}

// DisbandGroup 群主解散群
func (s *Service) DisbandGroup(ctx context.Context, operatorID, convID uint64) error {
	members, err := s.repo.GetMembers(ctx, convID)
	if err != nil {
		return err
	}
	isOwner := false
	memberIDs := make([]uint64, 0, len(members))
	for _, m := range members {
		memberIDs = append(memberIDs, m.UserID)
		if m.UserID == operatorID && m.Role == 1 {
			isOwner = true
		}
	}
	if !isOwner {
		return ErrNotOwner
	}
	// 广播解散通知
	if payload, e := ws.BuildEnvelope(ws.TypeGroupDissolve, 0, ws.GroupDissolveData{ConversationID: convID}); e == nil {
		for _, uid := range memberIDs {
			s.hub.Deliver(ctx, uid, payload)
		}
	}
	return s.repo.DisbandConversation(ctx, convID)
}
