package data

import (
	"context"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"
	"github.com/hellokitty980808-ctrl/im-core/pkg/session"
	"github.com/hellokitty980808-ctrl/im-core/pkg/storage"

	driver "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

// Repository 持有账户库、会话缓存和会话服务客户端。
type Repository struct {
	DB         *gorm.DB
	Redis      *redis.Client
	Sessions   *session.Manager
	Chat       pb.ConversationServiceClient
	connection *grpc.ClientConn
}

// NewRepository 执行账户迁移并建立数据库、Redis 和会话服务连接。
func NewRepository(r *platform.Runtime) (*Repository, func(), error) {
	if err := storage.Migrate(context.Background(), "user", r.Config.MysqlDsn); err != nil {
		return nil, nil, err
	}
	db, err := storage.Open(r.Config.MysqlDsn)
	if err != nil {
		return nil, nil, err
	}
	pool, _ := db.DB()
	rdb := redis.NewClient(&redis.Options{
		Addr:     r.Config.RedisAddr,
		Password: r.Config.RedisPassword,
	})
	close := func() {
		_ = rdb.Close()
		_ = pool.Close()
	}
	cfg, err := driver.ParseDSN(r.Config.MysqlDsn)
	if err != nil {
		close()
		return nil, nil, err
	}
	manager, err := session.New(r.Config.PrivateKey, r.Config.PublicKey, cfg.DBName, rdb)
	if err != nil {
		close()
		return nil, nil, err
	}
	conn, err := r.Dial("chat")
	if err != nil {
		close()
		return nil, nil, err
	}
	repo := &Repository{
		DB:         db,
		Redis:      rdb,
		Sessions:   manager,
		Chat:       pb.NewConversationServiceClient(conn),
		connection: conn,
	}
	return repo, func() {
		_ = conn.Close()
		close()
	}, nil
}
