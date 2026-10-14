package server

import (
	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/app/connect/service/internal/biz"
	"github.com/hellokitty980808-ctrl/im-core/app/connect/service/internal/service"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"

	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

func NewHTTPServer(r *platform.Runtime, h *biz.Hub) *http.Server {
	srv := http.NewServer(http.Address(r.Config.HttpAddr))
	srv.Handle("/v2/ws", h)
	return srv
}
func NewGRPCServer(r *platform.Runtime, s *service.DeliveryService) *grpc.Server {
	srv := grpc.NewServer(grpc.Address(r.Config.GrpcAddr), grpc.Middleware(recovery.Recovery()))
	pb.RegisterDeliveryServiceServer(srv, s)
	return srv
}
