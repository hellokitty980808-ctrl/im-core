# v2 JSON 长连接

连接 `ws://127.0.0.1:16200/v2/ws`。每个 WebSocket 文本帧是一个 JSON 对象，不使用自定义二进制包头。版本必须为 2。

## 认证

连接建立后五秒内发送：

```json
{"version":2,"type":"auth","requestId":"auth-1","data":{"token":"your-session-token"}}
```

成功返回 `type:ready`，`data` 包含 `connectionId/userId/heartbeatSeconds`，请求号原样返回。无效或撤销的 Token 关闭连接；订阅服务未就绪时握手返回 503。一个账户允许建立多个使用同一有效 Token 的连接。

不通过 URL 传 Token，也不需要设备身份头。浏览器 Origin 默认必须与长连接地址同源；可以通过 `WS_ALLOWED_ORIGINS` 指定允许的来源。

## 心跳

客户端建议每 20 秒发送：

```json
{"version":2,"type":"ping","requestId":"ping-1"}
```

响应为 `type:pong`，携带相同请求号。服务端也发送标准 WebSocket Ping，客户端需响应 Pong。60 秒内未收到有效心跳将关闭连接。心跳和每次在线投递重新检查会话撤销状态。

## 推送与恢复

- `type:message`：`data` 是 HTTP 发送响应使用的 `Message`，包含消息 ID、会话 ID、序号、发送者、内容、发送时间和撤回状态。
- `type:event`：`data` 是独立同步事件，包含类型、游标、对象 ID、操作人和发生时间。

发送文字使用 HTTP 消息接口。长连接只接收认证、心跳和服务端推送，不能通过客户端帧指定发送者或接收人。

客户端按消息 ID 去重；事件按事件流和游标去重。好友事件与会话事件使用独立游标。网络中断或慢连接关闭后，重新认证、读取会话状态、补拉消息与两类事件，处理完成再确认事件游标。

每个连接有容量 64 的发送队列和唯一 writer。队列满时关闭慢连接，已持久化消息和事件可通过离线同步恢复。在线广播至少一次，允许重复，不承诺长连接恰好一次或跨消息与事件流的严格顺序。

可参考 `test/e2e/im_test.go` 的认证、心跳、消息接收及事件补拉测试。
