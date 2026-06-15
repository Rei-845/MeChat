package level

import (
	"context"
	"fmt"
	"time"

	"mechat/internal/user"

	"github.com/redis/go-redis/v9"
)

const (
	xpPost          = 6
	xpComment       = 3
	xpCheckin       = 6
	commentDailyCap = 12 // 每日评论 XP 上限
)

// Service 经验与等级
type Service struct {
	userRepo *user.Repository
	rdb      *redis.Client
}

// 创建等级服务
func NewService(userRepo *user.Repository, rdb *redis.Client) *Service {
	return &Service{userRepo: userRepo, rdb: rdb}
}

// ── Redis 键 ──

func checkinKey(userID uint64, date string) string {
	return fmt.Sprintf("level:checkin:%d:%s", userID, date)
}

func today() string { return time.Now().Format("20060102") }

// ── 经验发放 ──

// 发帖加经验
func (s *Service) AddPostXP(ctx context.Context, userID uint64, isVIP bool) {
	xp := xpPost
	if isVIP {
		xp *= 2
	}
	s.userRepo.AddExperience(ctx, userID, xp)
}

// 评论加经验 每日封顶
func (s *Service) AddCommentXP(ctx context.Context, userID uint64, isVIP bool) int {
	base := xpComment
	cap := commentDailyCap
	if isVIP {
		base *= 2
		cap *= 2
	}
	key := fmt.Sprintf("level:comment_xp:%d:%s", userID, today())

	earned, _ := s.rdb.Get(ctx, key).Int()
	if earned >= cap {
		return 0
	}
	toAdd := base
	if earned+toAdd > cap {
		toAdd = cap - earned
	}
	pipe := s.rdb.Pipeline()
	pipe.IncrBy(ctx, key, int64(toAdd))
	pipe.Expire(ctx, key, 25*time.Hour)
	pipe.Exec(ctx)

	s.userRepo.AddExperience(ctx, userID, toAdd)
	return toAdd
}

// 每日签到
func (s *Service) Checkin(ctx context.Context, userID uint64, isVIP bool) (int, bool) {
	key := checkinKey(userID, today())
	ok, _ := s.rdb.SetNX(ctx, key, 1, 25*time.Hour).Result()
	if !ok {
		return 0, true // 已签到
	}
	xp := xpCheckin
	if isVIP {
		xp *= 2
	}
	s.userRepo.AddExperience(ctx, userID, xp)
	return xp, false
}

// 查等级信息
func (s *Service) GetLevelInfo(ctx context.Context, userID uint64) (*LevelInfo, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	checkedIn := false
	if n, _ := s.rdb.Exists(ctx, checkinKey(userID, today())).Result(); n > 0 {
		checkedIn = true
	}
	info := BuildLevelInfo(u.Experience, checkedIn)
	return &info, nil
}
