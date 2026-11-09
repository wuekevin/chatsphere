---
title: 'Story 2.4: 消息发送核心流程'
type: 'feature'
created: '2026-11-09'
status: 'review'
route: 'oneshot'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** ① `PersistMessage` 的 Redis 失败会返回错误导致消息不广播（违反"Redis 失败不回滚"约束）；② 在线计数使用内存 `len(h.clients)`（违反 Redis SADD/SREM/SCARD 要求）；③ 消息内容无 XSS 防护；④ 在线计数事件类型为 `"online"` 而非 `"online_count"`；⑤ `broadcastToAll` 直接移除慢客户端但未更新在线计数。

**Approach:**
1. `PersistMessage` 中 Redis 失败改为 `log.Printf` 后继续（返回 nil），只有 MySQL 失败才返回 error。
2. 新增 `updateOnlineCount(c *Client, joining bool)` — 在 goroutine 中执行 Redis SADD/SREM/SCARD，然后广播 `{"type":"online_count","count":N}`。多标签去重：SREM 前检查 `h.clients` 中是否还有同 `clientMember` 的其他客户端。
3. 新增 `clientMember(c *Client) string` helper：认证用户 → `u<userID>`，匿名 → `a<anonID>`。
4. `addClient`/`removeClient`/`broadcastToAll` 调用 `go h.updateOnlineCount(c, joining)` 替换原 `BroadcastJSON(Event{Type:"online",...})`。
5. `client.go` `handleSend`：先检查长度约束，再 `html.EscapeString(content)`。

</frozen-after-approval>

## Implementation Notes

### Code Map

| 文件 | 操作 | 要点 |
|------|------|------|
| `internal/ws/hub.go` | 修改 | Redis 在线计数，PersistMessage Redis 非致命，online_count 事件 |
| `internal/ws/client.go` | 修改 | html.EscapeString，长度检查顺序 |

### 写入顺序（严格）

```
① db.Create(msg) — 失败: return error，调用方中止，不广播
② Redis RPUSH+LTRIM+EXPIRE — 失败: log 并继续（return nil）
③ Hub.BroadcastJSON(Event{Type:"message",...}) — 步骤②成功与否都执行
```

### 在线计数

```go
// clientMember: 多标签同用户/同 IP 只计一个
func clientMember(c *Client) string {
    if c.userID != nil { return fmt.Sprintf("u%d", *c.userID) }
    return "a" + c.anonID
}

func (h *Hub) updateOnlineCount(c *Client, joining bool) {
    ctx, cancel := context.WithTimeout(ctx, 2s)
    // SAdd or SRem（带多标签去重检查）
    // SCard → n；fallback 到内存 count
    h.BroadcastJSON(Event{Type: "online_count", Count: &n})
}
```

### XSS 防护顺序

```go
// 先限长（escape 会膨胀 & → &amp;），再 escape
if utf8.RuneCountInString(content) > maxLen { return error }
content = html.EscapeString(content)
```

### Verification

```bash
cd /Users/kevin/Documents/projects/chatsphere && go build ./...
```

预期：无输出（零错误）。✅ 已验证通过。

## Review Triage Log

Blind Hunter 发现 2 个问题：

| # | 描述 | 处置 |
|---|------|------|
| F1 | `updateOnlineCount` goroutine 内 `BroadcastJSON` 向 broadcast 通道（容量512）阻塞写；通道满时 goroutine 阻塞 | 可接受 — 不是死锁（Run() 持续消费），只是短暂延迟；512 缓冲在正常负载下足够 |
| F2 | `html.EscapeString` 在长度检查前执行，`&` → `&amp;` 膨胀导致合法消息误报超长 | ✅ 已修复：先检查长度再 EscapeString |
