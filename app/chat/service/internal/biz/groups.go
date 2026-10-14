package biz

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/pkg/delivery"
	"github.com/hellokitty980808-ctrl/im-core/pkg/platform"
	"github.com/hellokitty980808-ctrl/im-core/pkg/storage"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func validName(name string) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(name))
	return n >= 1 && n <= 32
}
func (u *ConversationUseCase) checkFriends(ctx context.Context, actor uint64, targets []uint64) ([]uint64, error) {
	if len(targets) > 100 {
		return nil, bad("每次最多邀请 100 人")
	}
	ids := make([]uint64, 0, len(targets))
	seen := map[uint64]bool{actor: true}
	for _, id := range targets {
		if id == 0 {
			return nil, bad("用户 ID 无效")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		allowed, err := u.repo.Users.CheckRelationship(ctx, &pb.RelationshipQuery{
			FirstUserId:  actor,
			SecondUserId: id,
		})
		if err != nil {
			return nil, err
		}
		if !allowed.Allowed {
			return nil, forbidden()
		}
		ids = append(ids, id)
	}
	return ids, nil
}
func lockGroup(tx *gorm.DB, id uint64) (*storage.Group, *storage.Conversation, error) {
	if id == 0 {
		return nil, nil, bad("群 ID 不能为空")
	}
	var group storage.Group
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&group, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, missing()
	}
	if err != nil {
		return nil, nil, err
	}
	if group.Closed {
		return nil, nil, kerrors.Conflict("GROUP_CLOSED", "群已解散")
	}
	var conversation storage.Conversation
	if err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&conversation, group.ConversationID).Error; err != nil {
		return nil, nil, err
	}
	return &group, &conversation, nil
}
func activeUsers(tx *gorm.DB, conversation uint64) ([]uint64, error) {
	var ids []uint64
	err := tx.Model(&storage.ConversationMember{}).Where("conversation_id = ? AND state = ?", conversation, pb.ConversationState_ACTIVE).Order("user_id").Pluck("user_id", &ids).Error
	return ids, err
}
func addMember(
	tx *gorm.DB,
	group storage.Group,
	conversation storage.Conversation,
	user uint64,
	role pb.MemberRole,
) error {
	var count int64
	if err := tx.Model(&storage.ConversationMember{}).Where(
		"conversation_id = ? AND user_id = ? AND state = ?",
		conversation.ID,
		user,
		pb.ConversationState_ACTIVE,
	).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	member := storage.ConversationMember{
		ConversationID: conversation.ID,
		UserID:         user,
		State:          int32(pb.ConversationState_ACTIVE),
		JoinedSeq:      conversation.LastMessageSeq + 1,
		ReadSeq:        conversation.LastMessageSeq,
		JoinedAt:       now,
	}
	if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&member).Error; err != nil {
		return err
	}
	groupMember := storage.GroupMember{
		GroupID:  group.ID,
		UserID:   user,
		Role:     int32(role),
		JoinedAt: now,
	}
	return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&groupMember).Error
}
func emitMember(tx *gorm.DB, group storage.Group, actor, target uint64, state pb.ConversationState) error {
	users, err := activeUsers(tx, group.ConversationID)
	if err != nil {
		return err
	}
	found := false
	for _, id := range users {
		if id == target {
			found = true
		}
	}
	if !found {
		users = append(users, target)
	}
	return delivery.Emit(tx, "conversation_events", users, &pb.Event{
		Kind:           pb.EventKind_GROUP_MEMBER_CHANGED,
		ConversationId: group.ConversationID,
		GroupId:        group.ID,
		ActorId:        actor,
		UserId:         target,
		State:          state,
	})
}
func (u *ConversationUseCase) groupInfo(ctx context.Context, row storage.Group) (*pb.Group, error) {
	var count int64
	if err := u.repo.DB.WithContext(ctx).Model(&storage.GroupMember{}).Where("group_id = ? AND left_at = 0", row.ID).Count(&count).Error; err != nil {
		return nil, err
	}
	return &pb.Group{
		GroupId:         row.ID,
		ConversationId:  row.ConversationID,
		Name:            row.Name,
		OwnerId:         row.OwnerID,
		MemberCount:     uint32(count),
		RequireApproval: row.RequireApproval,
		Closed:          row.Closed,
	}, nil
}
func (u *ConversationUseCase) GetGroup(ctx context.Context, in *pb.GroupID) (*pb.Group, error) {
	if _, err := platform.Identity(ctx); err != nil {
		return nil, err
	}
	if in.GroupId == 0 {
		return nil, bad("群 ID 不能为空")
	}
	var group storage.Group
	err := u.repo.DB.WithContext(ctx).First(&group, in.GroupId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, missing()
	}
	if err != nil {
		return nil, err
	}
	return u.groupInfo(ctx, group)
}
func (u *ConversationUseCase) CreateGroup(ctx context.Context, in *pb.GroupDraft) (*pb.Group, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	if !validName(in.Name) {
		return nil, bad("群名长度为 1–32 个字符")
	}
	targets, err := u.checkFriends(ctx, id, in.MemberUserIds)
	if err != nil {
		return nil, err
	}
	targets = append([]uint64{id}, targets...)
	var group storage.Group
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		conversation := storage.Conversation{
			Type:      int32(pb.ConversationType_GROUP),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Create(&conversation).Error; err != nil {
			return err
		}
		group = storage.Group{
			ConversationID:  conversation.ID,
			Name:            strings.TrimSpace(in.Name),
			OwnerID:         id,
			RequireApproval: in.RequireApproval,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		for _, target := range targets {
			role := pb.MemberRole_MEMBER
			if target == id {
				role = pb.MemberRole_OWNER
			}
			if err := addMember(tx, group, conversation, target, role); err != nil {
				return err
			}
		}
		for _, target := range targets {
			if err := emitMember(tx, group, id, target, pb.ConversationState_ACTIVE); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return u.groupInfo(ctx, group)
}
func (u *ConversationUseCase) UpdateGroup(ctx context.Context, in *pb.GroupUpdate) (*pb.Group, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	if !validName(in.Name) {
		return nil, bad("群名长度为 1–32 个字符")
	}
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, _, err := lockGroup(tx, in.GroupId)
		if err != nil {
			return err
		}
		if group.OwnerID != id {
			return forbidden()
		}
		return tx.Model(group).Updates(map[string]any{
			"name":       strings.TrimSpace(in.Name),
			"updated_at": time.Now().UnixMilli(),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return u.GetGroup(ctx, &pb.GroupID{GroupId: in.GroupId})
}
func (u *ConversationUseCase) ListMembers(ctx context.Context, in *pb.GroupID) (*pb.GroupMembers, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	var group storage.Group
	if err = u.repo.DB.WithContext(ctx).First(&group, in.GroupId).Error; err != nil {
		return nil, err
	}
	if _, err = membership(u.repo.DB.WithContext(ctx), group.ConversationID, id, true); err != nil {
		return nil, err
	}
	var rows []storage.GroupMember
	if err = u.repo.DB.WithContext(ctx).Where("group_id = ? AND left_at = 0", in.GroupId).Order("user_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := &pb.GroupMembers{}
	for _, row := range rows {
		info, err := u.repo.Users.GetProfile(ctx, &pb.UserID{UserId: row.UserID})
		if err != nil {
			return nil, err
		}
		out.Members = append(out.Members, &pb.GroupMember{
			User:     info,
			Role:     pb.MemberRole(row.Role),
			JoinedAt: row.JoinedAt,
		})
	}
	return out, nil
}
func (u *ConversationUseCase) InviteMembers(ctx context.Context, in *pb.MemberInvite) (*pb.Group, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	targets, err := u.checkFriends(ctx, id, in.UserIds)
	if err != nil {
		return nil, err
	}
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, conversation, err := lockGroup(tx, in.GroupId)
		if err != nil {
			return err
		}
		if _, err = membership(tx, group.ConversationID, id, true); err != nil {
			return err
		}
		for _, target := range targets {
			if err = addMember(tx, *group, *conversation, target, pb.MemberRole_MEMBER); err != nil {
				return err
			}
			if err = emitMember(tx, *group, id, target, pb.ConversationState_ACTIVE); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return u.GetGroup(ctx, &pb.GroupID{GroupId: in.GroupId})
}
func removeMember(tx *gorm.DB, group storage.Group, actor, target uint64) error {
	member, err := membership(tx, group.ConversationID, target, false)
	if err != nil {
		return err
	}
	if member.State != int32(pb.ConversationState_ACTIVE) {
		return nil
	}
	now := time.Now().UnixMilli()
	if err = tx.Model(member).Updates(map[string]any{
		"state":   pb.ConversationState_LEFT,
		"left_at": now,
	}).Error; err != nil {
		return err
	}
	if err = tx.Model(&storage.GroupMember{}).Where("group_id = ? AND user_id = ?", group.ID, target).Update("left_at", now).Error; err != nil {
		return err
	}
	return emitMember(tx, group, actor, target, pb.ConversationState_LEFT)
}
func (u *ConversationUseCase) RemoveMember(ctx context.Context, in *pb.MemberID) (*emptypb.Empty, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, _, err := lockGroup(tx, in.GroupId)
		if err != nil {
			return err
		}
		if group.OwnerID != id || in.UserId == id {
			return forbidden()
		}
		return removeMember(tx, *group, id, in.UserId)
	})
	return &emptypb.Empty{}, err
}
func (u *ConversationUseCase) LeaveGroup(ctx context.Context, in *pb.GroupID) (*emptypb.Empty, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, _, err := lockGroup(tx, in.GroupId)
		if err != nil {
			return err
		}
		if group.OwnerID == id {
			return kerrors.Conflict("OWNER_MUST_CLOSE", "群主需要解散群")
		}
		return removeMember(tx, *group, id, id)
	})
	return &emptypb.Empty{}, err
}
func (u *ConversationUseCase) CloseGroup(ctx context.Context, in *pb.GroupID) (*emptypb.Empty, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing storage.Group
		if err := tx.First(&existing, in.GroupId).Error; err != nil {
			return err
		}
		if existing.OwnerID != id {
			return forbidden()
		}
		if existing.Closed {
			return nil
		}
		group, conversation, err := lockGroup(tx, in.GroupId)
		if err != nil {
			return err
		}
		users, err := activeUsers(tx, conversation.ID)
		if err != nil {
			return err
		}
		now := time.Now().UnixMilli()
		if err = tx.Model(group).Updates(map[string]any{
			"closed":     true,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		if err = tx.Model(conversation).Updates(map[string]any{
			"closed":     true,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		if err = tx.Model(&storage.ConversationMember{}).Where("conversation_id = ? AND state = ?", conversation.ID, pb.ConversationState_ACTIVE).Updates(map[string]any{
			"state":   pb.ConversationState_CLOSED,
			"left_at": now,
		}).Error; err != nil {
			return err
		}
		if err = tx.Model(&storage.GroupMember{}).Where("group_id = ? AND left_at = 0", group.ID).Update("left_at", now).Error; err != nil {
			return err
		}
		return delivery.Emit(tx, "conversation_events", users, &pb.Event{
			Kind:           pb.EventKind_GROUP_CLOSED,
			GroupId:        group.ID,
			ConversationId: conversation.ID,
			ActorId:        id,
			State:          pb.ConversationState_CLOSED,
		})
	})
	return &emptypb.Empty{}, err
}
func joinRequest(row storage.JoinRequest) *pb.JoinRequest {
	return &pb.JoinRequest{
		RequestId: row.ID,
		GroupId:   row.GroupID,
		UserId:    row.UserID,
		Note:      row.Note,
		State:     pb.RequestState(row.State),
	}
}
func (u *ConversationUseCase) RequestJoin(ctx context.Context, in *pb.JoinDraft) (*pb.JoinRequest, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(in.Note) > 200 {
		return nil, bad("申请备注过长")
	}
	var row storage.JoinRequest
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, conversation, err := lockGroup(tx, in.GroupId)
		if err != nil {
			return err
		}
		row = storage.JoinRequest{
			GroupID:   group.ID,
			UserID:    id,
			Note:      in.Note,
			State:     int32(pb.RequestState_PENDING),
			CreatedAt: time.Now().UnixMilli(),
			UpdatedAt: time.Now().UnixMilli(),
		}
		var existing storage.JoinRequest
		if err = tx.Where("group_id = ? AND user_id = ?", group.ID, id).First(&existing).Error; err == nil {
			row.ID = existing.ID
			row.CreatedAt = existing.CreatedAt
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var count int64
		if err = tx.Model(&storage.GroupMember{}).Where("group_id = ? AND user_id = ? AND left_at = 0", group.ID, id).Count(&count).Error; err != nil {
			return err
		}
		if !group.RequireApproval || count > 0 {
			row.State = int32(pb.RequestState_ACCEPTED)
			if count == 0 {
				if err = addMember(tx, *group, *conversation, id, pb.MemberRole_MEMBER); err != nil {
					return err
				}
				if err = emitMember(tx, *group, id, id, pb.ConversationState_ACTIVE); err != nil {
					return err
				}
			}
		}
		return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&row).Error
	})
	if err != nil {
		return nil, err
	}
	return joinRequest(row), nil
}
func (u *ConversationUseCase) ListJoinRequests(ctx context.Context, in *pb.GroupID) (*pb.JoinRequests, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	group, err := u.GetGroup(ctx, in)
	if err != nil {
		return nil, err
	}
	if group.OwnerId != id {
		return nil, forbidden()
	}
	var rows []storage.JoinRequest
	if err = u.repo.DB.WithContext(ctx).Where("group_id = ?", in.GroupId).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := &pb.JoinRequests{}
	for _, row := range rows {
		out.Requests = append(out.Requests, joinRequest(row))
	}
	return out, nil
}
func (u *ConversationUseCase) decideJoin(ctx context.Context, in *pb.JoinRequestID, state pb.RequestState) (*pb.JoinRequest, error) {
	id, err := platform.Identity(ctx)
	if err != nil {
		return nil, err
	}
	var row storage.JoinRequest
	err = u.repo.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group, conversation, err := lockGroup(tx, in.GroupId)
		if err != nil {
			return err
		}
		if group.OwnerID != id {
			return forbidden()
		}
		if err = tx.Where("id = ? AND group_id = ?", in.RequestId, group.ID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return missing()
			}
			return err
		}
		if row.State == int32(state) {
			return nil
		}
		if row.State != int32(pb.RequestState_PENDING) {
			return kerrors.Conflict("REQUEST_DECIDED", "申请已处理")
		}
		row.State = int32(state)
		row.UpdatedAt = time.Now().UnixMilli()
		if err = tx.Save(&row).Error; err != nil {
			return err
		}
		if state == pb.RequestState_ACCEPTED {
			if err = addMember(tx, *group, *conversation, row.UserID, pb.MemberRole_MEMBER); err != nil {
				return err
			}
			return emitMember(tx, *group, id, row.UserID, pb.ConversationState_ACTIVE)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return joinRequest(row), nil
}
func (u *ConversationUseCase) AcceptJoinRequest(ctx context.Context, in *pb.JoinRequestID) (*pb.JoinRequest, error) {
	return u.decideJoin(ctx, in, pb.RequestState_ACCEPTED)
}
func (u *ConversationUseCase) RejectJoinRequest(ctx context.Context, in *pb.JoinRequestID) (*pb.JoinRequest, error) {
	return u.decideJoin(ctx, in, pb.RequestState_REJECTED)
}
