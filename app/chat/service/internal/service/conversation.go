package service

import (
	"context"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/app/chat/service/internal/biz"

	"google.golang.org/protobuf/types/known/emptypb"
)

type ConversationService struct {
	pb.UnimplementedConversationServiceServer
	usecase *biz.ConversationUseCase
}

func NewConversationService(u *biz.ConversationUseCase) *ConversationService {
	return &ConversationService{usecase: u}
}
func (s *ConversationService) ListConversations(ctx context.Context, in *pb.Page) (*pb.Conversations, error) {
	return s.usecase.ListConversations(ctx, in)
}
func (s *ConversationService) GetConversation(ctx context.Context, in *pb.ConversationID) (*pb.Conversation, error) {
	return s.usecase.GetConversation(ctx, in)
}
func (s *ConversationService) CreateGroup(ctx context.Context, in *pb.GroupDraft) (*pb.Group, error) {
	return s.usecase.CreateGroup(ctx, in)
}
func (s *ConversationService) GetGroup(ctx context.Context, in *pb.GroupID) (*pb.Group, error) {
	return s.usecase.GetGroup(ctx, in)
}
func (s *ConversationService) UpdateGroup(ctx context.Context, in *pb.GroupUpdate) (*pb.Group, error) {
	return s.usecase.UpdateGroup(ctx, in)
}
func (s *ConversationService) ListMembers(ctx context.Context, in *pb.GroupID) (*pb.GroupMembers, error) {
	return s.usecase.ListMembers(ctx, in)
}
func (s *ConversationService) InviteMembers(ctx context.Context, in *pb.MemberInvite) (*pb.Group, error) {
	return s.usecase.InviteMembers(ctx, in)
}
func (s *ConversationService) RemoveMember(ctx context.Context, in *pb.MemberID) (*emptypb.Empty, error) {
	return s.usecase.RemoveMember(ctx, in)
}
func (s *ConversationService) LeaveGroup(ctx context.Context, in *pb.GroupID) (*emptypb.Empty, error) {
	return s.usecase.LeaveGroup(ctx, in)
}
func (s *ConversationService) CloseGroup(ctx context.Context, in *pb.GroupID) (*emptypb.Empty, error) {
	return s.usecase.CloseGroup(ctx, in)
}
func (s *ConversationService) RequestJoin(ctx context.Context, in *pb.JoinDraft) (*pb.JoinRequest, error) {
	return s.usecase.RequestJoin(ctx, in)
}
func (s *ConversationService) ListJoinRequests(ctx context.Context, in *pb.GroupID) (*pb.JoinRequests, error) {
	return s.usecase.ListJoinRequests(ctx, in)
}
func (s *ConversationService) AcceptJoinRequest(ctx context.Context, in *pb.JoinRequestID) (*pb.JoinRequest, error) {
	return s.usecase.AcceptJoinRequest(ctx, in)
}
func (s *ConversationService) RejectJoinRequest(ctx context.Context, in *pb.JoinRequestID) (*pb.JoinRequest, error) {
	return s.usecase.RejectJoinRequest(ctx, in)
}
func (s *ConversationService) SendMessage(ctx context.Context, in *pb.MessageDraft) (*pb.Message, error) {
	return s.usecase.SendMessage(ctx, in)
}
func (s *ConversationService) ListMessages(ctx context.Context, in *pb.MessageQuery) (*pb.Messages, error) {
	return s.usecase.ListMessages(ctx, in)
}
func (s *ConversationService) RecallMessage(ctx context.Context, in *pb.MessageID) (*pb.Message, error) {
	return s.usecase.RecallMessage(ctx, in)
}
func (s *ConversationService) UpdateReadProgress(ctx context.Context, in *pb.ReadProgress) (*pb.ReadProgress, error) {
	return s.usecase.UpdateReadProgress(ctx, in)
}
func (s *ConversationService) GetReadProgress(ctx context.Context, in *pb.ConversationID) (*pb.ReadProgress, error) {
	return s.usecase.GetReadProgress(ctx, in)
}
func (s *ConversationService) PullEvents(ctx context.Context, in *pb.Page) (*pb.Events, error) {
	return s.usecase.PullEvents(ctx, in)
}
func (s *ConversationService) AckEvents(ctx context.Context, in *pb.EventAck) (*emptypb.Empty, error) {
	return s.usecase.AckEvents(ctx, in)
}
func (s *ConversationService) EnsureDirect(ctx context.Context, in *pb.DirectPair) (*pb.ConversationID, error) {
	return s.usecase.EnsureDirect(ctx, in)
}
