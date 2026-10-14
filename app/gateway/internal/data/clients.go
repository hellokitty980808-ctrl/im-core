package data

import (
	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"
)

// Clients 持有内部 RPC 连接，Gateway 不直接访问业务数据库。
type Clients struct {
	Accounts      pb.AccountServiceClient
	Conversations pb.ConversationServiceClient
}

// NewClients 建立账户和会话 RPC 连接，并返回统一关闭函数。
func NewClients(r *platform.Runtime) (*Clients, func(), error) {
	accounts, err := r.Dial("user")
	if err != nil {
		return nil, nil, err
	}
	conversations, err := r.Dial("chat")
	if err != nil {
		_ = accounts.Close()
		return nil, nil, err
	}
	return &Clients{
			Accounts:      pb.NewAccountServiceClient(accounts),
			Conversations: pb.NewConversationServiceClient(conversations),
		}, func() {
			_ = accounts.Close()
			_ = conversations.Close()
		}, nil
}
