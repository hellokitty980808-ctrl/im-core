package biz

import (
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestOriginPolicy(t *testing.T) {
	t.Setenv("WS_ALLOWED_ORIGINS", "")
	req := httptest.NewRequest("GET", "http://localhost:8081/v2/ws", nil)
	req.Header.Set("Origin", "https://untrusted.example")
	if originAllowed(req) {
		t.Fatal("跨站长连接未被拒绝")
	}
	req.Header.Set("Origin", "http://localhost:8081")
	if !originAllowed(req) {
		t.Fatal("同源长连接被拒绝")
	}
	t.Setenv("WS_ALLOWED_ORIGINS", "https://client.example")
	req.Header.Set("Origin", "https://client.example")
	if !originAllowed(req) {
		t.Fatal("显式允许来源被拒绝")
	}
}
func TestClosedPeerRejectsDelivery(t *testing.T) {
	// 已关闭的连接不得继续接受后台投递。
	p := &peer{
		conn:     &websocket.Conn{},
		outgoing: make(chan Frame, 1),
		done:     make(chan struct{}),
	}
	close(p.done)
	if p.send(Frame{
		Version: 2,
		Type:    "message",
	}) {
		t.Fatal("已经退出的连接仍接收推送")
	}
}

// TestHubCloseWaitsForWriters 验证已认证连接的 writer 在关闭返回前退出。
func TestHubCloseWaitsForWriters(t *testing.T) {
	hub := NewHub(testAccounts{})
	hub.SetReady(true)
	server := httptest.NewServer(hub)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err = conn.WriteJSON(Frame{
		Version: 2,
		Type:    "auth",
		Data:    json.RawMessage(`{"token":"test-only"}`),
	}); err != nil {
		t.Fatal(err)
	}
	var ready Frame
	if err = conn.ReadJSON(&ready); err != nil || ready.Type != "ready" {
		t.Fatalf("认证失败: %v", err)
	}
	done := make(chan struct{})
	go func() {
		hub.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Hub 关闭未等待连接退出")
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err = conn.ReadMessage(); err == nil {
		t.Fatal("关闭后连接仍可读")
	}
}

type testAccounts struct{ pb.AccountServiceClient }

func (testAccounts) Authenticate(context.Context, *pb.SessionToken, ...grpc.CallOption) (*pb.Identity, error) {
	return &pb.Identity{
		UserId:    42,
		SessionId: "test-only",
	}, nil
}
