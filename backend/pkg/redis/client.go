package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// 创建 Redis 客户端
func NewClient(addr, password string, db, poolSize int) *redis.Client {
	if poolSize <= 0 {
		poolSize = 50
	}
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		PoolSize: poolSize,
	})
}

// Key helpers
func OnlineKey(userID uint64) string {
	return fmt.Sprintf("user:online:%d", userID)
}

func FriendListKey(userID uint64) string {
	return fmt.Sprintf("user:friends:%d", userID)
}

func EmailCodeLimitKey(email string) string {
	return fmt.Sprintf("email:code:limit:%s", email)
}

func ConvUnreadKey(convID, userID uint64) string {
	return fmt.Sprintf("conv:unread:%d:%d", convID, userID)
}

// UserInfoKey 用户信息缓存 key
func UserInfoKey(userID uint64) string {
	return fmt.Sprintf("user:info:%d", userID)
}

// MultiIncrConvUnread 批量给多个收件人未读 +1
func MultiIncrConvUnread(ctx context.Context, rdb *redis.Client, convID uint64, userIDs []uint64) {
	if len(userIDs) == 0 {
		return
	}
	rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
		for _, uid := range userIDs {
			key := ConvUnreadKey(convID, uid)
			p.Incr(ctx, key)
			p.Expire(ctx, key, 90*24*time.Hour)
		}
		return nil
	})
}

// MultiConvUnread 批量读多个会话的未读数
func MultiConvUnread(ctx context.Context, rdb *redis.Client, convIDs []uint64, userID uint64) (map[uint64]int64, error) {
	res := make(map[uint64]int64, len(convIDs))
	if len(convIDs) == 0 {
		return res, nil
	}
	keys := make([]string, len(convIDs))
	for i, cid := range convIDs {
		keys[i] = ConvUnreadKey(cid, userID)
	}
	vals, err := rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return res, err
	}
	for i, v := range vals {
		s, ok := v.(string)
		if !ok {
			continue // key 不存在跳过
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			res[convIDs[i]] = n
		}
	}
	return res, nil
}

// MultiIsOnline 批量查在线状态
func MultiIsOnline(ctx context.Context, rdb *redis.Client, userIDs []uint64) (map[uint64]bool, error) {
	res := make(map[uint64]bool, len(userIDs))
	if len(userIDs) == 0 {
		return res, nil
	}
	cmds := make([]*redis.IntCmd, len(userIDs))
	pipe := rdb.Pipeline()
	for i, uid := range userIDs {
		cmds[i] = pipe.Exists(ctx, OnlineKey(uid))
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return res, err
	}
	for i, uid := range userIDs {
		res[uid] = cmds[i].Val() > 0
	}
	return res, nil
}

// SetOnline 标记在线 (TTL 90s)
func SetOnline(ctx context.Context, rdb *redis.Client, userID uint64) error {
	return rdb.Set(ctx, OnlineKey(userID), 1, 90*time.Second).Err()
}

// SetOffline 标记下线
func SetOffline(ctx context.Context, rdb *redis.Client, userID uint64) error {
	return rdb.Del(ctx, OnlineKey(userID)).Err()
}

// IsOnline 是否在线
func IsOnline(ctx context.Context, rdb *redis.Client, userID uint64) (bool, error) {
	n, err := rdb.Exists(ctx, OnlineKey(userID)).Result()
	return n > 0, err
}
