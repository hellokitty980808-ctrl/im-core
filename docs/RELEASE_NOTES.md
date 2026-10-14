# 首版源码说明

源码仓库：[hellokitty980808-ctrl/im-core](https://github.com/hellokitty980808-ctrl/im-core)。本文件描述首次源码发布的功能范围；版本标签和 GitHub Release 单独管理。

## IM Core

IM Core 是一个基于 Go / Kratos 的基础即时通讯后端，提供四服务实现、v2 API、JSON WebSocket 协议和本地 Docker Compose 环境。

本版包括：

- 账户注册、密码登录、资料修改、密码修改、Token 刷新与注销。
- 好友申请、接受、拒绝、取消、备注、删除、拉黑与解除。
- 文字私聊和群聊、发送幂等、历史补拉、已读进度和消息撤回。
- 建群、邀请、开放或审批入群、成员查询、改名、退出、移除与解散。
- 长连接认证、心跳、在线消息与事件投递，以及断线后的离线恢复。
- MySQL 事务持久化、Outbox 重试、Redis 会话和 RabbitMQ 广播。

## 开始使用

在项目根目录运行：

```sh
make init-local
make up
```

HTTP：`http://127.0.0.1:18000`；WebSocket：`ws://127.0.0.1:16200/v2/ws`。接入步骤、协议和测试命令见项目 README。

## 验证与边界

已通过本地单元测试、静态检查、编译、镜像构建、相关并发检查、空库四服务 E2E、真实 MySQL 事件排序回归和 RabbitMQ 停机恢复测试。

本版仅包含后端与文字消息，不提供前端、文件上传或群主转让。尚未验证生产部署、多实例、压力容量、真实客户端 UI 或 HTTPS/WSS 部署。

## 许可与分发

项目采用 MIT。第三方依赖保留各自的许可与 NOTICE。首版发布源码，不附应用二进制或容器镜像。
