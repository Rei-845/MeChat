package user

import (
	"context"
	"encoding/json"
	"time"

	redispkg "mechat/pkg/redis"

	"github.com/redis/go-redis/v9"
)

// 用户信息缓存 TTL
const userInfoTTL = 5 * time.Minute

// Cache 按 ID 读用户的展示缓存 只读不可用于鉴权
type Cache struct {
	repo *Repository
	rdb  *redis.Client
}

// 创建用户缓存
func NewCache(repo *Repository, rdb *redis.Client) *Cache {
	return &Cache{repo: repo, rdb: rdb}
}

// GetByID 按 ID 读 缓存优先
func (c *Cache) GetByID(ctx context.Context, id uint64) (*User, error) {
	// 读缓存
	if data, err := c.rdb.Get(ctx, redispkg.UserInfoKey(id)).Bytes(); err == nil {
		var u User
		if json.Unmarshal(data, &u) == nil {
			return &u, nil
		}
	}
	// 回源
	u, err := c.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// 写回缓存
	if data, e := json.Marshal(u); e == nil {
		c.rdb.Set(ctx, redispkg.UserInfoKey(id), data, userInfoTTL)
	}
	return u, nil
}

// GetByIDs 批量读 返回 id->User
func (c *Cache) GetByIDs(ctx context.Context, ids []uint64) (map[uint64]*User, error) {
	result := make(map[uint64]*User, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	// 去重
	uniq := make([]uint64, 0, len(ids))
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	// MGET 批量读缓存
	keys := make([]string, len(uniq))
	for i, id := range uniq {
		keys[i] = redispkg.UserInfoKey(id)
	}
	vals, _ := c.rdb.MGet(ctx, keys...).Result() // 出错全部当未命中

	var miss []uint64
	for i, id := range uniq {
		if i < len(vals) {
			if s, ok := vals[i].(string); ok {
				var u User
				if json.Unmarshal([]byte(s), &u) == nil {
					result[id] = &u
					continue
				}
			}
		}
		miss = append(miss, id)
	}

	// 未命中回源
	if len(miss) > 0 {
		users, err := c.repo.GetByIDs(ctx, miss)
		if err != nil {
			return result, err
		}
		// pipeline 写回
		c.rdb.Pipelined(ctx, func(p redis.Pipeliner) error {
			for _, u := range users {
				result[u.ID] = u
				if data, e := json.Marshal(u); e == nil {
					p.Set(ctx, redispkg.UserInfoKey(u.ID), data, userInfoTTL)
				}
			}
			return nil
		})
	}
	return result, nil
}

// Invalidate 失效用户缓存
func Invalidate(ctx context.Context, rdb *redis.Client, id uint64) {
	if rdb == nil {
		return
	}
	rdb.Del(ctx, redispkg.UserInfoKey(id))
}
