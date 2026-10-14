// Package storage 定义基础 IM 的持久化记录；所有时间均为 Unix 毫秒。
package storage

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Account 是账户和密码摘要，密码摘要不进入公开资料。
type Account struct {
	ID           uint64
	Account      string
	PasswordHash string
	Nickname     string
	AvatarURL    string
	Bio          string
	Disabled     bool
	AckCursor    uint64
	CreatedAt    int64
	UpdatedAt    int64
}

func (Account) TableName() string { return "accounts" }

// FriendRequest 保存申请双方和明确的处理状态，不混入展示用状态码。
type FriendRequest struct {
	ID             uint64
	SenderID       uint64
	RecipientID    uint64
	Note           string
	State          int32
	ConversationID uint64
	CreatedAt      int64
	UpdatedAt      int64
}

func (FriendRequest) TableName() string { return "friend_requests" }

// Friendship 保存操作人一侧的好友快照及私聊会话 ID。
type Friendship struct {
	OwnerID        uint64 `gorm:"primaryKey;autoIncrement:false"`
	TargetID       uint64 `gorm:"primaryKey;autoIncrement:false"`
	Active         bool
	Remark         string
	ConversationID uint64
	UpdatedAt      int64
}

func (Friendship) TableName() string { return "friendships" }

// Block 保存单向拉黑，任一方拉黑都会阻止私聊发送。
type Block struct {
	OwnerID   uint64 `gorm:"primaryKey;autoIncrement:false"`
	TargetID  uint64 `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt int64
}

func (Block) TableName() string { return "blocks" }

// Conversation 保存会话类型和消息序号；私聊唯一键为空的记录属于群会话。
type Conversation struct {
	ID             uint64
	Type           int32
	DirectKey      *string
	LastMessageSeq uint64
	Closed         bool
	CreatedAt      int64
	UpdatedAt      int64
}

func (Conversation) TableName() string { return "conversations" }

// ConversationMember 保存成员状态和单调前进的已读序号；离开后记录继续存在以供同步。
type ConversationMember struct {
	ConversationID uint64 `gorm:"primaryKey;autoIncrement:false"`
	UserID         uint64 `gorm:"primaryKey;autoIncrement:false"`
	State          int32
	ReadSeq        uint64
	JoinedSeq      uint64
	JoinedAt       int64
	LeftAt         int64
}

func (ConversationMember) TableName() string { return "conversation_members" }

// Group 只保存群资料，通过 ConversationID 关联消息归属的会话。
type Group struct {
	ID              uint64
	ConversationID  uint64
	Name            string
	OwnerID         uint64
	RequireApproval bool
	Closed          bool
	CreatedAt       int64
	UpdatedAt       int64
}

func (Group) TableName() string { return "groups" }

// GroupMember 保存群角色与加入、离开时间，离开时间零表示当前成员。
type GroupMember struct {
	GroupID  uint64 `gorm:"primaryKey;autoIncrement:false"`
	UserID   uint64 `gorm:"primaryKey;autoIncrement:false"`
	Role     int32
	JoinedAt int64
	LeftAt   int64
}

func (GroupMember) TableName() string { return "group_members" }

// JoinRequest 保存每个群与申请人的最新入群申请状态。
type JoinRequest struct {
	ID        uint64
	GroupID   uint64
	UserID    uint64
	Note      string
	State     int32
	CreatedAt int64
	UpdatedAt int64
}

func (JoinRequest) TableName() string { return "join_requests" }

// Message 保存用户文字消息，撤回后保留序号并清空可见内容。
type Message struct {
	ID             uint64
	ConversationID uint64
	Seq            uint64
	SenderID       uint64
	Content        string
	SentAt         int64
	Recalled       bool
}

func (Message) TableName() string { return "messages" }

// SendKey 与消息在同一事务写入；Response 保存首个成功响应，供重试准确重放。
type SendKey struct {
	SenderID    uint64 `gorm:"primaryKey;autoIncrement:false"`
	Key         string `gorm:"primaryKey;column:send_key"`
	RequestHash string
	MessageID   uint64
	Response    string
	CreatedAt   int64
}

func (SendKey) TableName() string { return "send_keys" }

// Event 保存某个用户可补拉的事件，ID 是该事件流的游标。
type Event struct {
	ID        uint64
	UserID    uint64
	Data      string
	CreatedAt int64
}

// Outbox 在消息或事件事务中写入，发布失败保留记录并按 next_attempt_at 重试。
type Outbox struct {
	ID            uint64
	EventKey      string
	Payload       string
	Published     bool
	Attempts      uint32
	NextAttemptAt int64
	CreatedAt     int64
}

func (Outbox) TableName() string { return "outbox" }

// Open 建立有限连接池；SQL 日志不记录密码或消息内容。
func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(20)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(30 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return db, nil
}
