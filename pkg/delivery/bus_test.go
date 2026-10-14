package delivery

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/pkg/storage"

	driver "github.com/go-sql-driver/mysql"
)

// TestConversationEventCommitOrder 验证跨会话事件不会因事务反序提交而遗漏。
func TestConversationEventCommitOrder(t *testing.T) {
	if os.Getenv("IMCORE_DB_INTEGRATION") != "1" {
		t.Skip("仅在隔离 MySQL 中显式运行")
	}
	cfg := driver.NewConfig()
	cfg.User = os.Getenv("DB_USER")
	if cfg.User == "" {
		cfg.User = "root"
	}
	cfg.Passwd = os.Getenv("DB_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = "mysql:3306"
	prefix := os.Getenv("DB_NAME_PREFIX")
	if prefix == "" {
		prefix = "imcore"
	}
	cfg.DBName = prefix + "_chat"
	db, err := storage.Open(cfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	t.Cleanup(func() { _ = pool.Close() })
	user := uint64(time.Now().UnixNano())
	t.Cleanup(func() {
		var ids []uint64
		if err := db.Table("conversation_events").Where("user_id = ?", user).Pluck("id", &ids).Error; err != nil {
			t.Error(err)
		}
		for _, id := range ids {
			if err := db.Where("event_key = ?", "conversation_events:"+strconv.FormatUint(id, 10)).Delete(&storage.Outbox{}).Error; err != nil {
				t.Error(err)
			}
		}
		if err := db.Table("conversation_events").Where("user_id = ?", user).Delete(&storage.Event{}).Error; err != nil {
			t.Error(err)
		}
		if err := db.Exec("DELETE FROM event_cursors WHERE user_id = ?", user).Error; err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	first := db.WithContext(ctx).Begin()
	if first.Error != nil {
		t.Fatal(first.Error)
	}
	defer first.Rollback()
	if err = Emit(first, "conversation_events", []uint64{user}, &pb.Event{
		Kind:           pb.EventKind_GROUP_MEMBER_CHANGED,
		ConversationId: 100,
	}); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	committed := make(chan error, 1)
	go func() {
		second := db.WithContext(ctx).Begin()
		if second.Error != nil {
			committed <- second.Error
			return
		}
		close(started)
		if err := Emit(second, "conversation_events", []uint64{user}, &pb.Event{
			Kind:           pb.EventKind_GROUP_MEMBER_CHANGED,
			ConversationId: 200,
		}); err != nil {
			_ = second.Rollback()
			committed <- err
			return
		}
		committed <- second.Commit().Error
	}()
	select {
	case <-started:
	case err := <-committed:
		t.Fatalf("第二个事务启动失败: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	// 持有首个事务时，第二个事务必须等待；旧实现会直接提交较大游标。
	select {
	case err := <-committed:
		t.Fatalf("第二个事件越过未提交事件: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	var visible int64
	if err = db.Table("conversation_events").Where("user_id = ?", user).Count(&visible).Error; err != nil {
		t.Fatal(err)
	}
	if visible != 0 {
		t.Fatalf("首个事件提交前已经看到后续事件: %d", visible)
	}
	if err = first.Commit().Error; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-committed:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	var rows []storage.Event
	if err = db.Table("conversation_events").Where("user_id = ?", user).Order("id").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].ID >= rows[1].ID {
		t.Fatalf("游标序列不正确: %s", fmt.Sprint(rows))
	}
}
