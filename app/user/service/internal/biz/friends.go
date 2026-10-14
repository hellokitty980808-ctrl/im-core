package biz

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/pkg/delivery"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"
	"github.com/hellokitty980808-ctrl/im-core/pkg/storage"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func pageLimit(n uint32) int {
	if n == 0 {
		return 30
	}
	if n > 100 {
		return 100
	}
	return int(n)
}
func friendRequest(row storage.FriendRequest) *pb.FriendRequest {
	return &pb.FriendRequest{
		RequestId:      row.ID,
		SenderId:       row.SenderID,
		RecipientId:    row.RecipientID,
		Note:           row.Note,
		State:          pb.RequestState(row.State),
		ConversationId: row.ConversationID,
		CreatedAt:      row.CreatedAt,
	}
}

// lockPair 按 ID 顺序锁定两个账户，所有好友与拉黑变更共用此锁顺序。
func lockPair(tx *gorm.DB, a, b uint64) error {
	if a == 0 || b == 0 || a == b {
		return bad("目标用户无效")
	}
	ids := []uint64{a, b}
	if a > b {
		ids[0], ids[1] = b, a
	}
	for _, id := range ids {
		var row storage.Account
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return missing()
		}
		if err != nil {
			return err
		}
		if row.Disabled {
			return forbidden()
		}
	}
	return nil
}
func blocked(tx *gorm.DB, a, b uint64) (bool, error) {
	var count int64
	err := tx.Model(&storage.Block{}).Where("(owner_id = ? AND target_id = ?) OR (owner_id = ? AND target_id = ?)", a, b, b, a).Count(&count).Error
	return count > 0, err
}
func allowed(tx *gorm.DB, a, b uint64) (bool, error) {
	isBlocked, err := blocked(tx, a, b)
	if err != nil || isBlocked {
		return false, err
	}
	var count int64
	err = tx.Model(&storage.Friendship{}).Where(
		"active = true AND ((owner_id = ? AND target_id = ?) OR (owner_id = ? AND target_id = ?))",
		a,
		b,
		b,
		a,
	).Count(&count).Error
	return count == 2, err
}
func (u *AccountUseCase) CheckRelationship(ctx context.Context, in *pb.RelationshipQuery) (*pb.RelationshipCheck, error) {
	ok, err := allowed(u.repo.DB.WithContext(ctx), in.FirstUserId, in.SecondUserId)
	return &pb.RelationshipCheck{Allowed: ok}, err
}
func relation(tx *gorm.DB, a, b uint64) (pb.Relationship, error) {
	var count int64
	if err := tx.Model(&storage.Block{}).Where("owner_id = ? AND target_id = ?", a, b).Count(&count).Error; err != nil {
		return pb.Relationship_NONE, err
	}
	if count > 0 {
		return pb.Relationship_BLOCKED, nil
	}
	if err := tx.Model(&storage.Friendship{}).Where("owner_id = ? AND target_id = ? AND active = true", a, b).Count(&count).Error; err != nil {
		return pb.Relationship_NONE, err
	}
	if count > 0 {
		return pb.Relationship_FRIEND, nil
	}
	return pb.Relationship_NONE, nil
}
func emitPair(tx *gorm.DB, actor, target uint64) error {
	for _, pair := range [][2]uint64{{actor, target}, {target, actor}} {
		state, err := relation(tx, pair[0], pair[1])
		if err != nil {
			return err
		}
		if err = delivery.Emit(tx, "friend_events", []uint64{pair[0]}, &pb.Event{
			Kind:         pb.EventKind_FRIEND_CHANGED,
			ActorId:      actor,
			UserId:       pair[1],
			Relationship: state,
		}); err != nil {
			return err
		}
	}
	return nil
}
func (u *AccountUseCase) CreateFriendRequest(ctx context.Context, in *pb.FriendRequestDraft) (*pb.FriendRequest, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(in.Note) > 200 {
		return nil, bad("申请备注过长")
	}
	var row storage.FriendRequest
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPair(tx, id, in.UserId); err != nil {
			return err
		}
		deny, err := blocked(tx, id, in.UserId)
		if err != nil {
			return err
		}
		if deny {
			return forbidden()
		}
		ok, err := allowed(tx, id, in.UserId)
		if err != nil {
			return err
		}
		if ok {
			return kerrors.Conflict("ALREADY_FRIENDS", "双方已是好友")
		}
		err = tx.Where(
			"state = ? AND ((sender_id = ? AND recipient_id = ?) OR (sender_id = ? AND recipient_id = ?))",
			pb.RequestState_PENDING,
			id,
			in.UserId,
			in.UserId,
			id,
		).First(&row).Error
		if err == nil {
			if row.SenderID != id {
				return kerrors.Conflict("REQUEST_EXISTS", "请处理对方已发送的申请")
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		now := time.Now().UnixMilli()
		row = storage.FriendRequest{
			SenderID:    id,
			RecipientID: in.UserId,
			Note:        in.Note,
			State:       int32(pb.RequestState_PENDING),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err = tx.Create(&row).Error; err != nil {
			return err
		}
		return emitPair(tx, id, in.UserId)
	})
	if err != nil {
		return nil, err
	}
	return friendRequest(row), nil
}
func (u *AccountUseCase) ListFriendRequests(ctx context.Context, in *pb.Page) (*pb.FriendRequests, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	var rows []storage.FriendRequest
	if err = u.repo.DB.WithContext(ctx).Where("id > ? AND (sender_id = ? OR recipient_id = ?)", in.Cursor, id, id).Order("id").Limit(pageLimit(in.Limit)).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := &pb.FriendRequests{NextCursor: in.Cursor}
	for _, row := range rows {
		out.Requests = append(out.Requests, friendRequest(row))
		out.NextCursor = row.ID
	}
	return out, nil
}
func (u *AccountUseCase) decideRequest(ctx context.Context, in *pb.FriendRequestID, state pb.RequestState) (*pb.FriendRequest, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	var initial storage.FriendRequest
	if err = u.repo.DB.WithContext(ctx).First(&initial, in.RequestId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, missing()
		}
		return nil, err
	}
	var row storage.FriendRequest
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPair(tx, initial.SenderID, initial.RecipientID); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, in.RequestId).Error; err != nil {
			return err
		}
		if state == pb.RequestState_CANCELLED {
			if id != row.SenderID {
				return forbidden()
			}
		} else if id != row.RecipientID {
			return forbidden()
		}
		if row.State == int32(state) {
			return nil
		}
		if row.State != int32(pb.RequestState_PENDING) {
			return kerrors.Conflict("REQUEST_DECIDED", "申请已处理")
		}
		if state == pb.RequestState_ACCEPTED {
			deny, err := blocked(tx, row.SenderID, row.RecipientID)
			if err != nil {
				return err
			}
			if deny {
				return forbidden()
			}
			// 私聊建立使用双方唯一键；跨服务响应丢失后的重试不会生成第二个会话。
			direct, err := u.repo.Chat.EnsureDirect(ctx, &pb.DirectPair{
				FirstUserId:  row.SenderID,
				SecondUserId: row.RecipientID,
			})
			if err != nil {
				return err
			}
			row.ConversationID = direct.ConversationId
			for _, pair := range [][2]uint64{{row.SenderID, row.RecipientID}, {row.RecipientID, row.SenderID}} {
				friendship := storage.Friendship{
					OwnerID:        pair[0],
					TargetID:       pair[1],
					Active:         true,
					ConversationID: row.ConversationID,
					UpdatedAt:      time.Now().UnixMilli(),
				}
				if err = tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&friendship).Error; err != nil {
					return err
				}
			}
		}
		row.State = int32(state)
		row.UpdatedAt = time.Now().UnixMilli()
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		return emitPair(tx, id, other(row, id))
	})
	if err != nil {
		return nil, err
	}
	return friendRequest(row), nil
}
func other(row storage.FriendRequest, id uint64) uint64 {
	if id == row.SenderID {
		return row.RecipientID
	}
	return row.SenderID
}
func (u *AccountUseCase) AcceptFriendRequest(ctx context.Context, in *pb.FriendRequestID) (*pb.FriendRequest, error) {
	return u.decideRequest(ctx, in, pb.RequestState_ACCEPTED)
}
func (u *AccountUseCase) RejectFriendRequest(ctx context.Context, in *pb.FriendRequestID) (*pb.FriendRequest, error) {
	return u.decideRequest(ctx, in, pb.RequestState_REJECTED)
}
func (u *AccountUseCase) CancelFriendRequest(ctx context.Context, in *pb.FriendRequestID) (*pb.FriendRequest, error) {
	return u.decideRequest(ctx, in, pb.RequestState_CANCELLED)
}
func (u *AccountUseCase) GetFriend(ctx context.Context, in *pb.UserID) (*pb.Friend, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	info, err := u.GetProfile(ctx, in)
	if err != nil {
		return nil, err
	}
	state, err := relation(u.repo.DB.WithContext(ctx), id, in.UserId)
	if err != nil {
		return nil, err
	}
	var row storage.Friendship
	err = u.repo.DB.WithContext(ctx).Where("owner_id = ? AND target_id = ?", id, in.UserId).First(&row).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &pb.Friend{
		User:           info,
		Remark:         row.Remark,
		Relationship:   state,
		ConversationId: row.ConversationID,
	}, nil
}
func (u *AccountUseCase) listFriends(ctx context.Context, in *pb.Page, blocks bool) (*pb.Friends, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	var ids []uint64
	db := u.repo.DB.WithContext(ctx)
	if blocks {
		err = db.Model(&storage.Block{}).Where("owner_id = ? AND target_id > ?", id, in.Cursor).Order("target_id").Limit(pageLimit(in.Limit)).Pluck("target_id", &ids).Error
	} else {
		err = db.Model(&storage.Friendship{}).Where(
			"owner_id = ? AND active = true AND target_id > ? AND target_id NOT IN (SELECT target_id FROM blocks WHERE owner_id = ?)",
			id,
			in.Cursor,
			id,
		).Order("target_id").Limit(pageLimit(in.Limit)).Pluck("target_id", &ids).Error
	}
	if err != nil {
		return nil, err
	}
	out := &pb.Friends{NextCursor: in.Cursor}
	for _, target := range ids {
		friend, err := u.GetFriend(ctx, &pb.UserID{UserId: target})
		if err != nil {
			return nil, err
		}
		out.Friends = append(out.Friends, friend)
		out.NextCursor = target
	}
	return out, nil
}
func (u *AccountUseCase) ListFriends(ctx context.Context, in *pb.Page) (*pb.Friends, error) {
	return u.listFriends(ctx, in, false)
}
func (u *AccountUseCase) ListBlocks(ctx context.Context, in *pb.Page) (*pb.Friends, error) {
	return u.listFriends(ctx, in, true)
}
func (u *AccountUseCase) UpdateFriend(ctx context.Context, in *pb.FriendUpdate) (*pb.Friend, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(in.Remark) > 32 {
		return nil, bad("备注过长")
	}
	result := u.repo.DB.WithContext(ctx).Model(&storage.Friendship{}).Where("owner_id = ? AND target_id = ? AND active = true", id, in.UserId).Updates(map[string]any{
		"remark":     in.Remark,
		"updated_at": time.Now().UnixMilli(),
	})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, missing()
	}
	return u.GetFriend(ctx, &pb.UserID{UserId: in.UserId})
}
func (u *AccountUseCase) changeRelation(ctx context.Context, target uint64, operation string) (*emptypb.Empty, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPair(tx, id, target); err != nil {
			return err
		}
		switch operation {
		case "delete":
			if err := tx.Model(&storage.Friendship{}).Where("owner_id = ? AND target_id = ?", id, target).Updates(map[string]any{
				"active":     false,
				"updated_at": time.Now().UnixMilli(),
			}).Error; err != nil {
				return err
			}
		case "block":
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&storage.Block{
				OwnerID:   id,
				TargetID:  target,
				CreatedAt: time.Now().UnixMilli(),
			}).Error; err != nil {
				return err
			}
		case "unblock":
			if err := tx.Where("owner_id = ? AND target_id = ?", id, target).Delete(&storage.Block{}).Error; err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid relationship operation")
		}
		return emitPair(tx, id, target)
	})
	return &emptypb.Empty{}, err
}
func (u *AccountUseCase) DeleteFriend(ctx context.Context, in *pb.UserID) (*emptypb.Empty, error) {
	return u.changeRelation(ctx, in.UserId, "delete")
}
func (u *AccountUseCase) BlockUser(ctx context.Context, in *pb.UserID) (*emptypb.Empty, error) {
	return u.changeRelation(ctx, in.UserId, "block")
}
func (u *AccountUseCase) UnblockUser(ctx context.Context, in *pb.UserID) (*emptypb.Empty, error) {
	return u.changeRelation(ctx, in.UserId, "unblock")
}
func (u *AccountUseCase) PullFriendEvents(ctx context.Context, in *pb.Page) (*pb.Events, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	var rows []storage.Event
	if err = u.repo.DB.WithContext(ctx).Table("friend_events").Where("user_id = ? AND id > ?", id, in.Cursor).Order("id").Limit(pageLimit(in.Limit)).Find(&rows).Error; err != nil {
		return nil, err
	}
	row, err := u.account(ctx, id)
	if err != nil {
		return nil, err
	}
	out := &pb.Events{
		NextCursor:         in.Cursor,
		AcknowledgedCursor: row.AckCursor,
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
func (u *AccountUseCase) AckFriendEvents(ctx context.Context, in *pb.EventAck) (*emptypb.Empty, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	var max uint64
	if err = u.repo.DB.WithContext(ctx).Table("friend_events").Where("user_id = ?", id).Select("COALESCE(MAX(id),0)").Scan(&max).Error; err != nil {
		return nil, err
	}
	if in.Cursor > max {
		return nil, bad("事件确认游标超出范围")
	}
	err = u.repo.DB.WithContext(ctx).Model(&storage.Account{}).Where("id = ?", id).Update("ack_cursor", gorm.Expr("GREATEST(ack_cursor, ?)", in.Cursor)).Error
	return &emptypb.Empty{}, err
}
