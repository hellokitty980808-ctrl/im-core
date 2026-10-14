package biz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"
	"github.com/hellokitty980808-ctrl/im-core/pkg/delivery"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Frame 是 v2 长连接公开封装；每个 WebSocket 文本帧承载一个 JSON 对象。
type Frame struct {
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}
type peer struct {
	user     uint64
	token    string
	conn     *websocket.Conn
	outgoing chan Frame
	done     chan struct{}
	once     sync.Once
}

func (p *peer) close() {
	p.once.Do(func() {
		close(p.done)
		_ = p.conn.Close()
	})
}
func (p *peer) send(frame Frame) bool {
	select {
	case <-p.done:
		return false
	default:
	}
	select {
	case p.outgoing <- frame:
		return true
	default:
		p.close()
		return false
	}
}

// Hub 保存当前实例连接；所有写操作由连接自己的唯一 writer 执行。
type Hub struct {
	users    pb.AccountServiceClient
	mu       sync.RWMutex
	peers    map[*peer]struct{}
	ready    atomic.Bool
	writers  sync.WaitGroup
	stopping bool
}

// NewHub 创建实例内用户连接集合，订阅就绪后才允许认证。
func NewHub(users pb.AccountServiceClient) *Hub {
	return &Hub{
		users: users,
		peers: make(map[*peer]struct{}),
	}
}
func originAllowed(req *http.Request) bool {
	origin := req.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if allowed := os.Getenv("WS_ALLOWED_ORIGINS"); allowed != "" {
		for _, candidate := range strings.Split(allowed, ",") {
			if strings.TrimSpace(candidate) == origin {
				return true
			}
		}
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	return parsed.Host == req.Host
}

// ServeHTTP 认证在首个 auth 帧完成，五秒内未认证即关闭，不接受旧包头或旧路径。
func (h *Hub) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !h.ready.Load() {
		http.Error(w, "delivery unavailable", http.StatusServiceUnavailable)
		return
	}
	upgrader := websocket.Upgrader{CheckOrigin: originAllowed}
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(8192)
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var frame Frame
	kind, raw, err := conn.ReadMessage()
	if err != nil || kind != websocket.TextMessage {
		return
	}
	if err = json.Unmarshal(raw, &frame); err != nil || frame.Version != 2 || frame.Type != "auth" || len(frame.RequestID) > 64 {
		return
	}
	var credentials struct {
		Token string `json:"token"`
	}
	if err = json.Unmarshal(frame.Data, &credentials); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
	identity, err := h.users.Authenticate(ctx, &pb.SessionToken{Token: credentials.Token})
	cancel()
	if err != nil {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "authentication failed"),
			time.Now().Add(time.Second),
		)
		return
	}
	p := &peer{
		user:     identity.UserId,
		token:    credentials.Token,
		conn:     conn,
		outgoing: make(chan Frame, 64),
		done:     make(chan struct{}),
	}
	info, _ := json.Marshal(struct {
		ConnectionID     string `json:"connectionId"`
		UserID           uint64 `json:"userId,string"`
		HeartbeatSeconds int    `json:"heartbeatSeconds"`
	}{
		ConnectionID:     uuid.NewString(),
		UserID:           p.user,
		HeartbeatSeconds: 20,
	})
	p.send(Frame{
		Version:   2,
		Type:      "ready",
		RequestID: frame.RequestID,
		Data:      info,
	})
	h.mu.Lock()
	if h.stopping {
		h.mu.Unlock()
		p.close()
		return
	}
	h.peers[p] = struct{}{}
	h.writers.Add(1)
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.peers, p)
		h.mu.Unlock()
		p.close()
	}()
	// reader 与 writer 各自独占方向，后台投递只入队，不直接写连接。
	go h.write(p)
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(60 * time.Second)) })
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	for {
		kind, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if kind != websocket.TextMessage {
			return
		}
		var in Frame
		if err = json.Unmarshal(raw, &in); err != nil || in.Version != 2 || in.Type != "ping" || len(in.RequestID) > 64 {
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
		_, err = h.users.Authenticate(ctx, &pb.SessionToken{Token: p.token})
		cancel()
		if err != nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		if !p.send(Frame{
			Version:   2,
			Type:      "pong",
			RequestID: in.RequestID,
		}) {
			return
		}
	}
}
func (h *Hub) write(p *peer) {
	defer h.writers.Done()
	defer p.close()
	timer := time.NewTicker(20 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-p.done:
			return
		case frame := <-p.outgoing:
			_ = p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := p.conn.WriteJSON(frame); err != nil {
				return
			}
		case <-timer.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := h.users.Authenticate(ctx, &pb.SessionToken{Token: p.token})
			cancel()
			if err != nil {
				return
			}
			if err := p.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				return
			}
		}
	}
}

// Deliver 只路由至指定用户；慢连接关闭后依靠持久化历史和事件恢复。
func (h *Hub) Deliver(envelope delivery.Envelope) {
	targets := make(map[uint64]bool, len(envelope.Users))
	for _, id := range envelope.Users {
		targets[id] = true
	}
	h.mu.RLock()
	peers := make([]*peer, 0)
	for p := range h.peers {
		if targets[p.user] {
			peers = append(peers, p)
		}
	}
	h.mu.RUnlock()
	for _, p := range peers {
		// 每次投递重新校验会话，已注销或被新登录替换的连接不能继续收到消息。
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := h.users.Authenticate(ctx, &pb.SessionToken{Token: p.token})
		cancel()
		if err != nil {
			p.close()
			continue
		}
		p.send(Frame{
			Version: 2,
			Type:    envelope.Type,
			Data:    envelope.Data,
		})
	}
}

// Close 停止接受新连接并关闭现有连接，可重复调用。
func (h *Hub) Close() {
	h.mu.Lock()
	h.stopping = true
	peers := make([]*peer, 0, len(h.peers))
	for p := range h.peers {
		peers = append(peers, p)
	}
	h.mu.Unlock()
	for _, p := range peers {
		p.close()
	}
	h.writers.Wait()
}

// SetReady 在订阅中断时暂停接收新连接，防止报告认证成功却无法订阅投递。
func (h *Hub) SetReady(ready bool) { h.ready.Store(ready) }
