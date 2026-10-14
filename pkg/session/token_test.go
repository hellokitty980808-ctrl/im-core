package session

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestVerifyRejectsOtherEnvironmentAndAlgorithm(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	manager := &Manager{
		public:    &key.PublicKey,
		namespace: "isolated_accounts",
	}
	cases := []struct {
		name       string
		issuer     string
		algorithm  jwt.SigningMethod
		signingKey any
	}{
		{
			name:       "其他数据库会话",
			issuer:     "github.com/hellokitty980808-ctrl/im-core/other_accounts",
			algorithm:  jwt.SigningMethodRS256,
			signingKey: key,
		},
		{
			name:       "错误签名算法",
			issuer:     "github.com/hellokitty980808-ctrl/im-core/isolated_accounts",
			algorithm:  jwt.SigningMethodHS256,
			signingKey: []byte("test-only-key"),
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			claims := jwt.RegisteredClaims{
				Issuer:    test.issuer,
				Subject:   "42",
				ID:        "test-session",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			}
			token, err := jwt.NewWithClaims(test.algorithm, claims).SignedString(test.signingKey)
			if err != nil {
				t.Fatal(err)
			}
			// 无 Redis 客户端：被拒绝的 Token 不应到达会话缓存查询。
			if _, _, err = manager.Verify(context.Background(), token); err == nil {
				t.Fatal("跨环境或错误算法的 Token 被接受")
			}
		})
	}
}
