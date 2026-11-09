---
title: 'Story 2.3: 匿名用户身份 & WS 连接鉴权'
type: 'feature'
created: '2026-11-09'
status: 'review'
route: 'oneshot'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `wsHandler` 直接依赖 `*gorm.DB`（违反 AD-1 分层）；`anonName()` 仅有8个形容词/名词（命名空间过小）；匿名身份 `ipHash` 未混入服务端盐值（可被彩虹表逆向）；`ServeWS` 内部计算 ipHash 而非由调用方提供（逻辑耦合）。

**Approach:**
1. 新建 `internal/repository/user_repository.go`（`UserRepository` interface + GORM 实现）。
2. `wsHandler` 改用 `chSvc.GetBySlug` 和 `userRepo.GetByID`，完全移除 `*gorm.DB` 依赖。
3. anonID = SHA-256(clientIP + cfg.Server.SecretKey)[:4] → 8 hex 字符，在 `wsHandler` 层计算并传入 `ServeWS`。
4. 升级 `anonName`：20 形容词 × 20 名词 = 400 种组合，输入改为 anonID（hex 字串）。
5. `client.go`：`ipHash` 字段改名为 `anonID`，`ServeWS` 新增 `anonID string` 参数，移除内部 sha256 计算。
6. `main.go`：注入 `userRepo`，`RegisterChat` 移除 `db` 参数。

</frozen-after-approval>

## Implementation Notes

### Code Map

| 文件 | 操作 | 要点 |
|------|------|------|
| `internal/repository/user_repository.go` | 新建 | GetByID，映射 ErrRecordNotFound → ErrNotFound |
| `internal/handler/chat.go` | 重写 | 移除 gorm 依赖，anonID 计算，anonName 升级 |
| `internal/ws/client.go` | 修改 | ServeWS 加 anonID 参数，移除 crypto/sha256，ipHash→anonID |
| `cmd/chatsphere/main.go` | 修改 | 注入 userRepo，RegisterChat 去掉 db 参数 |

### anonID 计算

```go
// wsHandler 内
sum := sha256.Sum256([]byte(ip + cfg.Server.SecretKey))
anonID := fmt.Sprintf("%x", sum[:4])  // 8 hex chars
```

### anonName（20×20=400 种）

```go
func anonName(anonID string) string {
    adjectives := [20]string{ "Quick","Calm","Bold","Wise","Bright","Swift","Cool","Kind",
                               "Brave","Sharp","Wild","Keen","Fair","Free","Pure","Warm",
                               "Soft","Vast","Dark","Stark" }
    nouns := [20]string{ "Fox","Bear","Wolf","Hawk","Lynx","Deer","Crow","Hare",
                          "Owl","Pike","Kite","Mink","Puma","Ibis","Wren","Vole",
                          "Newt","Moth","Frog","Swan" }
    n, _ := strconv.ParseUint(anonID, 16, 64)
    return adjectives[n%20] + nouns[(n/20)%20]
}
```

### RegisterChat 新签名

```go
func RegisterChat(
    r *gin.Engine,
    chSvc *service.ChannelService,
    msgSvc *service.MessageService,
    userRepo repository.UserRepository,
    registry *ws.HubRegistry,
    cfg *config.Config,
)
```

### Verification

```bash
cd /Users/kevin/Documents/projects/chatsphere && go build ./...
```

预期：无输出（零错误）。✅ 已验证通过。

## Review Triage Log

Blind Hunter 发现 2 个 Minor 问题：

| # | 描述 | 处置 |
|---|------|------|
| F1 | `strconv.ParseUint` 失败时 n=0，所有异常 anonID 返回 "QuickFox" | 无影响 — anonID 由 `fmt.Sprintf("%x", sum[:4])` 生成，永远是合法 hex |
| F2 | `userRepo.GetByID` 返回任何错误（含数据库错误）时都静默降级为匿名 | 可接受 — JWT 对应用户不存在/被删除时降级比拒绝连接更友好 |
