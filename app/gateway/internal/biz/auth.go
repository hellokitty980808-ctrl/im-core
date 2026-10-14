package biz

import (
	"context"
	"strings"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/app/gateway/internal/data"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// Authenticator 验证 Redis 有效会话后设置内部身份，忽略客户端身份元数据。
type Authenticator struct{ clients *data.Clients }

func NewAuthenticator(clients *data.Clients) *Authenticator { return &Authenticator{clients: clients} }
func (a *Authenticator) Middleware() middleware.Middleware {
	public := map[string]bool{
		pb.OperationAccountServiceCreateCaptcha: true,
		pb.OperationAccountServiceRegister:      true,
		pb.OperationAccountServiceLogin:         true,
	}
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return nil, errors.Unauthorized("AUTH_REQUIRED", "请先登录")
			}
			if public[tr.Operation()] {
				return next(ctx, req)
			}
			header := tr.RequestHeader().Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				return nil, errors.Unauthorized("AUTH_REQUIRED", "请先登录")
			}
			identity, err := a.clients.Accounts.Authenticate(ctx, &pb.SessionToken{Token: strings.TrimPrefix(header, "Bearer ")})
			if err != nil {
				return nil, err
			}
			return next(platform.ForwardIdentity(ctx, identity.UserId, identity.SessionId), req)
		}
	}
}
