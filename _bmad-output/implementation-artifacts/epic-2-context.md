# Epic 2 Context: 实时聊天 Widget 核心

<!-- Compiled from planning artifacts. Edit freely. Regenerate with compile-epic-context if planning docs change. -->

## Goal

实现聊天 Widget 的完整核心功能：任何访客打开嵌入了 Widget 的页面即可看到悬浮按钮，点击展开后以匿名身份在频道内收发实时消息。覆盖范围包括：服务端频道/消息数据层、WebSocket Hub 架构、匿名身份派生、消息发送流程、Widget JS 文件（零依赖单文件）、折叠/展开 UI、输入区交互和多语言支持。这是产品的最小可用核心——Epic 3（账号系统）和后续 Epic 均以此为基础扩展。

## Stories

- Story 2.1: 频道数据层 & 服务端 API
- Story 2.2: WebSocket Hub 架构
- Story 2.3: 匿名用户身份 & WS 连接鉴权
- Story 2.4: 消息发送核心流程
- Story 2.5: Widget JS 基础架构 & 设计系统
- Story 2.6: Widget 折叠/展开 & 消息列表 UI
- Story 2.7: Widget 输入区交互 & 多语言

## Requirements & Constraints

**频道数据：**
- Channel 表包含 slug（唯一索引）、is_enabled、max_message_length、rate_limit_seconds、allow_anonymous、welcome_message、telegram_link 等配置字段
- Message 表包含 channel_id（FK）、user_id（nullable FK）、anon_id（nullable）、content、is_deleted、is_filtered
- GET /api/channels 无需 JWT，供 Widget 初始化查询默认频道 slug
- 每频道独立维护：Redis RPUSH/LTRIM 最近 500 条消息（24h TTL）+ MySQL 永久持久化
- 在线计数通过 Redis SADD/SREM/SCARD 维护，每次变化后广播 `{"type":"online_count","count":N}`

**WebSocket 实时层：**
- 单频道支持数百并发连接（v1 基线）；无状态架构（多实例共享 MySQL + Redis）
- 消息写入严格顺序：① INSERT MySQL 获取稳定 ID → ② Redis RPUSH+LTRIM+EXPIRE → ③ Hub.broadcast；步骤 1 失败中止，步骤 2/3 失败记录日志不回滚
- 断开后客户端指数退避重连，最大间隔 30s
- 新连接建立后立即推送 Redis 中最近 20 条历史消息

**匿名身份：**
- anonID = `SHA-256(RemoteAddr + server_salt)[:8]`（16 进制，8 字符）
- 显示名：以 anonID 数值哈希为种子，从形容词词表（≥20）和名词词表（≥20）各取一个英文单词组合，同一 anonID 结果稳定
- 注册用户通过 `?token=<JWT>` 传递身份；无 token 或无效 token → 匿名用户
- 已封禁用户 WS 升级前返回 HTTP 403；allow_anonymous=false 的频道拒绝匿名连接（HTTP 403）

**Widget JS：**
- 单文件 `web/widget/chatsphere.js`，零运行时外部依赖；gzip 后 < 30KB
- 所有 DOM 和 CSS 限定在 `[data-cs="root"]` 命名空间，不污染宿主页面
- 通过 `<script>` data-* 属性或 `window.ChatSphere.init(opts)` 初始化
- 同一页面支持多实例互不干扰

**速率限制：**
- rate_limit_seconds 在服务端 WS readPump 层强制执行；超限消息直接丢弃并返回错误给发送方，不入库

**消息内容安全：**
- 所有消息内容写入前执行 `html.EscapeString()` 防 XSS

## Technical Decisions

**AD-1 分层约束（约束 Epic 2 所有后端代码）：**
HTTP 通路和 WebSocket 通路在 handler 层分叉，共享 service + repository 层。Hub.broadcast 是唯一的消息广播发起者；service 层只返回数据，不感知 WS 协议。handler 不直接操作 DB。

**AD-2 包边界（严格执行）：**
```
internal/repository/   ← ChannelRepository、MessageRepository interface + GORM 实现
internal/service/      ← ChannelService、MessageService，依赖接口不依赖 *gorm.DB
internal/handler/      ← Gin HTTP handler（/api/channels 等）
internal/ws/           ← hub.go、client.go、registry.go
web/widget/            ← chatsphere.js（go:embed 目标）
```
导入方向单向：handler → service → repository → model；ws → service（单向）。

**AD-3 Hub 设计：**
- 每个活跃频道一个 Hub goroutine，存入 HubRegistry（sync.Map[slug→*Hub]），懒创建
- 每个 Client 两 goroutine：readPump + writePump；writePump 是唯一写 websocket.Conn 的 goroutine
- 心跳：60s read deadline + 54s ping timer；PongHandler 重置 read deadline
- send channel 缓冲 256；关闭后 writePump 发送 CloseMessage 退出

**AD-4 WS 鉴权：**
- 注册用户 `?token=<JWT>` query param；鉴权在 WS 升级前完成（HTTP 阶段）
- 匿名用户身份在升级后从 RemoteAddr 派生；Client 身份字段升级后不可修改

**AD-11 Repository 模式：**
- ChannelRepository 和 MessageRepository 均定义 Go interface
- service 层仅依赖接口，GORM 实现和接口在同一包（internal/repository）下独立文件

**AD-12 错误格式：**
- HTTP 错误：`{"error":"...","code":"..."} ` + HTTP 状态码
- WS 错误：`{"type":"error","code":"...","message":"..."}`

**AD-7 静态资源：**
- Widget JS 通过 go:embed 嵌入，路由 `/widget/chatsphere.js` 无 JWT 保护

## UX & Interaction Patterns

**设计 Token（Widget JS 内实现）：**
- 暗色主题：`--bg: #0e1017`，`--surface: #13151e`，`--surface-2: #1a1d28`，`--accent: #f5a623`，`--accent-dim: rgba(245,166,35,0.15)`，`--accent-glow: rgba(245,166,35,0.35)`
- 亮色主题通过 `prefers-color-scheme: light` 覆盖

**折叠按钮：** 56×56px 圆形，accent 背景，页面加载 500ms 后 scale 动画出现

**展开/收起动画：** 320ms `cubic-bezier(0.4,0,0.2,1)`，`translateY(20px)→0` + fade

**消息气泡（iMessage 风格）：**
- 他人：左对齐，`background: var(--surface-2)`，`border-radius: 16px 16px 16px 4px`
- 自己：右对齐，`background: var(--accent)`，白色文字，`border-radius: 16px 16px 4px 16px`

**滚动锚定：** 距底部 ≤200px 自动滚到底；否则显示"↓ 有新消息"浮动按钮

**连接状态指示：** 顶部黄色细条（断开/重连中）；输入区 disabled 状态；绿色波形动画（已连接）

**输入框：** min-height 44px，auto-expand 最多 4 行；超 max_length 时边框红色 + 剩余字数计数

**所有可交互元素：** `outline: 2px solid var(--accent); outline-offset: 2px`；最小触摸区域 44×44px

**prefers-reduced-motion：** 所有动画降级为瞬切

## Cross-Story Dependencies

- **2.1 → 2.2：** Hub 需要 ChannelRepository 验证频道存在（GetBySlug）
- **2.1 → 2.4：** 消息写入需要 MessageRepository.Create()
- **2.2 → 2.3：** Client 结构体在 2.2 定义，2.3 扩展身份字段
- **2.2 → 2.4：** 消息广播通过 Hub.broadcast；历史推送在 2.4 实现
- **2.3 → 2.4：** readPump 中速率限制使用 Client.lastMessageAt
- **2.5 → 2.6/2.7：** Widget JS 文件和 CSS token 系统是 UI 故事的基础
- **Epic 1 → Epic 2：** AutoMigrate 在安装时已创建 channels 和 messages 表；Epic 2 Story 2.1 使用已有模型骨架文件（user.go/channel.go/message.go 已在 Epic 1 Story 1.4 中创建）
- **接缝 A（Epic 2 → Epic 3）：** 用户在 Widget 内登录后，Widget JS 必须断开当前 WS 并携带新 JWT 重连；此约定写入 Story 2.3 的 ACs，Epic 3 Story 3.2 实现登录 UI 时必须触发此行为
