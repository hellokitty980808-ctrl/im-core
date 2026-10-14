package service

import (
	"context"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"

	"google.golang.org/protobuf/types/known/emptypb"
)

type AccountProxy struct {
	pb.UnimplementedAccountServiceServer
	Client pb.AccountServiceClient
}

func (s *AccountProxy) CreateCaptcha(ctx context.Context, req *emptypb.Empty) (*pb.Captcha, error) {
	return s.Client.CreateCaptcha(ctx, req)
}

func (s *AccountProxy) Register(ctx context.Context, req *pb.Registration) (*pb.Session, error) {
	return s.Client.Register(ctx, req)
}

func (s *AccountProxy) Login(ctx context.Context, req *pb.Credentials) (*pb.Session, error) {
	return s.Client.Login(ctx, req)
}

func (s *AccountProxy) Refresh(ctx context.Context, req *emptypb.Empty) (*pb.Session, error) {
	return s.Client.Refresh(ctx, req)
}

func (s *AccountProxy) Logout(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	return s.Client.Logout(ctx, req)
}

func (s *AccountProxy) GetMe(ctx context.Context, req *emptypb.Empty) (*pb.Profile, error) {
	return s.Client.GetMe(ctx, req)
}

func (s *AccountProxy) UpdateMe(ctx context.Context, req *pb.ProfileUpdate) (*pb.Profile, error) {
	return s.Client.UpdateMe(ctx, req)
}

func (s *AccountProxy) ChangePassword(ctx context.Context, req *pb.PasswordChange) (*emptypb.Empty, error) {
	return s.Client.ChangePassword(ctx, req)
}

func (s *AccountProxy) FindUser(ctx context.Context, req *pb.AccountQuery) (*pb.Profile, error) {
	return s.Client.FindUser(ctx, req)
}

func (s *AccountProxy) ListFriends(ctx context.Context, req *pb.Page) (*pb.Friends, error) {
	return s.Client.ListFriends(ctx, req)
}

func (s *AccountProxy) GetFriend(ctx context.Context, req *pb.UserID) (*pb.Friend, error) {
	return s.Client.GetFriend(ctx, req)
}

func (s *AccountProxy) UpdateFriend(ctx context.Context, req *pb.FriendUpdate) (*pb.Friend, error) {
	return s.Client.UpdateFriend(ctx, req)
}

func (s *AccountProxy) DeleteFriend(ctx context.Context, req *pb.UserID) (*emptypb.Empty, error) {
	return s.Client.DeleteFriend(ctx, req)
}

func (s *AccountProxy) BlockUser(ctx context.Context, req *pb.UserID) (*emptypb.Empty, error) {
	return s.Client.BlockUser(ctx, req)
}

func (s *AccountProxy) UnblockUser(ctx context.Context, req *pb.UserID) (*emptypb.Empty, error) {
	return s.Client.UnblockUser(ctx, req)
}

func (s *AccountProxy) ListBlocks(ctx context.Context, req *pb.Page) (*pb.Friends, error) {
	return s.Client.ListBlocks(ctx, req)
}

func (s *AccountProxy) CreateFriendRequest(ctx context.Context, req *pb.FriendRequestDraft) (*pb.FriendRequest, error) {
	return s.Client.CreateFriendRequest(ctx, req)
}

func (s *AccountProxy) ListFriendRequests(ctx context.Context, req *pb.Page) (*pb.FriendRequests, error) {
	return s.Client.ListFriendRequests(ctx, req)
}

func (s *AccountProxy) AcceptFriendRequest(ctx context.Context, req *pb.FriendRequestID) (*pb.FriendRequest, error) {
	return s.Client.AcceptFriendRequest(ctx, req)
}

func (s *AccountProxy) RejectFriendRequest(ctx context.Context, req *pb.FriendRequestID) (*pb.FriendRequest, error) {
	return s.Client.RejectFriendRequest(ctx, req)
}

func (s *AccountProxy) CancelFriendRequest(ctx context.Context, req *pb.FriendRequestID) (*pb.FriendRequest, error) {
	return s.Client.CancelFriendRequest(ctx, req)
}

func (s *AccountProxy) PullFriendEvents(ctx context.Context, req *pb.Page) (*pb.Events, error) {
	return s.Client.PullFriendEvents(ctx, req)
}

func (s *AccountProxy) AckFriendEvents(ctx context.Context, req *pb.EventAck) (*emptypb.Empty, error) {
	return s.Client.AckFriendEvents(ctx, req)
}
