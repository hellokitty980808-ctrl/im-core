package platform

import (
	"context"
	"strconv"

	"github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc/metadata"
)

// Identity 从内部 RPC 元数据读取用户身份；HTTP 客户端不能直接指定这个值。
func Identity(ctx context.Context) (uint64, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get("imcore-user-id")
	if len(values) != 1 {
		return 0, errors.Unauthorized("AUTH_REQUIRED", "请先登录")
	}
	id, err := strconv.ParseUint(values[0], 10, 64)
	if err != nil || id == 0 {
		return 0, errors.Unauthorized("AUTH_REQUIRED", "请先登录")
	}
	return id, nil
}

// ForwardIdentity 设置由认证服务确认的身份和会话；覆盖任何客户端输入。
func ForwardIdentity(ctx context.Context, id uint64, session string) context.Context {
	return metadata.NewOutgoingContext(
		ctx,
		metadata.Pairs("imcore-user-id", strconv.FormatUint(id, 10), "imcore-session-id", session),
	)
}
func SessionID(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	v := md.Get("imcore-session-id")
	if len(v) == 1 {
		return v[0]
	}
	return ""
}
