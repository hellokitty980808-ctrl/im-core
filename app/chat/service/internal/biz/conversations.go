package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/app/chat/service/internal/data"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"
	"github.com/hellokitty980808-ctrl/im-core/pkg/storage"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ConversationUseCase 管理私聊、群会话、消息和成员事件。
type ConversationUseCase struct{ repo *data.Repository }

func NewConversationUseCase(repo *data.Repository) *ConversationUseCase {
	return &ConversationUseCase{repo: repo}
}
func bad(message string) error { return kerrors.BadRequest("INVALID_ARGUMENT", message) }
func missing() error           { return kerrors.NotFound("NOT_FOUND", "记录不存在") }
func forbidden() error         { return kerrors.Forbidden("PERMISSION_DENIED", "无权执行此操作") }
func pageLimit(n uint32) int {
	if n == 0 {
		return 30
	}
	if n > 100 {
		return 100
	}
	return int(n)
}
func (u *ConversationUseCase) EnsureDirect(ctx context.Context, in *pb.DirectPair) (*pb.ConversationID, error) {
	a, b := in.FirstUserId, in.SecondUserId
	if a == 0 || b == 0 || a == b {
		return nil, bad("私聊双方无效")
	}
	if a > b {
		a, b = b, a
	}
	key := fmt.Sprintf("%d:%d", a, b)
	var row storage.Conversation
	err := u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		// 唯一索引处理不同请求并发建立同一私聊；锁行后读取已有会话。
		row = storage.Conversation{
			Type:      int32(pb.ConversationType_DIRECT),
			DirectKey: &key,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("direct_key = ?", key).First(&row).Error; err != nil {
			return err
		}
		for _, id := range []uint64{a, b} {
			member := storage.ConversationMember{
				ConversationID: row.ID,
				UserID:         id,
				State:          int32(pb.ConversationState_ACTIVE),
				JoinedSeq:      1,
				JoinedAt:       now,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return &pb.ConversationID{ConversationId: row.ID}, err
}
func membership(tx *gorm.DB, conversation, user uint64, active bool) (*storage.ConversationMember, error) {
	if conversation == 0 {
		return nil, bad("会话 ID 不能为空")
	}
	var row storage.ConversationMember
	err := tx.Where("conversation_id = ? AND user_id = ?", conversation, user).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, forbidden()
	}
	if err != nil {
		return nil, err
	}
	if active && row.State != int32(pb.ConversationState_ACTIVE) {
		return nil, forbidden()
	}
	return &row, nil
}
func (u *ConversationUseCase) conversation(ctx context.Context, row storage.Conversation, member storage.ConversationMember) (*pb.Conversation, error) {
	out := &pb.Conversation{
		ConversationId: row.ID,
		Type:           pb.ConversationType(row.Type),
		State:          pb.ConversationState(member.State),
		LastMessageSeq: row.LastMessageSeq,
		ReadSeq:        member.ReadSeq,
		UpdatedAt:      row.UpdatedAt,
	}
	if row.Type == int32(pb.ConversationType_DIRECT) {
		var peer storage.ConversationMember
		if err := u.repo.DB.WithContext(ctx).Where("conversation_id = ? AND user_id <> ?", row.ID, member.UserID).First(&peer).Error; err != nil {
			return nil, err
		}
		info, err := u.repo.Users.GetProfile(ctx, &pb.UserID{UserId: peer.UserID})
		if err != nil {
			return nil, err
		}
		out.Peer = info
	} else {
		var group storage.Group
		if err := u.repo.DB.WithContext(ctx).Where("conversation_id = ?", row.ID).First(&group).Error; err != nil {
			return nil, err
		}
		out.GroupId = group.ID
		out.Name = group.Name
	}
	return out, nil
}
func (u *ConversationUseCase) GetConversation(ctx context.Context, in *pb.ConversationID) (*pb.Conversation, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	member, err := membership(u.repo.DB.WithContext(ctx), in.ConversationId, id, false)
	if err != nil {
		return nil, err
	}
	var row storage.Conversation
	if err = u.repo.DB.WithContext(ctx).First(&row, in.ConversationId).Error; err != nil {
		return nil, err
	}
	return u.conversation(ctx, row, *member)
}
func (u *ConversationUseCase) ListConversations(ctx context.Context, in *pb.Page) (*pb.Conversations, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	var members []storage.ConversationMember
	if err = u.repo.DB.WithContext(ctx).Where("user_id = ? AND conversation_id > ?", id, in.Cursor).Order("conversation_id").Limit(pageLimit(in.Limit)).Find(&members).Error; err != nil {
		return nil, err
	}
	out := &pb.Conversations{NextCursor: in.Cursor}
	for _, member := range members {
		conversation, err := u.GetConversation(ctx, &pb.ConversationID{ConversationId: member.ConversationID})
		if err != nil {
			return nil, err
		}
		out.Conversations = append(out.Conversations, conversation)
		out.NextCursor = member.ConversationID
	}
	return out, nil
}
func (u *ConversationUseCase) PullEvents(ctx context.Context, in *pb.Page) (*pb.Events, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	var rows []storage.Event
	if err = u.repo.DB.WithContext(ctx).Table("conversation_events").Where("user_id = ? AND id > ?", id, in.Cursor).Order("id").Limit(pageLimit(in.Limit)).Find(&rows).Error; err != nil {
		return nil, err
	}
	var cursor struct{ AcknowledgedCursor uint64 }
	if err = u.repo.DB.WithContext(ctx).Table("event_cursors").Where("user_id = ?", id).Find(&cursor).Error; err != nil {
		return nil, err
	}
	out := &pb.Events{
		NextCursor:         in.Cursor,
		AcknowledgedCursor: cursor.AcknowledgedCursor,
	}
	for _, row := range rows {
		event := &pb.Event{}
		if err = protojson.Unmarshal([]byte(row.Data), event); err != nil {
			return nil, err
		}
		out.Events = append(out.Events, event)
		out.NextCursor = row.ID
	}
	return out, nil
}
func (u *ConversationUseCase) AckEvents(ctx context.Context, in *pb.EventAck) (*emptypb.Empty, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	var max uint64
	if err = u.repo.DB.WithContext(ctx).Table("conversation_events").Where("user_id = ?", id).Select("COALESCE(MAX(id),0)").Scan(&max).Error; err != nil {
		return nil, err
	}
	if in.Cursor > max {
		return nil, bad("事件确认游标超出范围")
	}
	err = u.repo.DB.WithContext(ctx).Exec(
		"INSERT INTO event_cursors (user_id,acknowledged_cursor) VALUES (?,?) ON DUPLICATE KEY UPDATE acknowledged_cursor=GREATEST(acknowledged_cursor,VALUES(acknowledged_cursor))",
		id,
		in.Cursor,
	).Error
	return &emptypb.Empty{}, err
}
