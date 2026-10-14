package service

import (
	"context"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"

	"google.golang.org/protobuf/types/known/emptypb"
)

type ConversationProxy struct {
	pb.UnimplementedConversationServiceServer
	Client pb.ConversationServiceClient
}

func (s *ConversationProxy) ListConversations(ctx context.Context, req *pb.Page) (*pb.Conversations, error) {
	return s.Client.ListConversations(ctx, req)
}

func (s *ConversationProxy) GetConversation(ctx context.Context, req *pb.ConversationID) (*pb.Conversation, error) {
	return s.Client.GetConversation(ctx, req)
}

func (s *ConversationProxy) CreateGroup(ctx context.Context, req *pb.GroupDraft) (*pb.Group, error) {
	return s.Client.CreateGroup(ctx, req)
}

func (s *ConversationProxy) GetGroup(ctx context.Context, req *pb.GroupID) (*pb.Group, error) {
	return s.Client.GetGroup(ctx, req)
}

func (s *ConversationProxy) UpdateGroup(ctx context.Context, req *pb.GroupUpdate) (*pb.Group, error) {
	return s.Client.UpdateGroup(ctx, req)
}

func (s *ConversationProxy) ListMembers(ctx context.Context, req *pb.GroupID) (*pb.GroupMembers, error) {
	return s.Client.ListMembers(ctx, req)
}

func (s *ConversationProxy) InviteMembers(ctx context.Context, req *pb.MemberInvite) (*pb.Group, error) {
	return s.Client.InviteMembers(ctx, req)
}

func (s *ConversationProxy) RemoveMember(ctx context.Context, req *pb.MemberID) (*emptypb.Empty, error) {
	return s.Client.RemoveMember(ctx, req)
}

func (s *ConversationProxy) LeaveGroup(ctx context.Context, req *pb.GroupID) (*emptypb.Empty, error) {
	return s.Client.LeaveGroup(ctx, req)
}

func (s *ConversationProxy) CloseGroup(ctx context.Context, req *pb.GroupID) (*emptypb.Empty, error) {
	return s.Client.CloseGroup(ctx, req)
}

func (s *ConversationProxy) RequestJoin(ctx context.Context, req *pb.JoinDraft) (*pb.JoinRequest, error) {
	return s.Client.RequestJoin(ctx, req)
}

func (s *ConversationProxy) ListJoinRequests(ctx context.Context, req *pb.GroupID) (*pb.JoinRequests, error) {
	return s.Client.ListJoinRequests(ctx, req)
}

func (s *ConversationProxy) AcceptJoinRequest(ctx context.Context, req *pb.JoinRequestID) (*pb.JoinRequest, error) {
	return s.Client.AcceptJoinRequest(ctx, req)
}

func (s *ConversationProxy) RejectJoinRequest(ctx context.Context, req *pb.JoinRequestID) (*pb.JoinRequest, error) {
	return s.Client.RejectJoinRequest(ctx, req)
}

func (s *ConversationProxy) SendMessage(ctx context.Context, req *pb.MessageDraft) (*pb.Message, error) {
	return s.Client.SendMessage(ctx, req)
}

func (s *ConversationProxy) ListMessages(ctx context.Context, req *pb.MessageQuery) (*pb.Messages, error) {
	return s.Client.ListMessages(ctx, req)
}

func (s *ConversationProxy) RecallMessage(ctx context.Context, req *pb.MessageID) (*pb.Message, error) {
	return s.Client.RecallMessage(ctx, req)
}

func (s *ConversationProxy) UpdateReadProgress(ctx context.Context, req *pb.ReadProgress) (*pb.ReadProgress, error) {
	return s.Client.UpdateReadProgress(ctx, req)
}

func (s *ConversationProxy) GetReadProgress(ctx context.Context, req *pb.ConversationID) (*pb.ReadProgress, error) {
	return s.Client.GetReadProgress(ctx, req)
}

func (s *ConversationProxy) PullEvents(ctx context.Context, req *pb.Page) (*pb.Events, error) {
	return s.Client.PullEvents(ctx, req)
}

func (s *ConversationProxy) AckEvents(ctx context.Context, req *pb.EventAck) (*emptypb.Empty, error) {
	return s.Client.AckEvents(ctx, req)
}
