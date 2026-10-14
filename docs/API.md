# v2 HTTP 接口

公开请求与响应以 `api/im/v2/im.proto` 和 `openapi.yaml` 为准。HTTP 成功返回 JSON 对象，错误示例：

```json
{"error":{"code":"PERMISSION_DENIED","message":"无权执行此操作"}}
```

ID、序号和游标使用 JSON 字符串，时间为 Unix 毫秒。除验证码、注册和登录外，全部接口传 `Authorization: Bearer <token>`。客户端不能通过身份头或请求中的用户 ID 指定操作人。

## 账户与会话

| 方法 | 路径 | 用途 |
|---|---|---|
| GET | `/v2/captcha` | 一次性图片验证码，只返回 ID 和图片 |
| POST | `/v2/accounts` | 注册并登录 |
| POST | `/v2/sessions` | 密码登录 |
| POST | `/v2/sessions/refresh` | 替换当前会话 |
| DELETE | `/v2/sessions/current` | 注销当前会话 |
| GET / PATCH | `/v2/users/me` | 当前账户资料、修改资料 |
| PATCH | `/v2/users/me/password` | 修改密码并撤销会话 |
| GET | `/v2/users?account=alice` | 按账号精确搜索 |

注册请求：

```json
{"account":"alice_01","password":"your-local-password","nickname":"Alice","captchaId":"captcha-id","captchaCode":"user-input"}
```

账号规范化为小写，只允许 3–32 位 ASCII 字母、数字和下划线；密码为 8–72 字节；昵称为 2–32 个字符。登录使用相同的账号、密码和验证码字段。资料修改使用可选的 `nickname/avatarUrl/bio`，未提供的字段不修改；简介最多 200 个字符，头像 URL 最多 512 字节。

密码修改使用 `currentPassword/newPassword`。登录和刷新均只保留一个有效会话；两个并发刷新只有一个成功。Token 过期或被撤销返回 401。

## 好友

| 方法 | 路径 | 用途 |
|---|---|---|
| POST / GET | `/v2/friend-requests` | 发送申请、查询双方相关的申请 |
| POST | `/v2/friend-requests/{requestId}/accept` | 接受，仅接收人可操作 |
| POST | `/v2/friend-requests/{requestId}/reject` | 拒绝，仅接收人可操作 |
| POST | `/v2/friend-requests/{requestId}/cancel` | 取消，仅发送人可操作 |
| GET | `/v2/friends` | 当前好友列表 |
| GET / PATCH / DELETE | `/v2/friends/{userId}` | 查看关系、修改备注、删除关系 |
| POST / DELETE | `/v2/blocks/{userId}` | 拉黑、解除 |
| GET | `/v2/blocks` | 黑名单 |
| GET | `/v2/friend-events` | 好友变化事件补拉 |
| POST | `/v2/friend-events/ack` | 确认已处理游标 |

申请传 `userId/note`。重复待处理申请返回同一申请；状态为 `PENDING/ACCEPTED/REJECTED/CANCELLED`。接受返回 `conversationId`。恢复好友复用双方原私聊，唯一键防止重复会话。

好友备注传 `remark`，最长 32 个字符。关系为 `NONE/FRIEND/BLOCKED`；删除是操作人一侧的关系变更。双方都保持好友且任一方未拉黑时，才允许私聊发送。拉黑不删除好友快照，解除后仍是好友的双方可继续发送。

列表使用 `cursor/limit`，默认 30 条，最多 100 条，返回 `nextCursor`；好友列表按目标用户 ID 排序，申请列表按申请 ID 排序。好友事件通知客户端重新读取关系或申请状态。

## 会话与文字消息

| 方法 | 路径 | 用途 |
|---|---|---|
| GET | `/v2/conversations` | 自己的会话，包含离开和解散状态 |
| GET | `/v2/conversations/{conversationId}` | 会话信息 |
| POST / GET | `/v2/conversations/{conversationId}/messages` | 发送、补拉文字消息 |
| POST | `/v2/conversations/{conversationId}/messages/{messageId}/recall` | 发送者撤回 |
| PATCH / GET | `/v2/conversations/{conversationId}/read-progress` | 更新、读取已读进度 |
| GET | `/v2/conversation-events` | 会话事件补拉 |
| POST | `/v2/conversation-events/ack` | 确认事件游标 |

会话类型为 `DIRECT/GROUP`，状态为 `ACTIVE/LEFT/CLOSED`。群资料的 `groupId` 与消息归属的 `conversationId` 是不同概念，不能混用。发送请求不接收会话类型或发送者身份：

```json
{"content":"你好","idempotencyKey":"client-generated-stable-key"}
```

文字最大 2048 字节，不能全为空白。发送键为 1–64 个可见 ASCII 字符，每个发送账户下唯一。同一键且相同会话和内容返回首个成功响应；不同载荷返回 409。

历史使用 `afterSeq/limit`，按 `seq` 升序返回，`nextSeq` 用于后续补拉。撤回保留序号占位，返回 `recalled:true` 和空内容，并产生独立事件。非成员、已离开成员不能读取或发送。

已读更新传 `seq`，不能超过会话最新消息序号，也不会回退。文字序号不与事件游标混用。

## 群组

| 方法 | 路径 | 用途 |
|---|---|---|
| POST | `/v2/groups` | 建群 |
| GET / PATCH / DELETE | `/v2/groups/{groupId}` | 群资料、群主改名、群主解散 |
| GET / POST | `/v2/groups/{groupId}/members` | 成员列表、邀请好友 |
| DELETE | `/v2/groups/{groupId}/members/{userId}` | 群主移除成员 |
| POST | `/v2/groups/{groupId}/leave` | 普通成员退出 |
| POST / GET | `/v2/groups/{groupId}/join-requests` | 申请加入、群主查看申请 |
| POST | `/v2/groups/{groupId}/join-requests/{requestId}/accept` | 群主接受 |
| POST | `/v2/groups/{groupId}/join-requests/{requestId}/reject` | 群主拒绝 |

建群传 `name/memberUserIds/requireApproval`，群主自动加入；创建及单次邀请最多指定 100 人，目标必须与邀请人保持可通信的好友关系。重复用户 ID 不会产生重复成员。邀请传 `userIds`。群名为 1–32 个字符。

开放群申请立即返回 `ACCEPTED`；审批群返回 `PENDING`，审批前不能发送或读取群消息。群成员角色只包括 `MEMBER/OWNER`。群主不能直接退出或移除自己，需解散群；普通成员不能改名、审批、移除其他人或解散。

退出、移除和解散事务会持久化成员状态及事件。新加入或重新加入的成员只能读取本次加入后的消息，历史可见起点由服务端保存。

## 离线同步

好友事件和会话事件拥有各自的游标。事件类型只有 `FRIEND_CHANGED/GROUP_MEMBER_CHANGED/GROUP_CLOSED/MESSAGE_RECALLED`；每条事件包含对应对象 ID、操作人、发生时间及变化状态。

客户端重连后读取会话列表，补拉当前活跃会话的新消息，分别补拉两类事件。事件处理完成后传 `cursor` 确认；确认不会回退，不能超过自己的事件范围。确认状态不删除事件，补拉起点仍由请求游标指定。

同一用户的会话事件串行分配游标并提交，避免跨会话事务反序提交造成遗漏。不同用户的事件游标可以有间隙。在线投递和补拉可能重复，客户端必须按事件流与游标去重。
