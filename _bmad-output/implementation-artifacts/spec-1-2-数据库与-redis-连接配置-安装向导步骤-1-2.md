---
title: 'Story 1.2: 数据库与 Redis 连接配置（安装向导步骤 1-2）'
type: 'feature'
created: '2026-11-09'
status: 'done'
route: 'oneshot'
review_loop_iteration: 0
context: []
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** 安装向导 HTML 使用 `r.Static` 文件系统方式提供服务（不满足 AD-7 go:embed 要求），设计 token 仍为绿色主题（不符合 UX 规范的琥珀暗色主题），API 路径带 `/api/` 前缀（AC4 要求 `/setup/test-db`），且"下一步"按钮未在测试通过前锁定（AC5）。

**Approach:** 创建 `web/assets.go` 嵌入 installer 目录；在 `installer.go` 改用 `ServeFS` 并调整路由路径；全面重写 HTML CSS token 和行为逻辑满足全部 7 条 AC。

</frozen-after-approval>

## Implementation Notes

**修改文件：**
- `web/assets.go`（新建）— `//go:embed installer` + 导出 `InstallerFS`
- `internal/installer/installer.go` — 添加 `io/fs`、`github.com/wuekevin/chatsphere/web` 导入；`Run()` 中改用 `fs.Sub(web.InstallerFS, "installer")` + `r.StaticFS`；路由 `/api/setup/test-db` → `/setup/test-db`，同理 test-redis；`/api/setup/install` 保留（Story 1.4 重构）
- `web/installer/index.html` — 全量重写 CSS token 至暗色 amber 主题（`--accent: #f5a623`，light 变体），step active 使用 `--accent` + `box-shadow: 0 0 12px var(--accent-glow)`，DB user 默认 root，Continue 按钮初始 disabled，测试通过后启用，focus/error 使用 `var(--accent-dim)` / `var(--red)`，fetch 路径与后端路由对齐

**决策：**
- `err :=` 重用问题：`fs.Sub` 已声明 `err`，`err = <-done` 改为赋值而非短声明
- `/api/setup/install` 未更名：Story 1.4 将改为 SSE 端点，提前变更无意义
- alert() 在步骤 3-4 验证中保留：本 Story 范围仅步骤 1-2；Story 1.3 负责步骤 3-4 inline error

**Review 后 patch：**
- `goStep()` 中 `circle.textContent = i < n ? '✓' : i`，满足 UX-DR16 done 步骤 ✓ 要求
- DB/Redis 所有输入字段加 `oninput` 重置对应 Continue 按钮为 disabled，防止测试通过后改字段绕过（AC5）

## Review Triage Log

- finding: done 步骤显示数字而非 ✓（UX-DR16 要求黑色 ✓） | verdict: `medium` → patched | fix: `goStep()` 改用 `circle.textContent = i < n ? '✓' : i`
- finding: 测试通过后编辑字段不重置 Continue 按钮 | verdict: `medium` → patched | fix: 9 个输入字段加 `oninput` 重置 btn-step2/btn-step3 为 disabled
- finding: 日志框背景 `#0a0d14` 硬编码 | verdict: `low` → rejected | evidence: 终端暗色框为惯例，纯装饰，不影响功能
- finding: 端口清空后发送 0 | verdict: `low` → rejected | evidence: 默认值已预填，向导中极少发生；驱动报错可见
- finding: clearErrors 在测试结果前调用可能误清字段样式 | verdict: `low` → rejected | evidence: badge 仍显示错误文字，视觉反馈充足
