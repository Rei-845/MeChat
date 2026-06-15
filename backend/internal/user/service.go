package user

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"path/filepath"
	"time"

	"mechat/pkg/email"
	"mechat/pkg/jwt"
	"mechat/pkg/oss"
	redisPkg "mechat/pkg/redis"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// 预定义错误
var (
	ErrUserNotFound    = errors.New("用户不存在")
	ErrEmailExists     = errors.New("邮箱已注册")
	ErrNicknameExists  = errors.New("昵称已被占用，请换一个")
	ErrCodeInvalid     = errors.New("验证码无效或已过期")
	ErrCodeSendTooFast = errors.New("发送太频繁，请稍后再试")
)

// Service 用户服务
type Service struct {
	repo   *Repository
	jwtMgr *jwt.Manager
	rdb    *redis.Client
	mailer *email.Sender
	oss    oss.Uploader
}

// 创建用户服务
func NewService(repo *Repository, jwtMgr *jwt.Manager, rdb *redis.Client, mailer *email.Sender, uploader oss.Uploader) *Service {
	return &Service{repo: repo, jwtMgr: jwtMgr, rdb: rdb, mailer: mailer, oss: uploader}
}

// 发送验证码
func (s *Service) SendCode(ctx context.Context, req *SendCodeReq) error {
	// 防刷限流 1 分钟 1 次
	limitKey := redisPkg.EmailCodeLimitKey(req.Email)
	n, err := s.rdb.Incr(ctx, limitKey).Result()
	if err == nil {
		if n == 1 {
			s.rdb.Expire(ctx, limitKey, time.Minute)
		}
		if n > 1 {
			return ErrCodeSendTooFast
		}
	}

	// 生成验证码并存库
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	record := &EmailVerifyCode{
		Email:     req.Email,
		Code:      code,
		Purpose:   req.Purpose,
		ExpiredAt: time.Now().Add(5 * time.Minute),
	}
	if err := s.repo.CreateCode(ctx, record); err != nil {
		return err
	}
	return s.mailer.SendVerifyCode(req.Email, code, req.Purpose)
}

// 注册
func (s *Service) Register(ctx context.Context, req *RegisterReq) (*UserInfo, string, error) {
	// 校验验证码
	if err := s.verifyCode(ctx, req.Email, req.Code, "register"); err != nil {
		return nil, "", err
	}

	// 邮箱查重
	if _, err := s.repo.GetByEmail(ctx, req.Email); err == nil {
		return nil, "", ErrEmailExists
	}

	// 昵称全局唯一
	if exists, _ := s.repo.NicknameExists(ctx, req.Nickname, 0); exists {
		return nil, "", ErrNicknameExists
	}

	// 密码哈希
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	// 创建用户
	u := &User{
		Email:    req.Email,
		Password: string(hash),
		Nickname: req.Nickname,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, "", err
	}

	// 直接签发 JWT
	token, err := s.jwtMgr.Generate(u.ID)
	if err != nil {
		return nil, "", err
	}
	return u.ToInfo(), token, nil
}

// 登录
func (s *Service) Login(ctx context.Context, req *LoginReq) (*UserInfo, string, error) {
	u, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, "", ErrUserNotFound
	}

	// 密码或验证码
	if req.Password != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
			return nil, "", errors.New("密码错误")
		}
	} else if req.Code != "" {
		if err := s.verifyCode(ctx, req.Email, req.Code, "login"); err != nil {
			return nil, "", err
		}
	} else {
		return nil, "", errors.New("请提供密码或验证码")
	}

	// 签发 JWT
	token, err := s.jwtMgr.Generate(u.ID)
	if err != nil {
		return nil, "", err
	}
	return u.ToInfo(), token, nil
}

// 登出 加黑名单
func (s *Service) Logout(ctx context.Context, tokenStr string, expireAt time.Time) error {
	return s.jwtMgr.Blacklist(ctx, tokenStr, expireAt)
}

// 按 ID 取用户
func (s *Service) GetByID(ctx context.Context, id uint64) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

// 取公开信息
func (s *Service) GetPublicInfo(ctx context.Context, id uint64) (*PublicInfo, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return u.ToPublicInfo(), nil
}

// 更新资料
func (s *Service) UpdateProfile(ctx context.Context, userID uint64, req *UpdateProfileReq) (*UserInfo, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if req.Nickname != "" && req.Nickname != u.Nickname {
		if exists, _ := s.repo.NicknameExists(ctx, req.Nickname, userID); exists {
			return nil, ErrNicknameExists
		}
		u.Nickname = req.Nickname
	}
	if req.Bio != "" {
		u.Bio = req.Bio
	}
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	// 失效缓存
	Invalidate(ctx, s.rdb, userID)
	return u.ToInfo(), nil
}

// putFile 存文件返回 URL
func (s *Service) putFile(dir string, reader io.Reader, size int64, filename string) (string, error) {
	if s.oss == nil {
		return "", errors.New("存储未配置")
	}
	return s.oss.PutObject(dir, filepath.Ext(filename), reader, size)
}

// 上传头像
func (s *Service) UploadAvatar(ctx context.Context, userID uint64, reader io.Reader, size int64, filename string) (string, error) {
	url, err := s.putFile("avatars", reader, size, filename)
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdateAvatar(ctx, userID, url); err != nil {
		return "", err
	}
	// 失效缓存
	Invalidate(ctx, s.rdb, userID)
	return url, nil
}

// 通用图片上传
func (s *Service) UploadImage(reader io.Reader, size int64, filename string) (string, error) {
	return s.putFile("images", reader, size, filename)
}

// 推荐用户
func (s *Service) Recommend(ctx context.Context, excludeIDs []uint64, limit int) ([]*PublicInfo, error) {
	users, err := s.repo.Recommend(ctx, excludeIDs, limit)
	if err != nil {
		return nil, err
	}
	result := make([]*PublicInfo, 0, len(users))
	for _, u := range users {
		result = append(result, u.ToPublicInfo())
	}
	return result, nil
}

// 搜索用户
func (s *Service) Search(ctx context.Context, keyword string) ([]*PublicInfo, error) {
	users, err := s.repo.Search(ctx, keyword, 20)
	if err != nil {
		return nil, err
	}
	result := make([]*PublicInfo, 0, len(users))
	for _, u := range users {
		result = append(result, u.ToPublicInfo())
	}
	return result, nil
}

// 校验验证码并标记已用
func (s *Service) verifyCode(ctx context.Context, email, code, purpose string) error {
	record, err := s.repo.GetValidCode(ctx, email, purpose)
	if err != nil {
		return err
	}
	if record == nil || record.Code != code {
		return ErrCodeInvalid
	}
	return s.repo.MarkCodeUsed(ctx, record.ID)
}
