// Package delivery 提供消息和同步事件的事务 Outbox 与在线广播。
package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/pkg/storage"

	"github.com/go-kratos/kratos/v2/log"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const exchange = "imcore.v2.delivery"

// Envelope 是队列内部封装，data 是公开 v2 消息或事件 JSON。
type Envelope struct {
	Version int             `json:"version"`
	Type    string          `json:"type"`
	Users   []uint64        `json:"users"`
	Data    json.RawMessage `json:"data"`
}

// Enqueue 必须使用业务事务的 tx，保证消息持久化和待投递记录同时提交。
func Enqueue(tx *gorm.DB, key, kind string, users []uint64, data []byte) error {
	payload, err := json.Marshal(Envelope{
		Version: 2,
		Type:    kind,
		Users:   users,
		Data:    data,
	})
	if err != nil {
		return err
	}
	return tx.Create(&storage.Outbox{
		EventKey:  key,
		Payload:   string(payload),
		CreatedAt: time.Now().UnixMilli(),
	}).Error
}

// Emit 对每个接收人持久化自己的事件游标，同时写入在线投递 Outbox。
func Emit(tx *gorm.DB, table string, users []uint64, event *pb.Event) error {

	users = append([]uint64(nil), users...)
	sort.Slice(users, func(i, j int) bool { return users[i] < users[j] })
	unique := users[:0]
	for _, user := range users {
		if len(unique) == 0 || unique[len(unique)-1] != user {
			unique = append(unique, user)
		}
	}
	users = unique
	if table == "conversation_events" {
		// 全局自增 ID 分配顺序不等于提交顺序。先按用户 ID 加锁，再分配游标，
		// 防止另一会话先提交较大游标，使补拉跳过仍未提交的较小游标。
		for _, user := range users {
			if err := tx.Exec(
				"INSERT INTO event_cursors (user_id,acknowledged_cursor) VALUES (?,0) ON DUPLICATE KEY UPDATE user_id=VALUES(user_id)",
				user,
			).Error; err != nil {
				return err
			}
		}
	}
	event.OccurredAt = time.Now().UnixMilli()
	for _, user := range users {
		row := storage.Event{
			UserID:    user,
			Data:      "{}",
			CreatedAt: event.OccurredAt,
		}
		if err := tx.Table(table).Create(&row).Error; err != nil {
			return err
		}
		event.Cursor = row.ID
		data, err := protojson.Marshal(event)
		if err != nil {
			return err
		}
		if err = tx.Table(table).Where("id = ?", row.ID).Update("data", string(data)).Error; err != nil {
			return err
		}
		if err = Enqueue(tx, table+":"+strconv.FormatUint(row.ID, 10), "event", []uint64{user}, data); err != nil {
			return err
		}
	}
	return nil
}

// PublishLoop 使用确认发布。多个工作进程以 SKIP LOCKED 领取不同记录，至少一次投递。
func PublishLoop(ctx context.Context, db *gorm.DB, url string, logger log.Logger) {
	helper := log.NewHelper(logger)
	for ctx.Err() == nil {
		if err := publish(ctx, db, url); err != nil && ctx.Err() == nil {
			helper.Error("outbox publish failed; records retained for retry")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
func publish(ctx context.Context, db *gorm.DB, url string) error {
	conn, err := amqp.DialConfig(url, amqp.Config{Dial: amqp.DefaultDial(5 * time.Second)})
	if err != nil {
		return fmt.Errorf("broker unavailable")
	}
	defer conn.Close()
	channel, err := conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()
	if err = channel.ExchangeDeclare(exchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	if err = channel.Confirm(false); err != nil {
		return err
	}
	for ctx.Err() == nil {
		empty := false
		failed := false
		err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var rows []storage.Outbox
			if err := tx.Clauses(clause.Locking{
				Strength: "UPDATE",
				Options:  "SKIP LOCKED",
			}).Where("published = false AND next_attempt_at <= ?", time.Now().UnixMilli()).Order("id").Limit(20).Find(&rows).Error; err != nil {
				return err
			}
			if len(rows) == 0 {
				empty = true
				return nil
			}
			for _, row := range rows {
				confirmation, err := channel.PublishWithDeferredConfirmWithContext(ctx, exchange, "", false, false, amqp.Publishing{
					ContentType:  "application/json",
					DeliveryMode: amqp.Persistent,
					MessageId:    row.EventKey,
					Body:         []byte(row.Payload),
				})
				if err == nil {
					waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
					var confirmed bool
					confirmed, err = confirmation.WaitContext(waitCtx)
					cancel()
					if err == nil && !confirmed {
						err = fmt.Errorf("broker rejected publish")
					}
				}
				if err != nil {
					// 发布错误保留重试时间，提交后重建连接，避免在已关闭 channel 上持续重试。
					failed = true
					if e := tx.Model(&storage.Outbox{}).Where("id = ?", row.ID).Updates(map[string]any{
						"attempts":        gorm.Expr("attempts + 1"),
						"next_attempt_at": time.Now().Add(2 * time.Second).UnixMilli(),
					}).Error; e != nil {
						return e
					}
					return nil
				}
				if err = tx.Model(&storage.Outbox{}).Where("id = ?", row.ID).Update("published", true).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		if failed {
			return fmt.Errorf("broker publish failed")
		}
		if empty {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	return nil
}

// Subscribe 每个 Connect 实例使用独立队列，广播至该实例持有的连接。
func Subscribe(
	ctx context.Context,
	url string,
	receive func(Envelope),
	ready func(bool),
	logger log.Logger,
) {
	helper := log.NewHelper(logger)
	for ctx.Err() == nil {
		if err := subscribe(ctx, url, receive, ready); err != nil && ctx.Err() == nil {
			helper.Error("delivery subscriber reconnecting")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}
func subscribe(ctx context.Context, url string, receive func(Envelope), ready func(bool)) error {
	conn, err := amqp.DialConfig(url, amqp.Config{Dial: amqp.DefaultDial(5 * time.Second)})
	if err != nil {
		return fmt.Errorf("broker unavailable")
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	if err = ch.ExchangeDeclare(exchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}

	if err = ch.ExchangeDeclare("imcore.v2.quarantine", "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	quarantine, err := ch.QueueDeclare("imcore.v2.quarantine", true, false, false, false, nil)
	if err != nil {
		return err
	}
	if err = ch.QueueBind(quarantine.Name, "", "imcore.v2.quarantine", false, nil); err != nil {
		return err
	}
	q, err := ch.QueueDeclare(
		"",
		false,
		true,
		true,
		false,
		amqp.Table{"x-dead-letter-exchange": "imcore.v2.quarantine"},
	)
	if err != nil {
		return err
	}
	if err = ch.QueueBind(q.Name, "", exchange, false, nil); err != nil {
		return err
	}
	if err = ch.Qos(100, 0, false); err != nil {
		return err
	}
	messages, err := ch.Consume(q.Name, "", false, true, false, false, nil)
	if err != nil {
		return err
	}
	ready(true)
	defer ready(false)
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-messages:
			if !ok {
				return fmt.Errorf("broker stream closed")
			}
			var envelope Envelope
			if err := json.Unmarshal(msg.Body, &envelope); err != nil || envelope.Version != 2 || (envelope.Type != "message" && envelope.Type != "event") || !json.Valid(envelope.Data) {
				// 无法解析的投递进入隔离队列；离线状态仍以数据库为准。
				if err := msg.Reject(false); err != nil {
					return err
				}
				continue
			}
			receive(envelope)
			if err := msg.Ack(false); err != nil {
				return err
			}
		}
	}
}
