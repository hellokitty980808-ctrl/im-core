package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestOutboxRecovery 在隔离 Compose 中停启 RabbitMQ，验证已提交消息的重试恢复。
func TestOutboxRecovery(t *testing.T) {
	if os.Getenv("IMCORE_E2E") != "1" || os.Getenv("IMCORE_FAULTS") != "1" {
		t.Skip("仅在隔离 Compose 中设置 IMCORE_E2E=1 和 IMCORE_FAULTS=1 执行停机测试")
	}
	compose := func(args ...string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, "docker", append([]string{"compose"}, args...)...)
		command.Dir = "../.."
		output, err := command.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Compose %v: %w: %s", args, err, string(output))
		}
		return nil
	}
	base := os.Getenv("IMCORE_URL")
	if base == "" {
		base = "http://127.0.0.1:18000"
	}
	endpoint := os.Getenv("IMCORE_WS_URL")
	if endpoint == "" {
		endpoint = "ws://127.0.0.1:16200/v2/ws"
	}
	prefix := fmt.Sprintf("r%d", time.Now().UnixNano()%1000000000000)
	a := register(t, base, prefix+"a")
	b := register(t, base, prefix+"b")
	direct := makeFriends(t, a, b)
	conn := websocketClient(t, endpoint, b.token)
	baseline := send(t, a, direct, "before-outage", "队列正常")
	awaitMessage(t, conn, baseline.MessageId)
	t.Cleanup(func() {
		if err := compose("up", "-d", "--no-build", "--wait", "rabbitmq"); err != nil {
			t.Error(err)
		}
	})
	if err := compose("stop", "rabbitmq"); err != nil {
		t.Fatal(err)
	}
	pending := send(t, a, direct, "during-outage", "停机期间持久化")
	rows := history(t, b, direct, baseline.Seq)
	if len(rows.Messages) != 1 || rows.Messages[0].MessageId != pending.MessageId {
		t.Fatal("队列停机导致消息丢失")
	}
	if err := compose("up", "-d", "--no-build", "--wait", "rabbitmq"); err != nil {
		t.Fatal(err)
	}
	received := awaitMessage(t, conn, pending.MessageId)
	if received.Content != pending.Content {
		t.Fatal("队列恢复后投递内容改变")
	}
	after := send(t, a, direct, "after-outage", "队列恢复")
	awaitMessage(t, conn, after.MessageId)
}
