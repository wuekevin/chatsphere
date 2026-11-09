# Epic 1 Context: 安装向导 & 系统基础

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

让部署工程师能够通过浏览器安装向导在 30 分钟内完成 ChatSphere 的完整安装：分步填写并即时测试数据库/Redis 连接信息、创建管理员账号、设置站点基本信息；安装完成后服务自动切换到正常运行模式，无需手动编辑任何配置文件或重启命令。Epic 1 的四个 Story 共同搭建项目的 Go 骨架、安装向导 HTTP 服务、配置文件读写逻辑，以及安装完成后的进程重启机制——这是后续所有 Epic 的运行基础。

## Stories

- Story 1.1: 项目基础启动 & 配置检测
- Story 1.2: 数据库与 Redis 连接配置（安装向导步骤 1-2）
- Story 1.3: 管理员账号与站点信息配置（安装向导步骤 3-4）
- Story 1.4: 安装完成 & 服务自动切换

## Requirements & Constraints

**配置检测行为（FR-4.1, FR-4.5）**
- 启动时检测当前工作目录 config.yaml 是否存在
- 不存在 → 安装模式：仅注册 `/setup` 路由，非 `/setup` 路径 302 重定向；不初始化数据库/Redis
- 存在且连接成功 → 正常模式：全量路由，全部初始化
- 存在但连接失败 → stderr 明确错误（含失败原因），非零退出码退出，不静默失败

**安装向导流程（FR-4.2）**
- 5 步骤：① 数据库配置 ② Redis 配置 ③ 管理员账号 ④ 站点信息 ⑤ 执行安装
- 步骤 1-2 各有独立"测试连接"按钮，连接通过后才启用"下一步"
- 步骤 3-4 数据在前端本地持有，步骤 5 统一提交

**config.yaml 生成（FR-4.3）**
- 包含：数据库连接、Redis 连接、JWT 密钥（`crypto/rand` 32 字节 base64）、安装随机盐（32 字节）、管理后台路径
- 每个配置项附中英文注释

**安装后自动切换（FR-4.4）**
- 安装成功写入 config.yaml 后调用 `os.Exit(0)`
- 依赖进程管理器自动重启（Docker restart:always / systemd Restart=always / 宝塔异常重启）
- 安装日志通过 SSE 实时推送给前端

**部署文档（NFR-4）**
- Story 1.4 须交付：docker-compose.yml、systemd unit、宝塔部署指南、Nginx WebSocket 代理配置

## Technical Decisions

**配置生命周期（AD-6）**
- 安装模式与正常模式互斥，同一进程不同时注册两套路由
- `internal/config` 包封装配置读取，main() 通过该包的函数判断模式，不直接读文件
- `os.Exit(0)` 是终止安装进程的唯一方式；进程管理器负责重启

**包边界（AD-2）**
- 严格层次：`cmd/ → internal/config → internal/installer → internal/database → internal/cache → model/repository/service/handler`
- `internal/` 是隔离边界，不可从外部直接导入

**静态资源嵌入（AD-7）**
- 安装向导 HTML 在 `web/installer/`，通过 `go:embed` 编译入二进制
- 安装模式下 Gin 以 `ServeFS` 提供安装向导页面；正常模式不注册此路由

**错误处理约定（AD-12）**
- HTTP 接口统一返回 `{"error":"...","code":"..."}` + 对应 HTTP 状态码
- 连接测试失败在响应 body 返回 JSON 错误，不返回原始数据库错误堆栈（脱敏）
- 内部错误写 ERROR 级 slog 日志

**GORM 模型骨架（Story 1.4 依赖）**
- Story 1.4 执行 AutoMigrate 时需要 6 个模型文件存在：`internal/model/{user,channel,message,direct_message,report,reputation_tier}.go`
- 实现 Story 1.4 时须同步创建完整字段定义和 GORM tag 的骨架文件（Repository/Service 实现由后续 Story 完成）

**Go 技术栈约定**
- Go 1.21+，Gin 框架，GORM ORM，`gopkg.in/yaml.v3` 读写 config.yaml
- MySQL 8.0 / TiDB 兼容，Redis 7 / Upstash 兼容
- 日志使用标准库 `log/slog`

## UX & Interaction Patterns

**安装步骤指示器（UX-DR16）**
- 5 个圆圈：已完成（绿色背景 + 黑色 ✓）；进行中（`--accent` 背景 + 白色数字 + `box-shadow: 0 0 12px var(--accent-glow)`）；待完成（`--surface-3` 背景 + 细边框 + `--fg-muted` 数字）

**表单状态样式（UX-DR25）**
- focus：`border-color: var(--accent)` + `box-shadow: 0 0 0 3px var(--accent-dim)`
- 验证错误：`border-color: var(--red)` + 红色错误文字
- 连接测试成功：绿色 `✓ Connected` 徽章

**安装向导设计 token**
- 暗色主题：`--bg: #0e1017`，`--accent: #f5a623`，`--accent-dim: rgba(245,166,35,0.15)`，`--accent-glow: rgba(245,166,35,0.35)`
- 亮色主题（`prefers-color-scheme: light`）：`--bg: #f5f1e8`，`--accent: #d4891a`

## Cross-Story Dependencies

- Story 1.1 → Story 1.2：1.1 建立项目骨架和配置检测；1.2 在此基础上添加安装向导路由和前端
- Story 1.2 → Story 1.3：1.3 复用 1.2 建立的安装向导前端框架，添加步骤 3-4
- Story 1.3 → Story 1.4：1.4 在所有步骤数据就绪后执行安装；须在 Story 1.4 实现时同步创建 6 个 GORM 模型骨架文件（供 AutoMigrate 使用）
- Story 1.4 → Epic 2：Epic 2 的全部 Story 依赖 Epic 1 安装完成后的 config.yaml 和已初始化的数据库/Redis 环境
