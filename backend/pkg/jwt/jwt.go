package jwt

import (
	"context"
	"errors"
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

var ErrBlacklisted = errors.New("token has been revoked")

type Claims struct {
	UserID uint64 `json:"uid"`
	gojwt.RegisteredClaims
}

type Manager struct {
	secret []byte
	expire time.Duration
	rdb    *redis.Client
}

// 创建 JWT 管理器
func NewManager(secret string, expire time.Duration, rdb *redis.Client) *Manager {
	return &Manager{
		secret: []byte(secret),
		expire: expire,
		rdb:    rdb,
	}
}

// 签发 token
func (m *Manager) Generate(userID uint64) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: gojwt.RegisteredClaims{
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(m.expire)),
		},
	}
	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// 解析校验 token
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	token, err := gojwt.ParseWithClaims(tokenStr, &Claims{}, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// 加入登出黑名单
func (m *Manager) Blacklist(ctx context.Context, tokenStr string, expireAt time.Time) error {
	ttl := time.Until(expireAt)
	if ttl <= 0 {
		return nil
	}
	key := fmt.Sprintf("jwt:blacklist:%s", tokenStr)
	return m.rdb.Set(ctx, key, 1, ttl).Err()
}

// 是否在黑名单
func (m *Manager) IsBlacklisted(ctx context.Context, tokenStr string) (bool, error) {
	key := fmt.Sprintf("jwt:blacklist:%s", tokenStr)
	n, err := m.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
