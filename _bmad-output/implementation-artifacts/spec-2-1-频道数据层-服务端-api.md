---
title: 'Story 2.1: 频道数据层 & 服务端 API'
type: 'refactor'
created: '2026-11-09'
status: 'done'
route: 'oneshot'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `internal/handler/chat.go` 直接依赖 `*gorm.DB`，违反架构 AD-1/AD-2 分层约束；`internal/repository/` 包不存在、`internal/service/` 包为空，导致 Story 2.2–2.4 无法依赖 ChannelRepository/MessageRepository interface。

**Approach:** 新建 `internal/repository/` 包，定义 `ChannelRepository` 和 `MessageRepository` Go interface 并提供各自的 GORM 实现；在 `internal/service/` 创建 `ChannelService`（依赖 `ChannelRepository`）和 `MessageService`（依赖 `MessageRepository`）；重构 `internal/handler/chat.go` 中的 `listChannelsHandler` / `getMessagesHandler` 改用 `ChannelService`；更新 `cmd/chatsphere/main.go` 注入依赖。`wsHandler` 和 `anonName()` 本次不改动（属于 Story 2.2/2.3 范围）。

</frozen-after-approval>

## Implementation Notes

### Code Map

| 文件 | 现状 | 本次操作 |
|------|------|----------|
| `internal/model/channel.go` | 已有 Channel 结构体（含 Slug unique） | 只读，不改 |
| `internal/model/message.go` | 已有 Message 结构体（UserID nullable，IPHash，IsDeleted） | 只读，不改 |
| `internal/handler/chat.go` | `listChannelsHandler`/`getMessagesHandler` 直接用 `*gorm.DB`；`wsHandler`/`anonName()` 保持不变 | 修改：两个 handler 改依赖 ChannelService |
| `cmd/chatsphere/main.go` | `handler.RegisterChat(r, db, rdb, hub, cfg)` | 修改：在调用前初始化 repo 和 service，传入 ChannelService |
| `internal/repository/channel_repository.go` | 不存在 | 新建 |
| `internal/repository/message_repository.go` | 不存在 | 新建 |
| `internal/service/channel_service.go` | 不存在 | 新建 |
| `internal/service/message_service.go` | 不存在 | 新建 |

### ChannelRepository interface（`internal/repository/channel_repository.go`）

```go
type ChannelRepository interface {
    ListActive() ([]model.Channel, error)
    GetBySlug(slug string) (*model.Channel, error)
}
```

GORM 实现：`ListActive` → `WHERE is_active = true ORDER BY is_default DESC, name ASC`；`GetBySlug` → `WHERE slug = ? AND is_active = true`。

### MessageRepository interface（`internal/repository/message_repository.go`）

```go
type MessageRepository interface {
    ListByChannel(channelID uint64, limit int) ([]model.Message, error)
    Create(msg *model.Message) error
}
```

GORM 实现：`ListByChannel` → `WHERE channel_id = ? AND is_deleted = false ORDER BY created_at DESC LIMIT ?`；`Create` → `db.Create`。

### ChannelService（`internal/service/channel_service.go`）

```go
type ChannelService struct {
    repo repository.ChannelRepository
}
func NewChannelService(repo repository.ChannelRepository) *ChannelService
func (s *ChannelService) ListActive() ([]model.Channel, error)
func (s *ChannelService) GetBySlug(slug string) (*model.Channel, error)
```

### MessageService（`internal/service/message_service.go`）

```go
type MessageService struct {
    repo repository.MessageRepository
}
func NewMessageService(repo repository.MessageRepository) *MessageService
func (s *MessageService) ListByChannel(channelID uint64, limit int) ([]model.Message, error)
func (s *MessageService) Create(msg *model.Message) error
```

### handler/chat.go 重构要点

- `RegisterChat` 签名改为接收 `*service.ChannelService`（和 `*service.MessageService`，可选但建议一并传入备 2.4 扩展）替代 `*gorm.DB`；保留 `*redis.Client`、`*ws.Hub`、`*config.Config` 参数不变
- `listChannelsHandler` 改用 `chSvc.ListActive()`
- `getMessagesHandler` 改用 `chSvc.GetBySlug(slug)` + `msgSvc.ListByChannel(channel.ID, 100)`
- `wsHandler` 和 `anonName()` 完全不变

### main.go 注入顺序

```go
chRepo := repository.NewChannelRepository(db)
msgRepo := repository.NewMessageRepository(db)
chSvc := service.NewChannelService(chRepo)
msgSvc := service.NewMessageService(msgRepo)
handler.RegisterChat(r, chSvc, msgSvc, rdb, hub, cfg)
```

### Verification

```bash
cd /Users/kevin/Documents/projects/chatsphere && go build ./...
```

预期：无输出（零错误）。✅ 已验证通过。

## Review Triage Log

Blind Hunter 发现 6 个问题：

| # | 描述 | 处置 |
|---|------|------|
| F1 | `rdb *redis.Client` 在 `RegisterChat` 中未使用（冗余参数） | ✅ 已修复：从签名移除 rdb |
| F2 | handler 直接依赖 `gorm.ErrRecordNotFound`，破坏抽象边界 | ✅ 已修复：引入 `repository.ErrNotFound` sentinel，GetBySlug 映射并由 handler 检查 |
| F3 | ChannelService/MessageService 以 concrete type 注入，阻碍测试 | 延期 — Story 2.2 需要时补 service interface |
| F4 | `ListByChannel` 返回 DESC 顺序无文档，Story 2.6 可能误解 | ✅ 已修复：接口注释明确说明 DESC 顺序及调用方须自行反转 |
| F5 | repository 返回原始 GORM 错误，not-found 以外类型不可区分 | 延期 — 需系统性 error domain 设计 |
| F6 | wsHandler 重复 `is_active = true` 过滤，与 GetBySlug 未来可能发散 | ✅ 已修复：GetBySlug 接口注释说明过滤语义及与 wsHandler 的关系 |
