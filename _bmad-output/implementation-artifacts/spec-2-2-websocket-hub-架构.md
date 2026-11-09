---
title: 'Story 2.2: WebSocket Hub 架构'
type: 'refactor'
created: '2026-11-09'
status: 'review'
route: 'oneshot'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `internal/ws/hub.go` 使用单个 Hub 管理所有频道（`channels map[string]map[*Client]bool`），违反 AD-3 "per-channel Hub goroutine" 要求；`sendHistory` 使用 channelSlug 构建 Redis key，但 `PersistMessage` 使用 channelID，导致缓存命中率为零；`sendHistory` 使用 `LRange(0, -1)` 无上限读取历史。

**Approach:**
1. 重构 `hub.go`：Hub 变为单频道（`clients map[*Client]bool` + `channelSlug/channelID` 字段），移除 `Envelope`，`BroadcastJSON` 去掉 `channelSlug` 参数。
2. 新建 `internal/ws/registry.go`：`HubRegistry`（`sync.Map[slug→*Hub]`，`GetOrCreate` 懒惰创建）。
3. 修复 Redis key 不一致：统一使用 channelID（`chatsphere:channel:%d:messages`）。
4. 修复 `sendHistory`：`LRange(-20, -1)`，只返回最近 20 条。
5. 更新 `client.go`：`BroadcastJSON` 调用去掉 channelSlug 参数。
6. 更新 `handler/chat.go`：`wsHandler` 接受 `*ws.HubRegistry`，`GetOrCreate` 获取频道 Hub。
7. 更新 `main.go`：用 `HubRegistry` 替代单个 Hub。

`client.go` readPump/writePump/心跳（60s+54s）/缓冲256 本次不改（已符合要求）。

</frozen-after-approval>

## Implementation Notes

### Code Map

| 文件 | 操作 | 要点 |
|------|------|------|
| `internal/ws/hub.go` | 重写 | 单频道 Hub，统一 channelID Redis key，sendHistory 限20条 |
| `internal/ws/registry.go` | 新建 | HubRegistry，sync.Map，GetOrCreate 懒创建 |
| `internal/ws/client.go` | 修改 | BroadcastJSON 去掉 channelSlug 参数 |
| `internal/handler/chat.go` | 修改 | wsHandler 改用 HubRegistry |
| `cmd/chatsphere/main.go` | 修改 | HubRegistry 替代 NewHub |

### Hub 单频道设计

```go
type Hub struct {
    mu      sync.RWMutex
    clients map[*Client]bool

    channelSlug string
    channelID   uint64

    broadcast  chan []byte       // 不再是 chan *Envelope
    register   chan *Client
    unregister chan *Client

    db    *gorm.DB
    redis *redis.Client
}
```

### HubRegistry（`internal/ws/registry.go`）

```go
type HubRegistry struct {
    hubs sync.Map // slug → *Hub
    db   *gorm.DB
    rdb  *redis.Client
}

func (r *HubRegistry) GetOrCreate(slug string, channelID uint64) *Hub {
    if v, ok := r.hubs.Load(slug); ok {
        return v.(*Hub)
    }
    hub := newHub(slug, channelID, r.db, r.rdb)
    if actual, loaded := r.hubs.LoadOrStore(slug, hub); loaded {
        return actual.(*Hub)
    }
    go hub.Run()
    return hub
}
```

### Redis key 修复

| 位置 | 旧 | 新 |
|------|----|----|
| `sendHistory` | `fmt.Sprintf(redisChannelKey, c.channelSlug)` | `fmt.Sprintf(redisChannelKey, h.channelID)` |
| `PersistMessage` | `fmt.Sprintf(redisChannelKey, msg.ChannelID)` | `fmt.Sprintf(redisChannelKey, h.channelID)` |
| 常量格式字串 | `"chatsphere:channel:%s:messages"` | `"chatsphere:channel:%d:messages"` |

### sendHistory 限制

```go
raw, err := h.redis.LRange(ctx, key, -historyLimit, -1).Result()  // historyLimit = 20
```

### Verification

```bash
cd /Users/kevin/Documents/projects/chatsphere && go build ./...
```

预期：无输出（零错误）。✅ 已验证通过。

## Review Triage Log

Blind Hunter 发现 3 个问题：

| # | 描述 | 处置 |
|---|------|------|
| F1 | `broadcastToAll` 向 unregister 通道阻塞写入（容量64），Run() goroutine 自身执行中无法消费，>64 个慢客户端时死锁 | ✅ 已修复：改为在 broadcastToAll 内直接同步清理（`delete(h.clients, c); close(c.send)`），彻底消除对 unregister 通道的依赖 |
| F2 | Hub goroutine 永不停止：频道零客户端后 Hub 仍在 registry 中运行，注册表无限增长 | 延期 — Hub 生命周期管理属于 Story 2.4+ 范围（需结合 Redis 在线计数 SADD/SREM/SCARD 一并设计） |
| F3 | `redisOnlineKey` 常量定义但从未使用（在线人数仍为内存计数） | ✅ 已修复：删除未使用常量 |
