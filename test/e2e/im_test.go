// Package e2e 验证四服务、真实数据库和 JSON 长连接的基础 IM 契约。
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	pb "github.com/hellokitty980808-ctrl/im-core/api/im/v2"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type client struct {
	base  string
	token string
	user  *pb.Profile
}

func (c *client) request(method, path string, in, out proto.Message) (int, error) {
	var encoded []byte
	var err error
	if in != nil {
		encoded, err = protojson.Marshal(in)
		if err != nil {
			return 0, err
		}
	}
	req, err := http.NewRequest(method, c.base+path, bytes.NewReader(encoded))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	// 用户提供的身份元数据不能覆盖认证结果。
	req.Header.Set("imcore-user-id", "1")
	res, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return res.StatusCode, err
	}
	if res.StatusCode != 200 {
		var failure struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err = json.Unmarshal(raw, &failure); err != nil {
			return res.StatusCode, fmt.Errorf("HTTP %d", res.StatusCode)
		}
		return res.StatusCode, fmt.Errorf("%s: %s", failure.Error.Code, failure.Error.Message)
	}
	if out != nil {
		err = protojson.Unmarshal(raw, out)
	}
	return res.StatusCode, err
}
func (c *client) call(t *testing.T, method, path string, in, out proto.Message) {
	t.Helper()
	code, err := c.request(method, path, in, out)
	if err != nil || code != 200 {
		t.Fatalf("%s %s: HTTP %d, %v", method, path, code, err)
	}
}
func (c *client) denied(t *testing.T, method, path string, in proto.Message, status int) {
	t.Helper()
	code, err := c.request(method, path, in, nil)
	if code != status || err == nil {
		t.Fatalf("%s %s: got HTTP %d, want %d, err=%v", method, path, code, status, err)
	}
}
func register(t *testing.T, base, account string) *client {
	t.Helper()
	c := &client{base: base}
	result := &pb.Session{}
	c.call(t, "POST", "/v2/accounts", &pb.Registration{
		Account:  account,
		Password: "LocalExample_2026!",
		Nickname: "测试用户",
	}, result)
	if result.User == nil || result.User.UserId == 0 || result.Token == "" {
		t.Fatal("注册未返回账户和会话")
	}
	if result.ExpiresAt < time.Now().UnixMilli() {
		t.Fatal("会话过期时间不是未来的毫秒时间戳")
	}
	c.token = result.Token
	c.user = result.User
	return c
}
func makeFriends(t *testing.T, a, b *client) uint64 {
	t.Helper()
	request := &pb.FriendRequest{}
	a.call(t, "POST", "/v2/friend-requests", &pb.FriendRequestDraft{
		UserId: b.user.UserId,
		Note:   "你好",
	}, request)
	if request.State != pb.RequestState_PENDING {
		t.Fatal("好友申请不是待处理状态")
	}
	accepted := &pb.FriendRequest{}
	b.call(
		t,
		"POST",
		fmt.Sprintf("/v2/friend-requests/%d/accept", request.RequestId),
		&emptypb.Empty{},
		accepted,
	)
	if accepted.State != pb.RequestState_ACCEPTED || accepted.ConversationId == 0 {
		t.Fatal("接受好友申请没有创建私聊")
	}
	return accepted.ConversationId
}
func messagesPath(conversation uint64) string {
	return fmt.Sprintf("/v2/conversations/%d/messages", conversation)
}
func send(t *testing.T, c *client, conversation uint64, key, content string) *pb.Message {
	t.Helper()
	out := &pb.Message{}
	c.call(t, "POST", messagesPath(conversation), &pb.MessageDraft{
		Content:        content,
		IdempotencyKey: key,
	}, out)
	if out.MessageId == 0 || out.Seq == 0 || out.ConversationId != conversation || out.Sender.GetUserId() != c.user.UserId {
		t.Fatal("发送响应缺少正确消息、会话或发送者")
	}
	return out
}
func history(t *testing.T, c *client, conversation, after uint64) *pb.Messages {
	t.Helper()
	out := &pb.Messages{}
	c.call(
		t,
		"GET",
		fmt.Sprintf("%s?afterSeq=%d&limit=100", messagesPath(conversation), after),
		nil,
		out,
	)
	return out
}

type frame struct {
	Version   int             `json:"version"`
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

func websocketClient(t *testing.T, endpoint, token string) *websocket.Conn {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	var conn *websocket.Conn
	var err error
	for {
		conn, _, err = websocket.DefaultDialer.Dial(endpoint, nil)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("长连接握手: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Cleanup(func() { _ = conn.Close() })
	credentials, _ := json.Marshal(struct {
		Token string `json:"token"`
	}{Token: token})
	if err = conn.WriteJSON(frame{
		Version:   2,
		Type:      "auth",
		RequestID: "authenticate",
		Data:      credentials,
	}); err != nil {
		t.Fatal(err)
	}
	out := readFrame(t, conn)
	if out.Type != "ready" || out.RequestID != "authenticate" {
		t.Fatalf("认证响应不是 ready: %s", out.Type)
	}
	return conn
}
func readFrame(t *testing.T, conn *websocket.Conn) frame {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	kind, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("读取长连接: %v", err)
	}
	if kind != websocket.TextMessage {
		t.Fatal("v2 长连接返回了二进制帧")
	}
	var out frame
	if err = json.Unmarshal(raw, &out); err != nil || out.Version != 2 {
		t.Fatalf("长连接封装无效: %v", err)
	}
	return out
}
func awaitMessage(t *testing.T, conn *websocket.Conn, id uint64) *pb.Message {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		out := readFrame(t, conn)
		if out.Type != "message" {
			continue
		}
		value := &pb.Message{}
		if err := protojson.Unmarshal(out.Data, value); err != nil {
			t.Fatal(err)
		}
		if value.MessageId == id {
			return value
		}
	}
	t.Fatal("没有收到目标文字消息")
	return nil
}
func awaitEvent(t *testing.T, conn *websocket.Conn, kind pb.EventKind) *pb.Event {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		out := readFrame(t, conn)
		if out.Type != "event" {
			continue
		}
		value := &pb.Event{}
		if err := protojson.Unmarshal(out.Data, value); err != nil {
			t.Fatal(err)
		}
		if value.Kind == kind {
			return value
		}
	}
	t.Fatal("没有收到目标同步事件")
	return nil
}

// TestBasicIM 验证基础 IM 完整链路；需显式开启并使用关闭验证码的隔离环境。
func TestBasicIM(t *testing.T) {
	if os.Getenv("IMCORE_E2E") != "1" {
		t.Skip("设置 IMCORE_E2E=1，在隔离环境临时关闭验证码后运行")
	}
	base := os.Getenv("IMCORE_URL")
	if base == "" {
		base = "http://127.0.0.1:18000"
	}
	wsURL := os.Getenv("IMCORE_WS_URL")
	if wsURL == "" {
		wsURL = "ws://127.0.0.1:16200/v2/ws"
	}
	deadline := time.Now().Add(30 * time.Second)
	probe := &client{base: base}
	for {
		code, err := probe.request("GET", "/v2/captcha", nil, &pb.Captcha{})
		if code == 200 && err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("服务未就绪: HTTP %d, %v", code, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	prefix := fmt.Sprintf("v%d", time.Now().UnixNano()%1000000000000)
	a := register(t, base, prefix+"a")
	b := register(t, base, prefix+"b")
	c := register(t, base, prefix+"c")
	d := register(t, base, prefix+"d")
	deadline = time.Now().Add(30 * time.Second)
	for {
		status, err := a.request("GET", "/v2/conversations", nil, &pb.Conversations{})
		if status == 200 && err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("会话服务未就绪: HTTP %d, %v", status, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	run := func(name string, test func(*testing.T)) {
		if !t.Run(name, test) {
			t.FailNow()
		}
	}
	run("新协议与身份隔离", func(t *testing.T) {
		probe.denied(t, "GET", "/v2/users/me", nil, 401)
		probe.denied(t, "POST", "/v1/auth/login", &emptypb.Empty{}, 404)
		probe.denied(t, "GET", "/v1/health", nil, 404)
		info := &pb.Profile{}
		b.call(t, "GET", "/v2/users/me", nil, info)
		if info.UserId != b.user.UserId {
			t.Fatal("客户端身份头覆盖了登录身份")
		}
		a.call(t, "GET", "/v2/users?account="+b.user.Account, nil, info)
		if info.UserId != b.user.UserId {
			t.Fatal("账号搜索结果不正确")
		}
		nick := "独立昵称"
		bio := "基础 IM 用户"
		avatar := "https://example.com/avatar.png"
		a.call(t, "PATCH", "/v2/users/me", &pb.ProfileUpdate{
			Nickname:  &nick,
			Bio:       &bio,
			AvatarUrl: &avatar,
		}, info)
		if info.Nickname != nick || info.Bio != bio {
			t.Fatal("资料修改未保存")
		}
		a.user = info
		// 旧业务字段不能混入新请求。
		req, err := http.NewRequest("PATCH", a.base+"/v2/users/me", strings.NewReader(`{"phoneNumbers":["ignored"]}`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+a.token)
		req.Header.Set("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != 400 {
			t.Fatalf("旧业务字段被接受: HTTP %d", res.StatusCode)
		}

		probe.denied(t, "POST", "/v2/accounts", &pb.Registration{
			Account:  a.user.Account,
			Password: "LocalExample_2026!",
			Nickname: "测试用户",
		}, 409)
		probe.denied(t, "POST", "/v2/sessions", &pb.Credentials{
			Account:  a.user.Account,
			Password: "wrong-password",
		}, 401)
	})
	var direct uint64
	run("好友接受拒绝取消和重复申请", func(t *testing.T) {
		request := &pb.FriendRequest{}
		c.call(t, "POST", "/v2/friend-requests", &pb.FriendRequestDraft{UserId: d.user.UserId}, request)
		duplicate := &pb.FriendRequest{}
		c.call(
			t,
			"POST",
			"/v2/friend-requests",
			&pb.FriendRequestDraft{UserId: d.user.UserId},
			duplicate,
		)
		if request.RequestId != duplicate.RequestId {
			t.Fatal("重复申请生成了多条待处理记录")
		}
		c.denied(
			t,
			"POST",
			fmt.Sprintf("/v2/friend-requests/%d/accept", request.RequestId),
			&emptypb.Empty{},
			403,
		)
		d.call(
			t,
			"POST",
			fmt.Sprintf("/v2/friend-requests/%d/reject", request.RequestId),
			&emptypb.Empty{},
			duplicate,
		)
		if duplicate.State != pb.RequestState_REJECTED {
			t.Fatal("拒绝状态未保存")
		}
		c.call(t, "POST", "/v2/friend-requests", &pb.FriendRequestDraft{UserId: d.user.UserId}, request)
		c.call(
			t,
			"POST",
			fmt.Sprintf("/v2/friend-requests/%d/cancel", request.RequestId),
			&emptypb.Empty{},
			duplicate,
		)
		if duplicate.State != pb.RequestState_CANCELLED {
			t.Fatal("取消状态未保存")
		}
		direct = makeFriends(t, a, b)
		makeFriends(t, a, c)
		makeFriends(t, a, d)
		friends := &pb.Friends{}
		a.call(t, "GET", "/v2/friends?limit=1", nil, friends)
		if len(friends.Friends) != 1 || friends.NextCursor == 0 {
			t.Fatal("好友分页无效")
		}
		a.call(
			t,
			"GET",
			fmt.Sprintf("/v2/friends?cursor=%d&limit=100", friends.NextCursor),
			nil,
			friends,
		)
		if len(friends.Friends) != 2 {
			t.Fatal("好友分页丢失或重复")
		}
		friend := &pb.Friend{}
		a.call(
			t,
			"PATCH",
			fmt.Sprintf("/v2/friends/%d", b.user.UserId),
			&pb.FriendUpdate{Remark: "好友备注"},
			friend,
		)
		if friend.Relationship != pb.Relationship_FRIEND || friend.Remark != "好友备注" || friend.ConversationId != direct {
			t.Fatal("好友信息不正确")
		}
	})
	var socket *websocket.Conn
	var first *pb.Message
	run("JSON认证心跳与在线私聊", func(t *testing.T) {
		socket = websocketClient(t, wsURL, b.token)
		first = send(t, a, direct, "first-private", "你好")
		received := awaitMessage(t, socket, first.MessageId)
		if received.Content != "你好" || received.Sender.UserId != a.user.UserId {
			t.Fatal("推送内容或发送者不正确")
		}
		if err := socket.WriteJSON(frame{
			Version:   2,
			Type:      "ping",
			RequestID: "heartbeat",
		}); err != nil {
			t.Fatal(err)
		}
		for {
			value := readFrame(t, socket)
			if value.Type == "pong" {
				if value.RequestID != "heartbeat" {
					t.Fatal("心跳请求号不匹配")
				}
				break
			}
		}
	})
	// 上个子用例的清理会关闭连接，后续离线同步必须不依赖长连接。
	run("离线补拉已读与消息权限", func(t *testing.T) {
		second := send(t, a, direct, "second-private", "离线消息")
		rows := history(t, b, direct, first.Seq)
		if len(rows.Messages) != 1 || rows.Messages[0].MessageId != second.MessageId {
			t.Fatal("离线补拉未按序号返回新消息")
		}
		read := &pb.ReadProgress{}
		b.call(
			t,
			"PATCH",
			fmt.Sprintf("/v2/conversations/%d/read-progress", direct),
			&pb.ReadProgress{Seq: second.Seq},
			read,
		)
		b.call(
			t,
			"PATCH",
			fmt.Sprintf("/v2/conversations/%d/read-progress", direct),
			&pb.ReadProgress{Seq: first.Seq},
			read,
		)
		if read.Seq != second.Seq {
			t.Fatal("已读进度发生回退")
		}
		b.denied(
			t,
			"PATCH",
			fmt.Sprintf("/v2/conversations/%d/read-progress", direct),
			&pb.ReadProgress{Seq: 999999},
			400,
		)
		c.denied(t, "POST", messagesPath(direct), &pb.MessageDraft{
			Content:        "越权",
			IdempotencyKey: "outsider",
		}, 403)
		c.denied(t, "GET", messagesPath(direct), nil, 403)
		b.denied(
			t,
			"POST",
			fmt.Sprintf("%s/%d/recall", messagesPath(direct), first.MessageId),
			&emptypb.Empty{},
			403,
		)
		a.denied(t, "POST", messagesPath(direct), &pb.MessageDraft{
			Content:        strings.Repeat("x", 2049),
			IdempotencyKey: "oversize",
		}, 400)
	})
	run("发送幂等与并发重放", func(t *testing.T) {
		original := send(t, a, direct, "concurrent-private", "只保存一次")
		var wg sync.WaitGroup
		failures := make(chan error, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				out := &pb.Message{}
				_, err := a.request("POST", messagesPath(direct), &pb.MessageDraft{
					Content:        "只保存一次",
					IdempotencyKey: "concurrent-private",
				}, out)
				if err == nil && !proto.Equal(out, original) {
					err = fmt.Errorf("重放结果发生变化")
				}
				failures <- err
			}()
		}
		wg.Wait()
		close(failures)
		for err := range failures {
			if err != nil {
				t.Error(err)
			}
		}
		a.denied(t, "POST", messagesPath(direct), &pb.MessageDraft{
			Content:        "不同内容",
			IdempotencyKey: "concurrent-private",
		}, 409)
		rows := history(t, b, direct, original.Seq-1)
		if len(rows.Messages) != 1 || rows.Messages[0].MessageId != original.MessageId {
			t.Fatal("重试产生了重复消息")
		}
		// 未预先发送的同一键也必须只提交一次。
		failures = make(chan error, 8)
		ids := make(chan uint64, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				out := &pb.Message{}
				_, err := a.request("POST", messagesPath(direct), &pb.MessageDraft{
					Content:        "首次并发",
					IdempotencyKey: "first-concurrent",
				}, out)
				failures <- err
				ids <- out.MessageId
			}()
		}
		wg.Wait()
		close(failures)
		close(ids)
		for err := range failures {
			if err != nil {
				t.Error(err)
			}
		}
		var winner uint64
		for id := range ids {
			if winner == 0 {
				winner = id
			}
			if id == 0 || id != winner {
				t.Fatal("首次并发产生不同消息 ID")
			}
		}
	})
	run("拉黑解除与好友事件", func(t *testing.T) {
		a.call(
			t,
			"POST",
			fmt.Sprintf("/v2/blocks/%d", b.user.UserId),
			&emptypb.Empty{},
			&emptypb.Empty{},
		)
		b.denied(t, "POST", messagesPath(direct), &pb.MessageDraft{
			Content:        "被拉黑",
			IdempotencyKey: "blocked",
		}, 403)
		list := &pb.Friends{}
		a.call(t, "GET", "/v2/blocks", nil, list)
		if len(list.Friends) != 1 || list.Friends[0].Relationship != pb.Relationship_BLOCKED {
			t.Fatal("黑名单不正确")
		}
		a.call(t, "DELETE", fmt.Sprintf("/v2/blocks/%d", b.user.UserId), nil, &emptypb.Empty{})
		send(t, b, direct, "unblocked", "恢复发送")
		events := &pb.Events{}
		a.call(t, "GET", "/v2/friend-events?limit=100", nil, events)
		if len(events.Events) == 0 {
			t.Fatal("好友事件没有持久化")
		}
		cursor := events.NextCursor
		a.call(t, "POST", "/v2/friend-events/ack", &pb.EventAck{Cursor: cursor}, &emptypb.Empty{})
		a.call(t, "GET", fmt.Sprintf("/v2/friend-events?cursor=%d", cursor), nil, events)
		if len(events.Events) != 0 || events.AcknowledgedCursor != cursor {
			t.Fatal("好友事件确认或补拉有误")
		}
		a.denied(t, "POST", "/v2/friend-events/ack", &pb.EventAck{Cursor: 999999999999}, 400)
	})
	var group *pb.Group
	run("群审批权限与成员", func(t *testing.T) {
		group = &pb.Group{}
		a.call(t, "POST", "/v2/groups", &pb.GroupDraft{
			Name:            "基础群组",
			MemberUserIds:   []uint64{b.user.UserId},
			RequireApproval: true,
		}, group)
		if group.MemberCount != 2 || group.ConversationId == direct || group.GroupId == group.ConversationId {
			t.Fatal("群资料与会话未正确分离")
		}
		join := &pb.JoinRequest{}
		c.call(
			t,
			"POST",
			fmt.Sprintf("/v2/groups/%d/join-requests", group.GroupId),
			&pb.JoinDraft{Note: "申请入群"},
			join,
		)
		if join.State != pb.RequestState_PENDING {
			t.Fatal("审批群直接加入了成员")
		}
		c.denied(t, "POST", messagesPath(group.ConversationId), &pb.MessageDraft{
			Content:        "尚未入群",
			IdempotencyKey: "pending",
		}, 403)
		b.denied(
			t,
			"POST",
			fmt.Sprintf("/v2/groups/%d/join-requests/%d/accept", group.GroupId, join.RequestId),
			&emptypb.Empty{},
			403,
		)
		a.call(
			t,
			"POST",
			fmt.Sprintf("/v2/groups/%d/join-requests/%d/accept", group.GroupId, join.RequestId),
			&emptypb.Empty{},
			join,
		)
		if join.State != pb.RequestState_ACCEPTED {
			t.Fatal("入群审批未生效")
		}
		rejected := &pb.JoinRequest{}
		d.call(
			t,
			"POST",
			fmt.Sprintf("/v2/groups/%d/join-requests", group.GroupId),
			&pb.JoinDraft{},
			rejected,
		)
		a.call(
			t,
			"POST",
			fmt.Sprintf("/v2/groups/%d/join-requests/%d/reject", group.GroupId, rejected.RequestId),
			&emptypb.Empty{},
			rejected,
		)
		if rejected.State != pb.RequestState_REJECTED {
			t.Fatal("入群拒绝未生效")
		}
		d.denied(t, "POST", messagesPath(group.ConversationId), &pb.MessageDraft{
			Content:        "已拒绝",
			IdempotencyKey: "rejected",
		}, 403)
		a.call(
			t,
			"POST",
			fmt.Sprintf("/v2/groups/%d/members", group.GroupId),
			&pb.MemberInvite{UserIds: []uint64{d.user.UserId, d.user.UserId}},
			group,
		)
		if group.MemberCount != 4 {
			t.Fatal("邀请未生效或重复成员计数")
		}
		members := &pb.GroupMembers{}
		a.call(t, "GET", fmt.Sprintf("/v2/groups/%d/members", group.GroupId), nil, members)
		if len(members.Members) != 4 {
			t.Fatal("群成员列表错误")
		}
		b.denied(
			t,
			"PATCH",
			fmt.Sprintf("/v2/groups/%d", group.GroupId),
			&pb.GroupUpdate{Name: "越权改名"},
			403,
		)
		a.call(
			t,
			"PATCH",
			fmt.Sprintf("/v2/groups/%d", group.GroupId),
			&pb.GroupUpdate{Name: "新群名"},
			group,
		)
		if group.Name != "新群名" {
			t.Fatal("群名修改未生效")
		}
	})
	run("群聊在线投递撤回与事件确认", func(t *testing.T) {
		conn := websocketClient(t, wsURL, b.token)
		sent := send(t, a, group.ConversationId, "group-message", "群消息")
		awaitMessage(t, conn, sent.MessageId)
		recalled := &pb.Message{}
		a.call(
			t,
			"POST",
			fmt.Sprintf("%s/%d/recall", messagesPath(group.ConversationId), sent.MessageId),
			&emptypb.Empty{},
			recalled,
		)
		if !recalled.Recalled || recalled.Content != "" {
			t.Fatal("撤回未清除可见内容")
		}
		event := awaitEvent(t, conn, pb.EventKind_MESSAGE_RECALLED)
		if event.MessageId != sent.MessageId || event.MessageSeq != sent.Seq {
			t.Fatal("撤回事件对应消息有误")
		}
		rows := history(t, b, group.ConversationId, 0)
		if len(rows.Messages) != 1 || !rows.Messages[0].Recalled || rows.Messages[0].Content != "" {
			t.Fatal("离线历史未体现撤回")
		}
		events := &pb.Events{}
		b.call(t, "GET", "/v2/conversation-events?limit=100", nil, events)
		found := false
		for _, e := range events.Events {
			if e.Kind == pb.EventKind_MESSAGE_RECALLED && e.MessageId == sent.MessageId {
				found = true
			}
		}
		if !found {
			t.Fatal("离线补拉缺少撤回事件")
		}
		cursor := events.NextCursor
		b.call(t, "POST", "/v2/conversation-events/ack", &pb.EventAck{Cursor: cursor}, &emptypb.Empty{})
		b.call(t, "POST", "/v2/conversation-events/ack", &pb.EventAck{Cursor: 0}, &emptypb.Empty{})
		b.call(t, "GET", fmt.Sprintf("/v2/conversation-events?cursor=%d", cursor), nil, events)
		if len(events.Events) != 0 || events.AcknowledgedCursor != cursor {
			t.Fatal("事件确认回退或补拉重复")
		}
	})
	run("移除退出解散及离线成员状态", func(t *testing.T) {
		b.denied(
			t,
			"DELETE",
			fmt.Sprintf("/v2/groups/%d/members/%d", group.GroupId, c.user.UserId),
			nil,
			403,
		)
		a.call(
			t,
			"DELETE",
			fmt.Sprintf("/v2/groups/%d/members/%d", group.GroupId, c.user.UserId),
			nil,
			&emptypb.Empty{},
		)
		c.denied(t, "POST", messagesPath(group.ConversationId), &pb.MessageDraft{
			Content:        "被移除",
			IdempotencyKey: "removed",
		}, 403)
		c.denied(t, "GET", messagesPath(group.ConversationId), nil, 403)
		c.call(
			t,
			"GET",
			fmt.Sprintf("/v2/conversations/%d", group.ConversationId),
			nil,
			&pb.Conversation{},
		)
		events := &pb.Events{}
		c.call(t, "GET", "/v2/conversation-events?limit=100", nil, events)
		found := false
		for _, event := range events.Events {
			if event.UserId == c.user.UserId && event.State == pb.ConversationState_LEFT {
				found = true
			}
		}
		if !found {
			t.Fatal("被移除成员没有收到离线事件")
		}
		d.call(
			t,
			"POST",
			fmt.Sprintf("/v2/groups/%d/leave", group.GroupId),
			&emptypb.Empty{},
			&emptypb.Empty{},
		)
		a.denied(t, "POST", fmt.Sprintf("/v2/groups/%d/leave", group.GroupId), &emptypb.Empty{}, 409)
		b.denied(t, "DELETE", fmt.Sprintf("/v2/groups/%d", group.GroupId), nil, 403)
		a.call(t, "DELETE", fmt.Sprintf("/v2/groups/%d", group.GroupId), nil, &emptypb.Empty{})
		b.denied(t, "POST", messagesPath(group.ConversationId), &pb.MessageDraft{
			Content:        "群已解散",
			IdempotencyKey: "closed",
		}, 403)
		conversation := &pb.Conversation{}
		b.call(t, "GET", fmt.Sprintf("/v2/conversations/%d", group.ConversationId), nil, conversation)
		if conversation.State != pb.ConversationState_CLOSED {
			t.Fatal("解散状态未同步")
		}
		a.call(t, "DELETE", fmt.Sprintf("/v2/groups/%d", group.GroupId), nil, &emptypb.Empty{})
	})
	run("开放群入群和加入前历史隔离", func(t *testing.T) {
		open := &pb.Group{}
		a.call(t, "POST", "/v2/groups", &pb.GroupDraft{Name: "开放群"}, open)
		send(t, a, open.ConversationId, "before-join", "加入前消息")
		join := &pb.JoinRequest{}
		c.call(
			t,
			"POST",
			fmt.Sprintf("/v2/groups/%d/join-requests", open.GroupId),
			&pb.JoinDraft{},
			join,
		)
		if join.State != pb.RequestState_ACCEPTED {
			t.Fatal("开放群仍需审批")
		}
		if len(history(t, c, open.ConversationId, 0).Messages) != 0 {
			t.Fatal("新成员看到了加入前消息")
		}
		after := send(t, a, open.ConversationId, "after-join", "加入后消息")
		rows := history(t, c, open.ConversationId, 0)
		if len(rows.Messages) != 1 || rows.Messages[0].MessageId != after.MessageId {
			t.Fatal("新成员未看到加入后消息")
		}
		a.call(t, "DELETE", fmt.Sprintf("/v2/groups/%d", open.GroupId), nil, &emptypb.Empty{})
	})
	run("删除好友阻止私聊与重新建立", func(t *testing.T) {
		a.call(t, "DELETE", fmt.Sprintf("/v2/friends/%d", b.user.UserId), nil, &emptypb.Empty{})
		b.denied(t, "POST", messagesPath(direct), &pb.MessageDraft{
			Content:        "已删除",
			IdempotencyKey: "deleted",
		}, 403)
		restored := makeFriends(t, a, b)
		if restored != direct {
			t.Fatal("恢复好友重复创建了私聊")
		}
		send(t, b, direct, "restored", "好友恢复")
	})
	run("登录刷新密码修改和注销撤销", func(t *testing.T) {
		old := *a
		session := &pb.Session{}
		a.call(t, "POST", "/v2/sessions", &pb.Credentials{
			Account:  a.user.Account,
			Password: "LocalExample_2026!",
		}, session)
		a.token = session.Token
		old.denied(t, "GET", "/v2/users/me", nil, 401)
		old = *a
		a.call(t, "POST", "/v2/sessions/refresh", &emptypb.Empty{}, session)
		a.token = session.Token
		old.denied(t, "GET", "/v2/users/me", nil, 401)

		current := *a
		type refreshResult struct {
			code    int
			session *pb.Session
			err     error
		}
		refreshes := make(chan refreshResult, 2)
		for i := 0; i < 2; i++ {
			go func() {
				out := &pb.Session{}
				code, err := current.request("POST", "/v2/sessions/refresh", &emptypb.Empty{}, out)
				refreshes <- refreshResult{
					code:    code,
					session: out,
					err:     err,
				}
			}()
		}
		successes := 0
		for i := 0; i < 2; i++ {
			result := <-refreshes
			if result.code == 200 && result.err == nil {
				successes++
				a.token = result.session.Token
			} else if result.code != 401 {
				t.Fatalf("并发刷新异常: HTTP %d, %v", result.code, result.err)
			}
		}
		if successes != 1 {
			t.Fatalf("并发刷新成功次数: %d", successes)
		}
		current.denied(t, "GET", "/v2/users/me", nil, 401)
		a.call(t, "PATCH", "/v2/users/me/password", &pb.PasswordChange{
			CurrentPassword: "LocalExample_2026!",
			NewPassword:     "ChangedExample_2026!",
		}, &emptypb.Empty{})
		a.denied(t, "GET", "/v2/users/me", nil, 401)
		a.denied(t, "POST", "/v2/sessions", &pb.Credentials{
			Account:  a.user.Account,
			Password: "LocalExample_2026!",
		}, 401)
		a.call(t, "POST", "/v2/sessions", &pb.Credentials{
			Account:  a.user.Account,
			Password: "ChangedExample_2026!",
		}, session)
		a.token = session.Token

		conn := websocketClient(t, wsURL, a.token)
		a.call(t, "DELETE", "/v2/sessions/current", nil, &emptypb.Empty{})
		send(t, b, direct, "after-logout", "注销后应离线")
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) || websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					break
				}
				t.Fatalf("注销后的连接未关闭: %v", err)
			}
		}

		a.denied(t, "GET", "/v2/users/me", nil, 401)
	})
	run("无效长连接和旧路径拒绝", func(t *testing.T) {
		old := strings.TrimSuffix(wsURL, "/v2/ws") + "/sub"
		conn, res, err := websocket.DefaultDialer.Dial(old, nil)
		if conn != nil {
			_ = conn.Close()
		}
		if err == nil || res == nil || res.StatusCode != 404 {
			t.Fatal("旧长连接路径仍可访问")
		}
		conn, _, err = websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()
		credentials, _ := json.Marshal(map[string]string{"token": "invalid"})
		_ = conn.WriteJSON(frame{
			Version: 2,
			Type:    "auth",
			Data:    credentials,
		})
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, _, err = conn.ReadMessage()
		if !websocket.IsCloseError(err, websocket.ClosePolicyViolation) {
			t.Fatalf("无效凭据未被拒绝: %v", err)
		}
	})
}
