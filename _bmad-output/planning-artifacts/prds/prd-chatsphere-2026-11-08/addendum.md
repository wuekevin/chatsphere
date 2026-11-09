---
title: "ChatSphere PRD Addendum"
updated: 2026-11-08
---

# ChatSphere PRD Addendum

## 技术栈决策（不进入 PRD 正文）

以下技术选型已在原型阶段确定，作为实现约束记录于此：

- **语言/框架**：Go 1.21+，Gin HTTP 框架，GORM ORM
- **数据库**：MySQL 8.0 / TiDB（GORM mysql driver，兼容 TiDB）
- **缓存**：Redis 7 / Upstash（go-redis/v9，兼容 Upstash REST 模式）
- **WebSocket**：gorilla/websocket
- **认证**：golang-jwt/jwt/v5，Bearer Token，30 天有效期
- **配置**：gopkg.in/yaml.v3，config.yaml，`go:embed` 打包静态资源
- **部署**：单二进制 + Docker Compose（app + mysql:8.0 + redis:7-alpine）

## 消息持久化策略（细节）

- Redis key 格式：`chatsphere:channel:{slug}:messages`（List，RPUSH + LTRIM 维持最多500条）
- Redis TTL：24 小时滚动续期（每次写入时 EXPIRE 重置）
- MySQL messages 表：软删除（`is_deleted` 字段），不物理删除，管理员仍可查看
- 新用户连接时：从 Redis 读取最近历史推送给客户端（LRANGE 0 -1）

## 在线用户计数策略

- Redis key 格式：`chatsphere:channel:{slug}:online`（Set）
- 用户连接时 SADD，断开时 SREM；心跳检测（54s ping）维持连接活性
- SCARD 获取实时在线数，广播给该频道所有客户端

## 速率限制实现

- 在 WebSocket Client 层维护 `lastSentAt` 时间戳
- 每次 send 消息前检查距上次发言是否超过 `channel.RateLimitSeconds`
- 超限时直接向该客户端返回错误消息，消息不广播
- 此为服务端强制执行，客户端无法绕过

## SaaS 多租户扩展预留（v2 参考）

v1 单租户架构中，所有数据共享同一 MySQL 数据库和 Redis 实例。  
v2 SaaS 扩展路径：
- 在 config.yaml 中增加 `tenant_id` 概念
- MySQL 表增加 `tenant_id` 分区键
- Redis key 前缀增加租户 ID：`chatsphere:{tenant_id}:channel:{slug}:messages`
- 安装向导改造为租户注册流程
