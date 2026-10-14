package data

import (
	"context"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"
	"github.com/hellokitty980808-ctrl/im-core/pkg/storage"

	"google.golang.org/grpc"
	"gorm.io/gorm"
)

// Repository 持有会话库和账户服务客户端，不复制用户密码或登录状态。
type Repository struct {
	DB         *gorm.DB
	Users      pb.AccountServiceClient
	connection *grpc.ClientConn
}

// NewRepository 执行会话迁移并建立数据库和账户服务连接。
func NewRepository(r *platform.Runtime) (*Repository, func(), error) {
	if err := storage.Migrate(context.Background(), "chat", r.Config.MysqlDsn); err != nil {
		return nil, nil, err
	}
	db, err := storage.Open(r.Config.MysqlDsn)
	if err != nil {
		return nil, nil, err
	}
	pool, _ := db.DB()
	conn, err := r.Dial("user")
	if err != nil {
		_ = pool.Close()
		return nil, nil, err
	}
	return &Repository{
			DB:         db,
			Users:      pb.NewAccountServiceClient(conn),
			connection: conn,
		}, func() {
			_ = conn.Close()
			_ = pool.Close()
		}, nil
}
