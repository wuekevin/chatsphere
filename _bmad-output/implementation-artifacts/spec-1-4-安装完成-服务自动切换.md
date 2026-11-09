---
title: 'Story 1.4: 安装完成 & 服务自动切换'
type: 'feature'
created: '2026-11-09'
status: 'review'
route: 'oneshot'
review_loop_iteration: 1
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** 安装向导最终步骤（执行安装）仍使用同步 POST+JSON，无法实时推送进度日志；缺少 `DirectMessage` 和 `ReputationTier` 模型骨架导致 AutoMigrate 不完整；无任何部署文档（Docker、systemd、Nginx、宝塔）。

**Approach:** 将 `/api/setup/install` 改为 SSE 流式响应（`text/event-stream`，POST + `c.Stream()`），后端逐步推送 JSON 事件；前端改用 `fetch` + `ReadableStream` 消费事件；创建两个缺失的模型骨架文件并加入 AutoMigrate；交付 Dockerfile、systemd unit、Nginx 配置和部署指南。

</frozen-after-approval>

## Implementation Notes

### 文件变更

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `internal/installer/installer.go` | 修改 | `installHandler` 改为 SSE；新增 `runInstall` goroutine-channel 模式；`runMigrations` 增加 2 个模型；`fail` helper 确保所有错误路径都 signal `done`；`errors.Is` 替代 `==` 比较 ErrRecordNotFound |
| `internal/model/direct_message.go` | 新建 | DirectMessage 模型骨架 |
| `internal/model/reputation_tier.go` | 新建 | ReputationTier 模型骨架 |
| `web/installer/index.html` | 修改 | `startInstall()` 改为 SSE ReadableStream 消费；HTTP 错误检查；admin-link 指向 `/admin` |
| `deploy/Dockerfile` | 新建 | 多阶段 Go 构建 |
| `deploy/chatsphere.service` | 新建 | systemd unit |
| `deploy/nginx.conf` | 新建 | WebSocket + SSE 反代配置 |
| `deploy/deploy.md` | 新建 | 宝塔 + 通用部署指南 |
| `.dockerignore` | 新建 | 防止 config.yaml 等机密文件进入镜像 |

### 关键设计决策

- **SSE 架构**: `installHandler` 设置 `text/event-stream` 头，buffered channel(20) 解耦 `runInstall` goroutine 与 `c.Stream()` 写循环
- **done channel**: `runInstall` 在所有路径（成功和失败）都同步写入 `done`，避免 installer HTTP server 永久阻塞
- **进程生命周期**: `installer.Run()` 返回 nil 后，`main()` 自然 fall-through 启动正式服务器，无需 `os.Exit`

## Review Triage Log

Blind Hunter 发现 7 个问题（1 次审查轮）：

| # | 严重度 | 描述 | 处置 |
|---|--------|------|------|
| F1 | high | startInstall 未检查 HTTP 错误状态，非 SSE 响应体被当 SSE 解析 | ✅ 已修复：添加 `if (!res.ok)` 检查 |
| F2 | medium | admin-link href 缺少 `/admin` 后缀 | ✅ 已修复（实际已正确，grep 确认） |
| F3 | low | 进度条步进 17 × 6 = 107，第 4 步后卡在 90% | 延期 — 视觉问题，不影响功能 |
| F4 | high | runInstall 错误路径 return 不 signal done，installer 永久阻塞 | ✅ 已修复：引入 `fail()` helper 在每个错误路径都写 `done <- err` |
| F5 | medium | done <- nil 在 goroutine 中发送，可能在最后一个 SSE 事件刷出前触发 shutdown | ✅ 已修复：改为同步 `done <- nil`，buffered channel 保证不阻塞 |
| F6 | medium | GORM v2 中 `== gorm.ErrRecordNotFound` 不匹配 wrapped error | ✅ 已修复：改用 `errors.Is(...)` |
| F7 | medium | Dockerfile `COPY . .` 会将 config.yaml 等机密文件打入镜像 | ✅ 已修复：创建 `.dockerignore` |

