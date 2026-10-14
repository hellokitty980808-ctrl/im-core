// Package session 提供 RS256 会话签发及 Redis 单会话撤销。
package session

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Manager 每个账户只保留一个有效会话，刷新使用原子比较替换。
type Manager struct {
	private   *rsa.PrivateKey
	public    *rsa.PublicKey
	redis     *redis.Client
	namespace string
}

// New 加载环境中的 RSA 密钥，并按数据库命名空间隔离会话。
func New(private, public, namespace string, rdb *redis.Client) (*Manager, error) {
	p, err := base64.StdEncoding.DecodeString(private)
	if err != nil {
		return nil, err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(p)
	if err != nil {
		return nil, err
	}
	p, err = base64.StdEncoding.DecodeString(public)
	if err != nil {
		return nil, err
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(p)
	if err != nil {
		return nil, err
	}
	return &Manager{
		private:   privateKey,
		public:    publicKey,
		redis:     rdb,
		namespace: namespace,
	}, nil
}
func (m *Manager) key(id uint64) string { return fmt.Sprintf("imcore:%s:session:%d", m.namespace, id) }

// Issue expected 为空表示登录替换；非空表示刷新，只有当前会话能替换自己。
func (m *Manager) Issue(ctx context.Context, id uint64, expected string) (string, int64, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    "github.com/hellokitty980808-ctrl/im-core/" + m.namespace,
		Subject:   strconv.FormatUint(id, 10),
		ID:        uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(m.private)
	if err != nil {
		return "", 0, err
	}
	if expected == "" {
		err = m.redis.Set(ctx, m.key(id), claims.ID, 24*time.Hour).Err()
	} else {
		changed, e := m.redis.Eval(
			ctx,
			`if redis.call('GET',KEYS[1]) ~= ARGV[1] then return 0 end redis.call('SET',KEYS[1],ARGV[2],'EX',86400) return 1`,
			[]string{m.key(id)},
			expected,
			claims.ID,
		).Int()
		if e != nil {
			return "", 0, e
		}
		if changed != 1 {
			return "", 0, errors.Unauthorized("SESSION_REVOKED", "会话已失效")
		}
	}
	return token, claims.ExpiresAt.UnixMilli(), err
}

// Verify 验证 RS256、签发环境、过期时间与 Redis 当前会话。
func (m *Manager) Verify(ctx context.Context, token string) (uint64, string, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(
		token,
		claims,
		func(*jwt.Token) (any, error) { return m.public, nil },
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer("github.com/hellokitty980808-ctrl/im-core/"+m.namespace),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, "", errors.Unauthorized("INVALID_SESSION", "登录凭据无效")
	}
	id, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || id == 0 {
		return 0, "", errors.Unauthorized("INVALID_SESSION", "登录凭据无效")
	}
	current, err := m.redis.Get(ctx, m.key(id)).Result()
	if err != nil && err != redis.Nil {
		return 0, "", err
	}
	if current != claims.ID {
		return 0, "", errors.Unauthorized("SESSION_REVOKED", "会话已失效")
	}
	return id, claims.ID, nil
}

// Revoke 撤销会话；expected 为空时撤销该账户的全部有效会话。
func (m *Manager) Revoke(ctx context.Context, id uint64, expected string) error {
	if expected == "" {
		return m.redis.Del(ctx, m.key(id)).Err()
	}
	return m.redis.Eval(
		ctx,
		`if redis.call('GET',KEYS[1]) == ARGV[1] then redis.call('DEL',KEYS[1]) end return 1`,
		[]string{m.key(id)},
		expected,
	).Err()
}
