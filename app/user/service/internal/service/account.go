package service

import (
	"context"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/app/user/service/internal/biz"

	"google.golang.org/protobuf/types/known/emptypb"
)

type AccountService struct {
	pb.UnimplementedAccountServiceServer
	usecase *biz.AccountUseCase
}

func NewAccountService(u *biz.AccountUseCase) *AccountService { return &AccountService{usecase: u} }
func (s *AccountService) CreateCaptcha(ctx context.Context, in *emptypb.Empty) (*pb.Captcha, error) {
	return s.usecase.CreateCaptcha(ctx, in)
}
func (s *AccountService) Register(ctx context.Context, in *pb.Registration) (*pb.Session, error) {
	return s.usecase.Register(ctx, in)
}
func (s *AccountService) Login(ctx context.Context, in *pb.Credentials) (*pb.Session, error) {
	return s.usecase.Login(ctx, in)
}
func (s *AccountService) Refresh(ctx context.Context, in *emptypb.Empty) (*pb.Session, error) {
	return s.usecase.Refresh(ctx, in)
}
func (s *AccountService) Logout(ctx context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return s.usecase.Logout(ctx, in)
}
func (s *AccountService) GetMe(ctx context.Context, in *emptypb.Empty) (*pb.Profile, error) {
	return s.usecase.GetMe(ctx, in)
}
func (s *AccountService) UpdateMe(ctx context.Context, in *pb.ProfileUpdate) (*pb.Profile, error) {
	return s.usecase.UpdateMe(ctx, in)
}
func (s *AccountService) ChangePassword(ctx context.Context, in *pb.PasswordChange) (*emptypb.Empty, error) {
	return s.usecase.ChangePassword(ctx, in)
}
func (s *AccountService) FindUser(ctx context.Context, in *pb.AccountQuery) (*pb.Profile, error) {
	return s.usecase.FindUser(ctx, in)
}
func (s *AccountService) ListFriends(ctx context.Context, in *pb.Page) (*pb.Friends, error) {
	return s.usecase.ListFriends(ctx, in)
}
func (s *AccountService) GetFriend(ctx context.Context, in *pb.UserID) (*pb.Friend, error) {
	return s.usecase.GetFriend(ctx, in)
}
func (s *AccountService) UpdateFriend(ctx context.Context, in *pb.FriendUpdate) (*pb.Friend, error) {
	return s.usecase.UpdateFriend(ctx, in)
}
func (s *AccountService) DeleteFriend(ctx context.Context, in *pb.UserID) (*emptypb.Empty, error) {
	return s.usecase.DeleteFriend(ctx, in)
}
func (s *AccountService) BlockUser(ctx context.Context, in *pb.UserID) (*emptypb.Empty, error) {
	return s.usecase.BlockUser(ctx, in)
}
func (s *AccountService) UnblockUser(ctx context.Context, in *pb.UserID) (*emptypb.Empty, error) {
	return s.usecase.UnblockUser(ctx, in)
}
func (s *AccountService) ListBlocks(ctx context.Context, in *pb.Page) (*pb.Friends, error) {
	return s.usecase.ListBlocks(ctx, in)
}
func (s *AccountService) CreateFriendRequest(ctx context.Context, in *pb.FriendRequestDraft) (*pb.FriendRequest, error) {
	return s.usecase.CreateFriendRequest(ctx, in)
}
func (s *AccountService) ListFriendRequests(ctx context.Context, in *pb.Page) (*pb.FriendRequests, error) {
	return s.usecase.ListFriendRequests(ctx, in)
}
func (s *AccountService) AcceptFriendRequest(ctx context.Context, in *pb.FriendRequestID) (*pb.FriendRequest, error) {
	return s.usecase.AcceptFriendRequest(ctx, in)
}
func (s *AccountService) RejectFriendRequest(ctx context.Context, in *pb.FriendRequestID) (*pb.FriendRequest, error) {
	return s.usecase.RejectFriendRequest(ctx, in)
}
func (s *AccountService) CancelFriendRequest(ctx context.Context, in *pb.FriendRequestID) (*pb.FriendRequest, error) {
	return s.usecase.CancelFriendRequest(ctx, in)
}
func (s *AccountService) PullFriendEvents(ctx context.Context, in *pb.Page) (*pb.Events, error) {
	return s.usecase.PullFriendEvents(ctx, in)
}
func (s *AccountService) AckFriendEvents(ctx context.Context, in *pb.EventAck) (*emptypb.Empty, error) {
	return s.usecase.AckFriendEvents(ctx, in)
}
func (s *AccountService) Authenticate(ctx context.Context, in *pb.SessionToken) (*pb.Identity, error) {
	return s.usecase.Authenticate(ctx, in)
}
func (s *AccountService) GetProfile(ctx context.Context, in *pb.UserID) (*pb.Profile, error) {
	return s.usecase.GetProfile(ctx, in)
}
func (s *AccountService) CheckRelationship(ctx context.Context, in *pb.RelationshipQuery) (*pb.RelationshipCheck, error) {
	return s.usecase.CheckRelationship(ctx, in)
}
