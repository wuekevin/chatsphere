---
title: 'Story 1.1: 项目基础启动 & 配置检测'
type: 'feature'
created: '2026-11-09'
status: 'done'
route: 'oneshot'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** 项目启动时需要自动区分"安装模式"与"正常运行模式"，但当前实现缺少安装模式下对非 /setup 路径的 302 重定向，以及正常模式启动时的 slog 日志记录（AC2、AC3 未完全满足）。

**Approach:** 在 installer.Run() 中添加 catch-all 路由对非 /setup 路径返回 302；在 main.go 进入正常模式时用 slog.Info 记录 "starting in normal mode"。其余 AC（配置检测、连接失败退出、模式互斥、通过 config 包判断）已实现。

</frozen-after-approval>

## Implementation Notes

**现有实现评估：**
- 大部分已实现；仅两处 gap：AC2 缺少 catch-all 302，AC3 缺少 slog 日志

**已修改文件：**
- `internal/installer/installer.go` — 在现有路由后添加 `r.NoRoute()` 重定向到 `/setup`（AC2）
- `cmd/chatsphere/main.go` — 添加 `log/slog` import，在 `else` 分支（config.yaml 已存在路径）添加 `slog.Info("starting in normal mode")`（AC3）

**决策：** `slog.Info` 放在 else 分支而非 if 块后，确保仅在 config.yaml 预先存在时触发，不混淆新安装流程日志。

**预存在问题（非本 Story 范围）：**
- `internal/ws/hub.go:140`: `fmt.Sprintf %s` 对 `uint64` — go vet 警告，Story 2.2 范围
- `installHandler` 中 `db.Create(ch)` 错误未处理 — Story 1.2/1.4 范围
- `generateSecret()` 中 `rand.Read` 错误忽略 — 技术债，后续修复

## Review Triage Log

- finding: Mixed log package usage (`log` + `log/slog` 混用) | verdict: `low` → rejected | evidence: AC3 明确要求 slog.Info；其余 log.* 调用迁移超出本 Story 范围；日志格式不一致不影响用户
- finding: `slog.Info` fires after fresh installation | verdict: `medium` → patched | fix: 移入 `else` 分支，仅在 config 已存在时触发
- finding: `r.GET("/", ...)` redundant after NoRoute | verdict: `low` → rejected | evidence: 两者行为相同；保留明确路由不影响正确性，不引入混淆
- finding: `db.Create(ch)` error silently swallowed | verdict: `defer` | evidence: 预存在 bug，本次变更未引入；来源 Story 1.4 范围
- finding: `rand.Read` error ignored | verdict: `defer` | evidence: 预存在 bug，本次变更未引入
