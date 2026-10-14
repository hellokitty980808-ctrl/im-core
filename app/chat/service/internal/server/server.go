package server

import (
	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/app/chat/service/internal/service"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"

	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

func NewGRPCServer(r *platform.Runtime, s *service.ConversationService) *grpc.Server {
	srv := grpc.NewServer(grpc.Address(r.Config.GrpcAddr), grpc.Middleware(recovery.Recovery()))
	pb.RegisterConversationServiceServer(srv, s)
	return srv
}
