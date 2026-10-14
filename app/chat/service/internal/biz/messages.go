package biz

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/pkg/delivery"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"
	"github.com/hellokitty980808-ctrl/im-core/pkg/storage"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func message(row storage.Message, profile *pb.Profile) *pb.Message {
	content := row.Content
	if row.Recalled {
		content = ""
	}
	return &pb.Message{
		MessageId:      row.ID,
		ConversationId: row.ConversationID,
		Seq:            row.Seq,
		Sender:         profile,
		Content:        content,
		SentAt:         row.SentAt,
		Recalled:       row.Recalled,
	}
}
func requestHash(in *pb.MessageDraft) string {
	encoded, _ := json.Marshal(struct {
		Conversation uint64
		Content      string
	}{
		Conversation: in.ConversationId,
		Content:      in.Content,
	})
	return fmt.Sprintf("%x", sha256.Sum256(encoded))
}
func (u *ConversationUseCase) replay(ctx context.Context, user uint64, in *pb.MessageDraft) (*pb.Message, error) {
	var row storage.SendKey
	err := u.repo.DB.WithContext(ctx).Where("sender_id = ? AND send_key = ?", user, in.IdempotencyKey).First(&row).Error
	if err != nil {
		return nil, err
	}
	if row.RequestHash != requestHash(in) {
		return nil, kerrors.Conflict("IDEMPOTENCY_CONFLICT", "发送键已用于不同请求")
	}
	out := &pb.Message{}
	if err = protojson.Unmarshal([]byte(row.Response), out); err != nil {
		return nil, err
	}
	return out, nil
}
func (u *ConversationUseCase) SendMessage(ctx context.Context, in *pb.MessageDraft) (*pb.Message, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	if !utf8.ValidString(in.Content) || strings.TrimSpace(in.Content) == "" || len(in.Content) > 2048 {
		return nil, bad("文字消息长度为 1–2048 字节")
	}
	if len(in.IdempotencyKey) < 1 || len(in.IdempotencyKey) > 64 {
		return nil, bad("发送键长度为 1–64 字节")
	}
	for _, r := range in.IdempotencyKey {
		if r < 33 || r > 126 {
			return nil, bad("发送键仅允许可见 ASCII 字符")
		}
	}
	// 重放不依赖后来变化的成员状态，但不会允许改变原请求载荷。
	if result, err := u.replay(ctx, id, in); err == nil {
		return result, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	sender, err := u.repo.Users.GetProfile(ctx, &pb.UserID{UserId: id})
	if err != nil {
		return nil, err
	}
	var result *pb.Message
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conversation storage.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conversation, in.ConversationId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return missing()
			}
			return err
		}
		if conversation.Closed {
			return forbidden()
		}
		if _, err := membership(tx, conversation.ID, id, true); err != nil {
			return err
		}
		users, err := activeUsers(tx, conversation.ID)
		if err != nil {
			return err
		}
		if conversation.Type == int32(pb.ConversationType_DIRECT) {
			if len(users) != 2 {
				return forbidden()
			}
			allowed, err := u.repo.Users.CheckRelationship(ctx, &pb.RelationshipQuery{
				FirstUserId:  users[0],
				SecondUserId: users[1],
			})
			if err != nil {
				return err
			}
			if !allowed.Allowed {
				return forbidden()
			}
		}
		// 锁定会话后用当前读检查发送键，同一会话并发重试只提交一次。
		var existing storage.SendKey
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("sender_id = ? AND send_key = ?", id, in.IdempotencyKey).First(&existing).Error
		if err == nil {
			if existing.RequestHash != requestHash(in) {
				return kerrors.Conflict("IDEMPOTENCY_CONFLICT", "发送键已用于不同请求")
			}
			result = &pb.Message{}
			return protojson.Unmarshal([]byte(existing.Response), result)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		now := time.Now().UnixMilli()
		row := storage.Message{
			ConversationID: conversation.ID,
			Seq:            conversation.LastMessageSeq + 1,
			SenderID:       id,
			Content:        in.Content,
			SentAt:         now,
		}
		if err = tx.Create(&row).Error; err != nil {
			return err
		}
		result = message(row, sender)
		encoded, err := protojson.Marshal(result)
		if err != nil {
			return err
		}
		if err = tx.Create(&storage.SendKey{
			SenderID:    id,
			Key:         in.IdempotencyKey,
			RequestHash: requestHash(in),
			MessageID:   row.ID,
			Response:    string(encoded),
			CreatedAt:   now,
		}).Error; err != nil {
			return err
		}
		if err = tx.Model(&conversation).Updates(map[string]any{
			"last_message_seq": row.Seq,
			"updated_at":       now,
		}).Error; err != nil {
			return err
		}
		return delivery.Enqueue(tx, fmt.Sprintf("message:%d", row.ID), "message", users, encoded)
	})
	if err != nil {
		// 跨会话发送键竞争或事务死锁回滚后，读取已提交赢家；不同载荷仍然冲突。
		if replay, e := u.replay(ctx, id, in); e == nil {
			return replay, nil
		} else if !errors.Is(e, gorm.ErrRecordNotFound) {
			return nil, e
		}
		return nil, err
	}
	return result, nil
}
func (u *ConversationUseCase) ListMessages(ctx context.Context, in *pb.MessageQuery) (*pb.Messages, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	member, err := membership(u.repo.DB.WithContext(ctx), in.ConversationId, id, true)
	if err != nil {
		return nil, err
	}
	var rows []storage.Message
	if err = u.repo.DB.WithContext(ctx).Where(
		"conversation_id = ? AND seq > ? AND seq >= ?",
		in.ConversationId,
		in.AfterSeq,
		member.JoinedSeq,
	).Order("seq").Limit(pageLimit(in.Limit)).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := &pb.Messages{NextSeq: in.AfterSeq}
	profiles := map[uint64]*pb.Profile{}
	for _, row := range rows {
		sender := profiles[row.SenderID]
		if sender == nil {
			sender, err = u.repo.Users.GetProfile(ctx, &pb.UserID{UserId: row.SenderID})
			if err != nil {
				return nil, err
			}
			profiles[row.SenderID] = sender
		}
		out.Messages = append(out.Messages, message(row, sender))
		out.NextSeq = row.Seq
	}
	return out, nil
}
func (u *ConversationUseCase) RecallMessage(ctx context.Context, in *pb.MessageID) (*pb.Message, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	var row storage.Message
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conversation storage.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conversation, in.ConversationId).Error; err != nil {
			return err
		}
		if _, err := membership(tx, conversation.ID, id, true); err != nil {
			return err
		}
		if err := tx.Where("id = ? AND conversation_id = ?", in.MessageId, conversation.ID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return missing()
			}
			return err
		}
		if row.SenderID != id {
			return forbidden()
		}
		if row.Recalled {
			return nil
		}
		if err := tx.Model(&row).Updates(map[string]any{
			"recalled": true,
			"content":  "",
		}).Error; err != nil {
			return err
		}
		row.Recalled = true
		row.Content = ""
		users, err := activeUsers(tx, conversation.ID)
		if err != nil {
			return err
		}
		return delivery.Emit(tx, "conversation_events", users, &pb.Event{
			Kind:           pb.EventKind_MESSAGE_RECALLED,
			ConversationId: conversation.ID,
			ActorId:        id,
			MessageId:      row.ID,
			MessageSeq:     row.Seq,
		})
	})
	if err != nil {
		return nil, err
	}
	sender, err := u.repo.Users.GetProfile(ctx, &pb.UserID{UserId: row.SenderID})
	if err != nil {
		return nil, err
	}
	return message(row, sender), nil
}
func (u *ConversationUseCase) UpdateReadProgress(ctx context.Context, in *pb.ReadProgress) (*pb.ReadProgress, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conversation storage.Conversation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conversation, in.ConversationId).Error; err != nil {
			return err
		}
		member, err := membership(tx, conversation.ID, id, true)
		if err != nil {
			return err
		}
		if in.Seq > conversation.LastMessageSeq {
			return bad("已读序号超出消息范围")
		}
		return tx.Model(member).Update("read_seq", gorm.Expr("GREATEST(read_seq, ?)", in.Seq)).Error
	})
	if err != nil {
		return nil, err
	}
	return u.GetReadProgress(ctx, &pb.ConversationID{ConversationId: in.ConversationId})
}
func (u *ConversationUseCase) GetReadProgress(ctx context.Context, in *pb.ConversationID) (*pb.ReadProgress, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	member, err := membership(u.repo.DB.WithContext(ctx), in.ConversationId, id, true)
	if err != nil {
		return nil, err
	}
	return &pb.ReadProgress{
		ConversationId: in.ConversationId,
		Seq:            member.ReadSeq,
	}, nil
}
