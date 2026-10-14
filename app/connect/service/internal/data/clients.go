package data

import (
	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"
)

// NewAccountClient 建立认证 RPC 连接，返回应用退出时使用的关闭函数。
func NewAccountClient(r *platform.Runtime) (pb.AccountServiceClient, func(), error) {
	conn, err := r.Dial("user")
	if err != nil {
		return nil, nil, err
	}
	return pb.NewAccountServiceClient(conn), func() { _ = conn.Close() }, nil
}
