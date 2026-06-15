package friend

import (
	"context"
	"errors"
	"strconv"
	"time"

	"mechat/internal/user"
	"mechat/internal/ws"
	redisPkg "mechat/pkg/redis"

	"github.com/redis/go-redis/v9"
)

var (
	ErrAlreadyFriends  = errors.New("你们已经是好友了")
	ErrRequestPending  = errors.New("已发送过好友申请，等待对方处理")
	ErrNotYourRequest  = errors.New("无权操作此申请")
	ErrRequestNotFound = errors.New("申请不存在")
)

// ChatConvRemover 解耦引用 chat.Service 防循环依赖
type ChatConvRemover interface {
	DeletePrivateConversation(ctx context.Context, userA, userB uint64) (uint64, error)
}

type Service struct {
	repo     *Repository
	userRepo *user.Repository
	hub      *ws.Hub
	rdb      *redis.Client
	chatSvc  ChatConvRemover
}

// 创建好友服务
func NewService(repo *Repository, userRepo *user.Repository, hub *ws.Hub, rdb *redis.Client) *Service {
	return &Service{repo: repo, userRepo: userRepo, hub: hub, rdb: rdb}
}

// SetChatSvc 注入 chat 服务
func (s *Service) SetChatSvc(c ChatConvRemover) { s.chatSvc = c }

// 发送好友请求
func (s *Service) SendRequest(ctx context.Context, fromUser uint64, req *SendRequestReq) error {
	// 已是好友
	ok, err := s.repo.AreFriends(ctx, fromUser, req.ToUserID)
	if err != nil {
		return err
	}
	if ok {
		return ErrAlreadyFriends
	}

	// 已有待处理申请
	pending, err := s.repo.HasPendingRequest(ctx, fromUser, req.ToUserID)
	if err != nil {
		return err
	}
	if pending {
		return ErrRequestPending
	}

	record := &FriendRequest{
		FromUser: fromUser,
		ToUser:   req.ToUserID,
		Message:  req.Message,
	}
	if err := s.repo.CreateRequest(ctx, record); err != nil {
		return err
	}

	// 推送通知
	fromUserInfo, _ := s.userRepo.GetByID(ctx, fromUser)
	notif, _ := ws.BuildEnvelope(ws.TypeFriendReq, 0, map[string]any{
		"request_id": record.ID,
		"from_user":  fromUser,
		"nickname":   fromUserInfo.Nickname,
		"avatar_url": fromUserInfo.AvatarURL,
		"message":    req.Message,
	})
	s.hub.Deliver(ctx, req.ToUserID, notif)
	return nil
}

// 处理好友请求
func (s *Service) HandleRequest(ctx context.Context, toUser, requestID uint64, action string) error {
	record, err := s.repo.GetRequest(ctx, requestID)
	if err != nil {
		return ErrRequestNotFound
	}
	if record.ToUser != toUser {
		return ErrNotYourRequest
	}
	if record.Status != 0 {
		return errors.New("该申请已处理")
	}

	if action == "accept" {
		if err := s.repo.UpdateRequestStatus(ctx, requestID, 1); err != nil {
			return err
		}
		if err := s.repo.CreateFriendship(ctx, toUser, record.FromUser); err != nil {
			return err
		}
		// 清除好友列表缓存
		s.rdb.Del(ctx, redisPkg.FriendListKey(toUser), redisPkg.FriendListKey(record.FromUser))

		// 通知申请方
		notif, _ := ws.BuildEnvelope(ws.TypeFriendAccept, 0, map[string]any{
			"user_id": toUser,
		})
		s.hub.Deliver(ctx, record.FromUser, notif)
	} else {
		if err := s.repo.UpdateRequestStatus(ctx, requestID, 2); err != nil {
			return err
		}
	}
	return nil
}

// 好友列表
func (s *Service) GetFriendList(ctx context.Context, userID uint64) ([]*FriendInfo, error) {
	ids, err := s.repo.GetFriendIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []*FriendInfo{}, nil
	}

	users, err := s.userRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]*FriendInfo, 0, len(users))
	for _, u := range users {
		online, _ := redisPkg.IsOnline(ctx, s.rdb, u.ID)
		result = append(result, &FriendInfo{
			UserID:    u.ID,
			Nickname:  u.Nickname,
			AvatarURL: u.AvatarURL,
			IsOnline:  online,
			Level:     u.Level(),
			Tier:      u.Tier(),
		})
	}
	return result, nil
}

// 删好友
func (s *Service) DeleteFriend(ctx context.Context, userID, friendID uint64) error {
	s.rdb.Del(ctx, redisPkg.FriendListKey(userID), redisPkg.FriendListKey(friendID))

	// 删私聊会话并通知两端
	if s.chatSvc != nil {
		if convID, err := s.chatSvc.DeletePrivateConversation(ctx, userID, friendID); err == nil && convID > 0 {
			if notif, e := ws.BuildEnvelope(ws.TypeConvRemove, 0, ws.ConvRemoveData{ConversationID: convID}); e == nil {
				s.hub.Deliver(ctx, userID, notif)
				s.hub.Deliver(ctx, friendID, notif)
			}
		}
	}
	return s.repo.DeleteFriendship(ctx, userID, friendID)
}

// 待处理请求
func (s *Service) GetPendingRequests(ctx context.Context, userID uint64) ([]*RequestInfo, error) {
	reqs, err := s.repo.GetPendingRequests(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 批量预取消除 N+1
	fromIDs := make([]uint64, len(reqs))
	for i, req := range reqs {
		fromIDs[i] = req.FromUser
	}
	users, _ := s.userRepo.GetByIDs(ctx, fromIDs)
	userByID := make(map[uint64]*user.User, len(users))
	for _, u := range users {
		userByID[u.ID] = u
	}

	result := make([]*RequestInfo, 0, len(reqs))
	for _, req := range reqs {
		info := &RequestInfo{
			ID:        req.ID,
			FromUser:  req.FromUser,
			Message:   req.Message,
			Status:    req.Status,
			CreatedAt: req.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if u := userByID[req.FromUser]; u != nil {
			info.Nickname = u.Nickname
			info.AvatarURL = u.AvatarURL
		}
		result = append(result, info)
	}
	return result, nil
}

// 是否好友
func (s *Service) AreFriends(ctx context.Context, userA, userB uint64) (bool, error) {
	return s.repo.AreFriends(ctx, userA, userB)
}

// 好友列表缓存兜底 TTL
const friendListTTL = 24 * time.Hour

// 好友 ID 列表 (带缓存)
func (s *Service) GetFriendIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	// 查缓存
	key := redisPkg.FriendListKey(userID)
	vals, err := s.rdb.LRange(ctx, key, 0, -1).Result()
	if err == nil && len(vals) > 0 {
		ids := make([]uint64, 0, len(vals))
		for _, v := range vals {
			id, perr := strconv.ParseUint(v, 10, 64)
			if perr == nil && id != 0 { // 跳过哨兵 0
				ids = append(ids, id)
			}
		}
		return ids, nil
	}

	// 未命中回源
	ids, err := s.repo.GetFriendIDs(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 写回缓存 空列表也占位防穿透
	s.cacheFriendIDs(ctx, key, ids)
	return ids, nil
}

// 重建好友列表缓存
func (s *Service) cacheFriendIDs(ctx context.Context, key string, ids []uint64) {
	s.rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
		p.Del(ctx, key)
		members := make([]any, 0, len(ids)+1)
		members = append(members, "0") // 哨兵 保证空列表也命中
		for _, id := range ids {
			members = append(members, strconv.FormatUint(id, 10))
		}
		p.RPush(ctx, key, members...)
		p.Expire(ctx, key, friendListTTL)
		return nil
	})
}
