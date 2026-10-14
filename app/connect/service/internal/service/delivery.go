package service

import (
	"context"
	"encoding/json"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/app/connect/service/internal/biz"
	"github.com/hellokitty980808-ctrl/im-core/pkg/delivery"

	"github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DeliveryService struct {
	pb.UnimplementedDeliveryServiceServer
	hub *biz.Hub
}

func NewDeliveryService(h *biz.Hub) *DeliveryService { return &DeliveryService{hub: h} }
func (s *DeliveryService) Push(ctx context.Context, in *pb.Delivery) (*emptypb.Empty, error) {
	if len(in.UserIds) == 0 || !json.Valid(in.Data) || (in.Type != "message" && in.Type != "event") {
		return nil, errors.BadRequest("INVALID_ARGUMENT", "投递内容无效")
	}
	s.hub.Deliver(delivery.Envelope{
		Version: 2,
		Type:    in.Type,
		Users:   in.UserIds,
		Data:    in.Data,
	})
	return &emptypb.Empty{}, nil
}
