---
stepsCompleted: ["step-01", "step-02", "step-03", "step-04"]
inputDocuments:
  - "_bmad/planning_artifacts/prds/prd-chatsphere-2026-11-08/prd.md"
  - "_bmad/planning_artifacts/architecture/architecture-chatsphere-2026-11-08/ARCHITECTURE-SPINE.md"
  - "_bmad/planning_artifacts/ux-designs/ux-chatsphere-2026-11-08/DESIGN.md"
  - "_bmad/planning_artifacts/ux-designs/ux-chatsphere-2026-11-08/EXPERIENCE.md"
---

# ChatSphere - Epic Breakdown

## Overview

本文件为 ChatSphere v1 完整 Epic 和 Story 分解，从 PRD、架构脊骨和 UX 规范中提取所有可实现的需求，拆解为可交付的开发单元。

## Requirements Inventory

### Functional Requirements

**F1 — 嵌入 Widget**

FR-1.1: Widget 通过单个 `<script>` 标签嵌入，支持 `data-server`、`data-channel`、`data-position`（bottom-left/bottom-right）、`data-title`、`data-notice` 等 data-* 属性配置。

FR-1.2: 支持 `window.ChatSphere.init(opts)` 手动初始化 API，参数与 data-* 属性对应；支持 `ChatSphere.setTitle()`、`ChatSphere.setNotice()` 运行时方法。

FR-1.3: Widget 所有 CSS 样式以 `[data-cs="root"]` 为根作用域，不与宿主页面样式产生冲突。

FR-1.4: Widget 支持折叠/展开两种状态；折叠时在角标显示未读消息数量。

FR-1.5: Widget 展开后展示频道头部（标题、在线人数）、消息列表、输入区域；支持"侧边栏"展开模式显示社区规则和在线用户统计。

FR-1.6: 输入区域提供 emoji 选择器（≥30 种常用表情），支持浏览器 Speech Recognition API 语音输入（有麦克风权限时显示语音按钮）。

FR-1.7: 每条消息展示：发送者国旗（基于 IP 地理位置）、声誉徽章（由管理员分配的积分段）、匿名/注册标识。

FR-1.8: 注册用户可对消息进行举报，举报后消息对举报者本地隐藏，举报进入管理后台待处理队列。

**F2 — 用户系统**

FR-2.1: 匿名用户以 SHA-256(IP + 盐) 生成稳定哈希身份，显示名以"形容词+名词"英文组合生成，同一 IP 在同一频道名称稳定。

FR-2.2: 注册接口接受用户名（唯一）、密码（bcrypt 哈希）、邮箱（可选）；登录返回 30 天有效期 JWT。

FR-2.3: 登录/注册流程在 widget 内以覆盖层形式完成，不跳转页面。

FR-2.4: 用户 profile 包含：用户名、头像颜色（用户名哈希确定，12 色调色板）、所在国家（基于 IP）、声誉积分（管理员可调整）、角色（admin/moderator/member）、封禁状态、最后在线时间。声誉积分以颜色徽章显示，默认4段（无/铜/银/金）；管理员可自定义积分段阈值、徽章颜色和标签文案。

FR-2.5: @ 提及功能：被提及用户 widget 折叠态显示红色提及计数角标；收到浏览器声音提示（默认内置音效，可关闭）；展开后包含 @ 的消息高亮显示并自动滚动至该消息；提及计数在用户查看对应消息后清零。管理员可在后台上传自定义音效文件替换默认音效。

FR-2.6: 私信功能：注册用户可对其他注册用户发起私信；入口在消息列表用户名旁；以浮层抽屉形式从 widget 侧边划入展示；收到私信时在 widget 角标叠加提示数字；消息持久化 MySQL；管理员可开启端对端加密开关（v1 仅记录配置意图，不实际执行加密）。

**F3 — 多频道**

FR-3.1: 管理员可创建任意数量频道，每个频道有独立的 slug（唯一）。

FR-3.2: 频道配置项：名称、slug、描述、是否为默认频道、是否启用、最大消息长度、发言速率限制（秒/次）、是否允许匿名用户、欢迎消息、Telegram 社群链接（可选）。

FR-3.3: 每个频道独立维护：Redis 最近 500 条消息缓存（24h TTL）+ MySQL 永久持久化；新用户连接时推送历史记录。

FR-3.4: 频道在线人数通过 Redis 维护，实时更新，显示在 widget 头部。

FR-3.5: 单个频道支持数百并发 WebSocket 连接（v1 基线），无状态架构支持未来水平扩展。

**F4 — 安装向导**

FR-4.1: 启动时检测 config.yaml 是否存在；若不存在，HTTP 服务在 `:8080/setup` 提供安装向导，阻塞正常服务启动。

FR-4.2: 向导分步骤完成：1) 数据库配置与连接测试（分字段：地址、端口、账号、密码、数据库名）；2) Redis 配置与连接测试（分字段：地址、端口、密码、库编号）；3) 管理员账号创建；4) 站点基本信息（站点名、默认频道名、管理后台路径）。

FR-4.3: 安装完成后自动生成 config.yaml，每个配置项附带中英文注释。

FR-4.4: 安装完成后服务自动切换到正常运行模式；向导前端显示实时安装日志。

FR-4.5: config.yaml 存在但连接失败时，服务启动返回明确错误信息，不静默失败。

**F5 — 管理后台**

FR-5.1: Dashboard 展示实时统计：总在线用户数、今日消息数、活跃频道数、待处理举报数。

FR-5.2: 频道管理：列表 + 新建、编辑、启用/停用频道；编辑面板覆盖 FR-3.2 全部配置项。

FR-5.3: 消息管理：按频道、关键词、用户名、时间范围筛选；支持单条和批量软删除。

FR-5.4: 用户管理：用户列表搜索；支持封禁/解封、调整声誉积分、查看历史消息。

FR-5.5: 举报管理：按状态分标签页（待处理/已处理/已忽略）；关联原始消息；支持删除消息并标记已处理，或直接忽略。

FR-5.6: Widget 嵌入代码生成器：可视化配置（服务器地址、频道、位置），实时生成可复制的 `<script>` 片段。

FR-5.7: 机器人与审核规则（见 F6）。

FR-5.8: Widget 主题自定义：管理员可修改 Widget accent color；颜色选择器（原生 `<input type="color">`）；实时预览（< 100ms）；可自定义 --accent/--accent-dim/--accent-glow；保存后服务端生成内联 CSS 片段注入 Widget 初始化脚本。

**F6 — 审核机器人**

FR-6.1: 管理员可为每个频道配置屏蔽词列表，支持精确匹配（普通字符串）和正则表达式（以 `/` 包裹标识）两种规则，可混合使用。

FR-6.2: 命中屏蔽词时消息被拦截，不发布到频道；Bot 以系统消息形式在频道内发出简短提示。

FR-6.3: Bot 以固定系统用户身份（`ChatSphere Bot`）出现，有专属头像标识，不可被普通用户 @ 或私信。

FR-6.4: 命中拦截的消息在管理后台消息列表中可见（标注"已过滤"），管理员可选择恢复或永久删除。

FR-6.5: 速率限制（rate_limit_seconds 频道配置）由服务端 WebSocket 层强制执行，超限消息直接丢弃并向客户端返回提示。

**F7 — Widget 多语言**

FR-7.1: Widget 所有面向用户的 UI 文案支持通过 `data-lang` 属性或 `init()` 参数指定语言。

FR-7.2: 默认语言为英文（en）；v1 内置简体中文（zh-CN）预置翻译包。

FR-7.3: 支持完全自定义文案覆盖：`init({ labels: { sendButton: "发送", ... } })`，任意 key 可单独覆盖，未覆盖 key 回退到所选语言默认值。

FR-7.4: 额外语言以社区贡献方式在后续版本添加；v1 只交付 en 和 zh-CN，翻译文件结构设计为可扩展格式。

---

### NonFunctional Requirements

NFR-1: 性能 — 单频道支持 500 个并发 WebSocket 连接（v1 基线）；消息端到端延迟 < 200ms p95（同地区部署）；Widget JS 文件 gzip 压缩后 < 30KB；管理后台页面首次加载 < 2s（本地网络）。

NFR-2: 可靠性 — 应用层无状态设计，支持水平扩展（多实例共享同一 MySQL + Redis）；WebSocket 断开后客户端自动重连（指数退避，最大间隔 30s）；服务进程支持优雅关闭（SIGTERM），等待活跃连接处理完成后退出。

NFR-3: 安全 — 所有 API 端点（除 WebSocket 和公开频道列表外）需 JWT 鉴权；管理后台路径可自定义（`admin.path` 配置项）；消息内容存储前 HTML 转义防 XSS；匿名用户 IP 以 SHA-256 + 随机盐哈希后存储；JWT 密钥在安装向导时由服务端 `crypto/rand` 自动生成，写入 config.yaml。

NFR-4: 部署 — 提供 Docker Compose 一键部署（app + MySQL + Redis）；单二进制文件，web 静态资源以 go:embed 打包；支持反向代理（Nginx/Caddy）部署，配置文档提供示例。

---

### Additional Requirements

**来自 Architecture Spine（影响实现的技术决策）：**

- **AD-1 分层约束**：HTTP 通路和 WebSocket 通路在 handler 层分叉，共享 service + repository 层；任何消息广播调用必须由 Hub 发起，service 只返回数据。禁止 handler 直接操作 DB、Hub 逻辑泄漏进 HTTP handler。
- **AD-2 包边界**：严格按 `cmd → internal/*` → `handler → service → repository → model` 导入方向；`internal/ws` 单向依赖 `internal/service`。禁止跨层跳跃导入，禁止循环依赖。
- **AD-3 Hub 设计**：每个活跃频道一个 Hub goroutine，存入 HubRegistry(sync.Map[slug→*Hub])，按需懒创建；每个 Client 两 goroutine（readPump + writePump）；writePump 是唯一向 websocket.Conn 写数据的 goroutine；心跳：60s read deadline + 54s ping。
- **AD-4 WS 鉴权**：注册用户 `?token=<JWT>` query param；匿名用户 `SHA-256(IP + 盐)[:8]` 派生身份；封禁用户升级前拒绝（HTTP 403）。
- **AD-5 消息写入顺序**：严格顺序 1→2→3：1) INSERT MySQL（获取稳定ID）2) RPUSH+LTRIM+EXPIRE Redis 3) Hub.broadcast。步骤1失败中止；步骤2/3失败记录日志不回滚 MySQL。在线计数：register 时 SADD，unregister 时 SREM，每次变化后 SCARD 并广播 online_count。
- **AD-6 配置生命周期**：安装完成写入 config.yaml 后调用 os.Exit(0)；三种部署方式（Docker restart:always、systemd Restart=always、宝塔异常重启）均支持此模式；文档交付三份并列说明。
- **AD-7 静态资源嵌入**：go:embed 编译时打包 web/widget、web/admin、web/installer；Widget 路由无鉴权，Admin 路由 JWT 保护，Installer 路由仅安装模式注册。
- **AD-8 Admin 渲染策略**：服务端 html/template 渲染完整 HTML；动态交互用原生 JS fetch + 局部 DOM 更新；无 React/Vue/HTMX 依赖。
- **AD-9 Widget JS 架构**：单文件 chatsphere.js，无运行时外部依赖；目标 < 30KB gzip；支持同一页面多频道实例。
- **AD-11 Repository 模式**：每个实体（User, Channel, Message, Report, DirectMessage）定义 Go interface；service 层仅依赖接口，不依赖具体类型。GORM 实现和 Redis 实现在同一包下独立文件。
- **AD-12 错误处理**：HTTP 返回 `{"error":"...","code":"..."}` + HTTP 状态码；WebSocket 返回 `{"type":"error","code":"...","message":"..."}`；内部错误 ERROR 级 slog 日志，用户侧返回通用消息。
- **AD-13 DM 加密**：v1 明文存储；管理员后台加密开关仅写入 config flag，不实际加密，界面标注"功能在后续版本启用"。
- **数据模型种子**：需实现以下 GORM 模型：User, Channel, Message, DirectMessage, Report, ReputationTier（含所有字段和外键关系）。
- **部署文档**：需交付 docker-compose.yml + systemd unit 模板 + 宝塔配置步骤说明（三份并列），以及 Nginx WebSocket 代理配置示例。

---

### UX Design Requirements

**来自 DESIGN.md（视觉实现要求）：**

UX-DR1: 实现完整 CSS 设计 token 系统：暗色主题（`--bg: #0e1017`，`--surface: #13151e`，`--surface-2: #1a1d28`，`--surface-3: #222638`，`--accent: #f5a623`，`--accent-dim: rgba(245,166,35,0.15)`，`--accent-glow: rgba(245,166,35,0.35)` 等全部 token）+ 亮色主题覆盖（`--bg: #f5f1e8`，`--accent: #d4891a` 等）。

UX-DR2: 实现 iMessage 风格气泡消息：他人消息左对齐深色气泡（`border-radius: 16px 16px 16px 4px`，使用 `--surface-2` 背景）；自己消息右对齐橙色气泡（`border-radius: 16px 16px 4px 16px`，使用 `--accent` 背景，白色文字）。

UX-DR3: 全部 UI 图标使用 SVG 线条图标（viewBox="0 0 16 16"，stroke-width="1.5"，stroke-linecap/linejoin="round"，fill="none"，颜色继承 currentColor），严禁使用 Emoji 作为 UI 图标。

UX-DR4: 实现 4 级声誉徽章：ANON（灰色 `--surface-3` 背景）、BRONZE（10+ 积分，`rgba(205,127,50,0.2)` 背景 + `#cd7f32` 文字）、SILVER（50+ 积分，`rgba(148,163,184,0.2)` 背景 + `#94a3b8` 文字）、GOLD（100+ 积分，`rgba(251,191,36,0.2)` 背景 + `#fbbf24` 文字）；颜色和阈值由管理员可自定义。

UX-DR5: 实现管理员自定义 Widget 主题：服务端接收 accent color 保存后生成 CSS 片段（包含 `--accent`，自动计算 `--accent-dim` 和 `--accent-glow`），注入 Widget 初始化脚本；admin 页面内嵌实时 Widget 预览。

UX-DR6: Widget 折叠按钮：56×56px 圆形，`background: var(--accent)`，`box-shadow: 0 4px 16px var(--accent-glow)`；未读数红色圆形角标（右上角），≥100 显示 "99+"。

UX-DR7: 私信抽屉：从 widget 左侧 `translateX(-100%)` 滑入，`transition: 280ms cubic-bezier(0.4,0,0.2,1)`，宽度等于 widget 宽；顶部返回按钮；频道聊天视图不销毁只被遮盖。

**来自 EXPERIENCE.md（交互行为要求）：**

UX-DR8: 消息列表滚动锚定：新消息到来时，若用户距底部 ≤ 200px 则自动滚到底；否则显示"↓ 有新消息"浮动提示点击跳底。

UX-DR9: 消息时间戳与头像显示规则：同一用户连续消息间隔 < 5 分钟时，不重复显示头像和昵称；≥ 5 分钟时显示时间戳分隔。

UX-DR10: 输入框自动扩展高度（最大 4 行后出现内部滚动条）；超过频道 max_length 时边框变红，右下角显示剩余字符计数（红色）。

UX-DR11: `@` 触发提及补全：输入 `@` 后出现在线用户下拉列表，键盘上下键选择，Enter 确认插入。

UX-DR12: Widget 折叠/展开动画：320ms `cubic-bezier(0.4,0,0.2,1)`；折叠按钮出现动画 `scale(0.8) → scale(1)` + fade；展开 `translateY(20px) → translateY(0)` + fade。

UX-DR13: 消息发送状态：发送中 opacity 0.6 + 右下角转圈图标；已发送 opacity 1 + 图标消失；发送失败气泡底部红色"发送失败，重试"链接。

UX-DR14: WebSocket 连接状态指示：连接中显示 `...`；已连接显示在线人数 + 绿色波形动画；断开/重连中顶部黄色细条显示"重新连接中…"，输入区禁用；重连失败显示"连接失败，点击重试"按钮。

UX-DR15: 覆盖层/抽屉滑入 280ms；新消息滑入（translateY 8px + fade）160ms；微交互（hover/focus/active）80–120ms；`prefers-reduced-motion: reduce` 时所有动画降级为瞬切。

UX-DR16: 安装向导5步骤圆圈指示器：已完成（绿色背景 + 黑色 ✓）；进行中（`--accent` 背景 + 白色数字 + `box-shadow: 0 0 12px var(--accent-glow)`）；待完成（`--surface-3` 背景 + 细边框 + `--fg-muted` 数字）。

UX-DR17: 所有可交互元素有清晰 focus outline：`outline: 2px solid var(--accent); outline-offset: 2px`，不用浏览器默认蓝色 focus ring。

UX-DR18: 触摸友好：所有可点击区域最小 44×44px（WCAG 2.5.5）；气泡操作通过长按触发操作菜单（代替 hover）。

UX-DR19: 消息气泡操作菜单（右键/长按）：「回复」「@ 他」「举报」（仅他人消息）。

UX-DR20: 登录/注册覆盖层：从底部 `translateY(100%) → translateY(0)` 滑入；Tab 切换登录/注册表单；登录成功后覆盖层下滑消失，输入框自动获焦。

UX-DR21: Widget 移动端响应式：≤ 400px 视口下展开铺满宽度（`width: calc(100vw - 32px)`），保留左右各 16px 安全距离。

UX-DR22: Admin 侧边栏移动端适配：< 520px 宽时折叠为 44px 宽图标条，文字隐藏，tooltip 显示文字。

UX-DR23: 屏幕阅读器支持：Widget 根节点 `role="complementary" aria-label="ChatSphere 聊天窗口"`；消息列表 `role="log" aria-live="polite" aria-atomic="false"`；私信抽屉和覆盖层开启时 focus trap。

UX-DR24: 颜色对比度满足 WCAG AA：`--fg` 在 `--surface` 上对比度 ≥ 7:1；`--accent` (#f5a623) 在深色背景 (#13151e) 上对比度 ≥ 6.8:1 (AA)。

UX-DR25: 表单验证状态：focus 时 `border-color: var(--accent)` + `box-shadow: 0 0 0 3px var(--accent-dim)`；验证错误时 `border-color: var(--red)` + 红色错误文字；连接测试成功时显示绿色 `✓ Connected` 徽章。

---

### FR Coverage Map

```
FR-1.1 → Epic 2  Widget <script> 嵌入 & data-* 属性配置
FR-1.2 → Epic 2  window.ChatSphere.init() JS API
FR-1.3 → Epic 2  [data-cs="root"] CSS 作用域隔离
FR-1.4 → Epic 2  折叠/展开状态 & 未读消息数角标
FR-1.5 → Epic 2  频道头部 + 消息列表 + 输入区展示
FR-1.6 → Epic 4  Emoji 选择器 & 语音输入按钮
FR-1.7 → Epic 3  消息国旗 + 声誉徽章显示
FR-1.8 → Epic 4  消息举报功能
FR-2.1 → Epic 2  匿名用户 SHA-256 哈希稳定身份
FR-2.2 → Epic 3  注册/登录接口 + bcrypt + JWT
FR-2.3 → Epic 3  Widget 内覆盖层登录/注册（不跳转页面）
FR-2.4 → Epic 3  用户 Profile + 声誉积分系统 + 4段徽章
FR-2.5 → Epic 4  @ 提及通知（角标+音效+高亮+清零）
FR-2.6 → Epic 4  私信功能（DM 抽屉 + 持久化）
FR-3.1 → Epic 2  频道创建（slug 唯一）
FR-3.2 → Epic 2  频道配置项（速率/匿名/欢迎语等）
FR-3.3 → Epic 2  Redis 500条缓存 + MySQL 永久持久化
FR-3.4 → Epic 2  在线人数 Redis 实时计数 & 广播
FR-3.5 → Epic 2  单频道数百并发 + 无状态架构
FR-4.1 → Epic 1  config.yaml 检测 & 安装向导触发
FR-4.2 → Epic 1  5步骤安装向导（DB/Redis/Admin/站点）
FR-4.3 → Epic 1  config.yaml 生成（含中英文注释）
FR-4.4 → Epic 1  安装后自动切换模式 + 实时日志流
FR-4.5 → Epic 1  config.yaml 存在但连接失败时明确错误
FR-5.1 → Epic 5  Dashboard 实时统计（4项指标）
FR-5.2 → Epic 5  频道管理 CRUD + 启用/停用
FR-5.3 → Epic 5  消息管理（筛选 + 软删除）
FR-5.4 → Epic 5  用户管理（封禁/积分/历史）
FR-5.5 → Epic 5  举报管理（3状态标签页处理）
FR-5.6 → Epic 5  Widget 嵌入代码生成器
FR-5.7 → Epic 6  （指向 F6 机器人审核，同 FR-6.1~6.4）
FR-5.8 → Epic 6  Widget 主题自定义（accent color CSS注入）
FR-6.1 → Epic 6  屏蔽词规则（精确匹配 + 正则表达式）
FR-6.2 → Epic 6  命中拦截 + Bot 系统消息提示
FR-6.3 → Epic 6  Bot 专属身份（不可 @ 或私信）
FR-6.4 → Epic 6  已过滤消息管理（恢复或永久删除）
FR-6.5 → Epic 2  速率限制（WS 层强制执行，readPump）
FR-7.1 → Epic 2  Widget 多语言 data-lang 参数支持
FR-7.2 → Epic 2  内置 en + zh-CN 翻译包
FR-7.3 → Epic 2  自定义文案覆盖 init({ labels: {} })
FR-7.4 → Epic 2  可扩展翻译文件结构（社区贡献格式）
```

**跨 Epic 接缝约定（来自高级启发）：**
- **接缝 A（Epic 2 → Epic 4）：** 匿名→注册身份切换时，客户端必须断开当前 WS 连接并携带新 JWT 重新连接；此行为写入 Epic 2 相关 Story 的验收标准。
- **接缝 B（Epic 2 内）：** i18n（FR-7.1~7.4）为 Epic 2 内最后一批 Story，核心聊天功能验收不依赖 i18n 完成。

## Epic List

### Epic 1：安装向导 & 系统基础

Kevin 能够通过浏览器安装向导在 30 分钟内完成 ChatSphere 安装——分步填写数据库/Redis 连接信息并测试、创建管理员账号、设置站点信息，安装完成后服务自动切换到正常运行模式，无需手动编辑任何配置文件。

**涵盖 FR：** FR-4.1, FR-4.2, FR-4.3, FR-4.4, FR-4.5

### Epic 2：实时聊天 Widget 核心

任何访客打开页面即可看到悬浮 Widget，点击展开后能以匿名身份在频道内发送和接收实时消息；Widget 完整呈现消息列表、在线人数、输入区；多语言文案、CSS 样式不干扰宿主页面。

**涵盖 FR：** FR-1.1, FR-1.2, FR-1.3, FR-1.4, FR-1.5, FR-2.1, FR-3.1, FR-3.2, FR-3.3, FR-3.4, FR-3.5, FR-6.5, FR-7.1, FR-7.2, FR-7.3, FR-7.4

### Epic 3：用户账号 & 身份系统

访客可以在 Widget 内完成注册/登录（不跳转页面），获得持久账号身份；消息旁展示国旗、声誉徽章（ANON/BRONZE/SILVER/GOLD）；管理员可在后台调整声誉积分。

**涵盖 FR：** FR-2.2, FR-2.3, FR-2.4, FR-1.7

### Epic 4：社交互动功能

注册用户可以 @ 提及他人并收到通知（角标+音效+高亮）、使用 emoji 选择器发表情、通过侧边私信抽屉与他人私聊，并对违规消息一键举报。

**涵盖 FR：** FR-1.6, FR-1.8, FR-2.5, FR-2.6

### Epic 5：管理后台

管理员可以通过后台实时查看系统统计（Dashboard）、对频道进行完整 CRUD、筛选/软删除消息、封禁/解封用户/调整积分、处理举报，并一键生成 Widget 嵌入代码。

**涵盖 FR：** FR-5.1, FR-5.2, FR-5.3, FR-5.4, FR-5.5, FR-5.6

### Epic 6：内容审核 & 个性化定制

管理员可以为每个频道配置屏蔽词规则（精确匹配 + 正则）自动拦截违禁内容；自定义 Widget 强调色并实时预览，保存后新访客立即生效；管理声誉积分分段和 @ 提醒音效。

**涵盖 FR：** FR-5.7/FR-6.1, FR-6.2, FR-6.3, FR-6.4, FR-5.8

---

## Stories

### Epic 1：安装向导 & 系统基础

#### Story 1.1: 项目基础启动 & 配置检测

**As a** 部署工程师，
**I want** 程序启动时自动检测 config.yaml 是否存在并做出正确响应，
**So that** 安装模式与正常运行模式之间能够自动区分，无需手动传参。

**Acceptance Criteria:**

1. 程序启动时检测当前工作目录是否存在 config.yaml
2. config.yaml 不存在时：仅注册 `/setup` 路由，HTTP 服务在 `:8080/setup` 启动，不初始化数据库/Redis；访问任何非 `/setup` 路径均返回 302 重定向到 `/setup`
3. config.yaml 存在且连接正常时：进入正常运行模式，全量路由注册，数据库/Redis 全部初始化，slog INFO 记录 "starting in normal mode"
4. config.yaml 存在但数据库或 Redis 连接失败时：stderr 输出明确错误信息（含失败原因，如 "failed to connect to MySQL: dial tcp ..."），进程以非零退出码退出，不静默失败（FR-4.5，AD-12）
5. 安装模式与正常模式互斥：同一进程不会同时注册安装路由和业务路由
6. main() 中模式判断逻辑通过 internal/config 包的函数完成，不直接在 main() 中读取文件

---

#### Story 1.2: 数据库与 Redis 连接配置（安装向导步骤 1-2）

**As a** 部署工程师，
**I want** 通过浏览器分步骤填写数据库和 Redis 连接信息并即时测试连通性，
**So that** 我能在进入下一步之前确认连接无误，无需猜测配置格式。

**Acceptance Criteria:**

1. GET /setup 返回安装向导 HTML 页面；顶部展示 5 步骤圆圈指示器，步骤 1 处于进行中状态（`--accent` 背景 + 白色数字 + `box-shadow: 0 0 12px var(--accent-glow)`）（UX-DR16）
2. 步骤 1（数据库配置）：5 个独立输入字段——地址（默认 127.0.0.1）、端口（默认 3306）、账号（默认 root）、密码（password input）、数据库名（默认 chatsphere）
3. 步骤 2（Redis 配置）：4 个独立输入字段——地址（默认 127.0.0.1）、端口（默认 6379）、密码（可为空）、库编号（默认 0）
4. 每个步骤有独立的"测试连接"按钮；点击后 POST 至 `/setup/test-db` 或 `/setup/test-redis`；成功时按钮旁显示绿色 `✓ Connected` 徽章；失败时显示红色错误文案（UX-DR25）
5. "下一步"按钮在连接测试通过后启用；步骤间通过 JS 切换 DOM 显示/隐藏，步骤指示器同步更新
6. 表单 focus 状态：`border-color: var(--accent)` + `box-shadow: 0 0 0 3px var(--accent-dim)`；验证错误时 `border-color: var(--red)` + 红色错误文字（UX-DR25）
7. 安装向导 HTML 通过 go:embed 嵌入 `web/installer/`，安装模式 Gin 路由直接 ServeFS（AD-7）

---

#### Story 1.3: 管理员账号与站点信息配置（安装向导步骤 3-4）

**As a** 部署工程师，
**I want** 在安装向导中创建管理员账号并填写站点基本信息，
**So that** 安装完成后我能直接以管理员身份登录后台，无需再单独创建账号。

**Acceptance Criteria:**

1. 步骤 3（管理员账号）：用户名输入框、密码输入框（含密码强度进度条：1-3 字符红/4-7 字符黄/8+ 字符绿）、确认密码输入框、显示/隐藏密码切换按钮（SVG 眼睛图标，UX-DR3）
2. 密码规则：≥ 8 字符；确认密码实时校验，不一致时立即显示红色错误文字
3. 用户名规则：3-20 字符，仅允许字母/数字/下划线；不合规时 inline 错误提示
4. 步骤 4（站点信息）：站点名称（文本，必填）、默认频道名（文本，必填，默认 "general"）、管理后台路径（文本，默认 "admin"，说明文字提示"如 admin 则后台地址为 /admin"）
5. 步骤指示器跟随当前步骤更新（已完成步骤绿色背景 + ✓，UX-DR16）
6. 所有字段均有最小 44×44px 交互区域（UX-DR18）；密码输入框使用 `type="password"`，不可自动填充历史密码（`autocomplete="new-password"`）
7. 步骤 3 和步骤 4 数据在前端 JS 本地持有，不单独提交；与步骤 1-2 数据一起在步骤 5 安装时统一提交

---

#### Story 1.4: 安装完成 & 服务自动切换

**As a** 部署工程师，
**I want** 点击安装向导的"开始安装"按钮后看到实时进度日志，安装完成后服务自动重启进入正常模式，
**So that** 我无需手动重启服务或修改任何文件即可完成完整部署。

**Acceptance Criteria:**

1. 步骤 5 展示"开始安装"按钮；点击后通过 SSE（Server-Sent Events）连接 `/setup/install` 端点，实时接收安装日志流
2. 安装日志流共 5 步，每步完成后推送日志行：① 连接数据库验证 ② AutoMigrate 全部 6 张表（User/Channel/Message/DirectMessage/Report/ReputationTier）③ 创建管理员账号（bcrypt 哈希密码）④ 创建默认频道（使用步骤 4 中填写的频道名） ⑤ 写入 config.yaml（FR-4.3）
2a. **模型骨架文件（依赖说明）**：此 Story 的 AutoMigrate 需要 6 个 GORM 模型 Go 文件预先存在。Dev agent 实现本 Story 时须同步创建 `internal/model/` 下的骨架文件（user.go / channel.go / message.go / direct_message.go / report.go / reputation_tier.go），内含完整字段定义和 GORM tag；Repository/Service/Handler 实现由后续 Story 完成，此时骨架只需满足 `db.AutoMigrate()` 能编译通过即可
3. config.yaml 自动生成内容：数据库连接信息、Redis 连接信息、管理员 JWT 密钥（`crypto/rand` 32 字节 base64）、安装向导随机盐（32 字节）、管理后台路径；每个配置项附中英文注释（FR-4.3）
4. 任意步骤失败时：SSE 推送错误行，前端展示红色错误信息，不继续后续步骤，不写入 config.yaml
5. 全部步骤成功后：前端展示"安装完成！正在重启服务…"，服务端调用 `os.Exit(0)`；Docker restart:always / systemd Restart=always / 宝塔异常重启均会触发进程重启进入正常模式（AD-6）
6. 进程重启后服务在 `:8080` 以正常模式启动（config.yaml 存在，连接成功），安装路由不再注册
7. 交付部署文档：`docs/deploy/docker-compose.yml`（app + MySQL 8.0 + Redis 7）、`docs/deploy/chatsphere.service`（systemd unit）、`docs/deploy/deploy-baota.md`（宝塔面板配置步骤）、`docs/deploy/nginx-websocket.conf`（Nginx WebSocket 反向代理示例）（NFR-4，AD-6）

---

### Epic 2：实时聊天 Widget 核心

#### Story 2.1: 频道数据层 & 服务端 API

**As a** 管理员，
**I want** 频道和消息的数据模型、Repository 接口、服务层以及频道查询 API 全部就位，
**So that** Widget 和后续 Story 能够读取频道信息并存储消息。

**Acceptance Criteria:**

1. GORM Channel 模型字段：id, name, slug(唯一索引), description, is_default, is_enabled, max_message_length(default 500), rate_limit_seconds(default 0), allow_anonymous(default true), welcome_message, telegram_link, created_at, updated_at（FR-3.2）
2. GORM Message 模型字段：id, channel_id(FK), user_id(nullable FK), anon_id(nullable), content, is_deleted(default false), is_filtered(default false), created_at
3. ChannelRepository Go interface：GetAll(), GetEnabled(), GetBySlug(slug), Create(ch), Update(ch), Delete(id), GetDefault()；GORM 实现在 `internal/repository/channel_repo.go`（AD-11）
4. MessageRepository Go interface：Create(msg), GetByChannel(channelID, limit, offset), SoftDelete(id), GetFiltered(channelID)；GORM 实现在 `internal/repository/message_repo.go`（AD-11）
5. ChannelService：通过 ChannelRepository 接口操作，不持有 *gorm.DB；MessageService：同理（AD-1，AD-2）
6. GET /api/channels 端点：返回全部已启用频道的 JSON 列表（无需 JWT），供 Widget 初始化时查询默认频道 slug
7. 安装向导 Story 1.4 中 AutoMigrate 执行后两张表结构自动创建；`internal/database/seed.go` 负责默认频道创建

---

#### Story 2.2: WebSocket Hub 架构

**As a** 访客，
**I want** 能够建立 WebSocket 长连接并持续收到频道内的实时消息推送，
**So that** 聊天室内容无需刷新页面即可实时更新。

**Acceptance Criteria:**

1. `internal/ws/hub.go`：Hub struct 含 clients map[*Client]bool、broadcast chan []byte、register chan *Client、unregister chan *Client；`Run()` goroutine 以 select 处理三个 channel，不阻塞（AD-3）
2. `internal/ws/registry.go`：HubRegistry 使用 sync.Map[channelSlug→*Hub]，`GetOrCreate(slug)` 懒创建 Hub 并启动其 `Run()` goroutine（AD-3）
3. `internal/ws/client.go`：Client struct 含 hub *Hub、conn *websocket.Conn、send chan []byte（缓冲 256）、以及身份字段（见 Story 2.3）
4. `readPump()`：设置 60s 读超时（`SetReadDeadline`），接收客户端 WS 消息并转发处理；`PongHandler` 重置读超时（AD-3）
5. `writePump()`：是唯一向 websocket.Conn 写数据的 goroutine（AD-3）；54s ping 定时器；从 send channel 读消息写入 WS；send channel 关闭后发送 CloseMessage 并退出
6. GET /ws/:slug 路由注册（正常模式下），Gin handler 负责 WS 升级并将 Client 注册到对应 Hub；安装模式下此路由不注册
7. Client 注册时：SADD `chatsphere:channel:{slug}:online {identity}`；注销时 SREM；每次变化后 SCARD 并广播 `{"type":"online_count","count":N}`（AD-5）

---

#### Story 2.3: 匿名用户身份 & WS 连接鉴权

**As a** 访客，
**I want** 无需登录就能连接到频道 WebSocket 并拥有一个稳定的匿名身份，
**So that** 我能立即开始聊天，消息在同一页面会话期间始终以同一名称显示。

**Acceptance Criteria:**

1. 匿名身份：`SHA-256(RemoteAddr + server_salt)[:8]`（16进制字符串，8 字符）作为 anonID（AD-4）
2. 匿名显示名：以 anonID 的数值哈希为种子，从形容词词表（≥20个）和名词词表（≥20个）各取一个英文单词组合为显示名（如 "Swift Hawk"），同一 anonID 结果稳定
3. WS 升级时鉴权流程：query param `?token=<JWT>` 存在且有效 → 提取 userID + username，isAnonymous=false；无 token 或无效 → 派生 anonID + 生成显示名，isAnonymous=true
4. 已封禁注册用户（banned_at != nil）尝试连接：WS 升级前返回 HTTP 403，body `{"error":"you are banned","code":"user_banned"}`（AD-4，AD-12）
5. 频道 allow_anonymous=false 时匿名用户连接：WS 升级前返回 HTTP 403，body `{"error":"anonymous users not allowed in this channel","code":"anon_not_allowed"}`
6. Client struct 新增字段：isAnonymous bool、userID uint（0表示匿名）、anonID string（空表示注册用户）、displayName string、channelSlug string；连接建立后这些字段不可修改
7. **接缝约定 A 写入 ACs**：当用户在 Widget 内完成登录/注册（见 Epic 3 Story 3.2），Widget JS 必须主动断开当前 WS 连接，并携带新获得的 JWT token 重新连接；服务端不支持同一连接上的身份切换

---

#### Story 2.4: 消息发送核心流程

**As a** 访客，
**I want** 在频道内发送消息并立即看到它出现在所有在线用户的消息列表中，
**So that** 我能与频道内的其他人实时交流。

**Acceptance Criteria:**

1. readPump 接收 WS 消息，解析 JSON `{"type":"message","content":"..."}`；其他 type 值忽略（不返回错误）
2. 速率限制：比较 lastMessageAt 与当前时间差和 channel.rate_limit_seconds；超限时向发送方推送 `{"type":"error","code":"rate_limit","message":"请稍后再试，剩余 Xs"}` 并丢弃消息（FR-6.5）
3. 消息内容 HTML 转义（`html.EscapeString()`）后进行写入顺序（AD-5 严格执行）：① MessageRepository.Create() INSERT MySQL 获取稳定 ID → ② Redis RPUSH + LTRIM(500) + EXPIRE(24h) → ③ Hub.broadcast 推送
4. MySQL 写入失败：返回错误给发送方 `{"type":"error","code":"internal_error","message":"消息发送失败"}`，不执行步骤 2/3
5. Redis 或广播失败：slog.ERROR 记录错误，不回滚 MySQL；消息已持久化，客户端通过历史加载可看到
6. 广播消息 payload：`{"type":"message","id":<uint>,"channelSlug":"...","content":"...","displayName":"...","anonID":"...","userID":<uint>,"isAnonymous":<bool>,"createdAt":"<RFC3339>"}`
7. 新 WS 连接建立后立即推送历史：LRANGE `chatsphere:channel:{slug}:messages 0 19`（最多 20 条），以 `{"type":"history","messages":[...]}` 一次性推送给新客户端

---

#### Story 2.5: Widget JS 基础架构 & 设计系统

**As a** 网站访客，
**I want** ChatSphere Widget 通过单个 `<script>` 标签加载后自动出现，且不影响宿主页面任何现有样式，
**So that** 任何网站都能零配置嵌入聊天功能。

**Acceptance Criteria:**

1. 单文件 `web/widget/chatsphere.js`，无任何运行时外部依赖（AD-9）；gzip 后目标 < 30KB
2. 所有 DOM 元素和 CSS 规则限定在 `[data-cs="root"]` 命名空间下，不使用全局选择器（FR-1.3，AD-9）
3. CSS 设计 token 系统完整实现（UX-DR1）：暗色主题全部 token（`--bg: #0e1017`，`--surface`，`--surface-2`，`--surface-3`，`--accent: #f5a623`，`--accent-dim`，`--accent-glow`，`--fg`，`--fg-muted`，`--fg-subtle`，`--green`，`--red`，`--amber` 等）；亮色主题通过 `prefers-color-scheme: light` 覆盖（`--bg: #f5f1e8`，`--accent: #d4891a` 等）
4. `<script>` 嵌入方式：`data-server`，`data-channel`，`data-position`（bottom-right/bottom-left），`data-title`，`data-notice`，`data-lang` 属性在脚本加载时自动读取并初始化（FR-1.1）
5. `window.ChatSphere.init(opts)` JS API：接受 `{ server, channel, position, title, notice, lang, labels }` 参数（FR-1.2）；`ChatSphere.setTitle(title)` 和 `ChatSphere.setNotice(notice)` 运行时方法
6. 同一页面多实例支持：每个 `<script>` 标签创建独立实例，互不干扰（FR-1.2 注）
7. go:embed 配置在 `internal/handler/static.go` 中声明：`//go:embed ../../../../web` 嵌入 web/ 目录；Widget 路由 `/widget/chatsphere.js` 无 JWT 保护（AD-7）
8. Widget 根节点 `role="complementary" aria-label="ChatSphere 聊天窗口"`（UX-DR23）

---

#### Story 2.6: Widget 折叠/展开 & 消息列表 UI

**As a** 访客，
**I want** 看到悬浮的 Widget 按钮，点击后展开精美的聊天面板并浏览消息历史，
**So that** 我能随时注意到新消息并方便地阅读频道对话。

**Acceptance Criteria:**

1. 折叠按钮：56×56px 圆形，`background: var(--accent)`，`box-shadow: 0 4px 16px var(--accent-glow)`，位于屏幕右下角（或左下角根据 data-position），页面加载 500ms 后以 `scale(0.8)→scale(1)` + fade 动画出现（UX-DR6，UX-DR12）
2. 点击折叠按钮展开：Widget 卡片以 `translateY(20px)→translateY(0)` + opacity 0→1，320ms `cubic-bezier(0.4,0,0.2,1)`；再次点击或点击关闭按钮收起（反向动画）（UX-DR12）
3. Widget 卡片：宽 340px，高 480px，`border-radius: 20px`，box-shadow Level 3（`0 20px 60px rgba(0,0,0,0.5)`）
4. 频道头部：频道名（`font-weight: 700`，`font-size: 15px`），在线人数 + 绿色波形动画指示器，右侧关闭 SVG 按钮（UX-DR3）
5. 未读消息角标：红色圆形叠在折叠按钮右上角，`scale(0)→scale(1)` 弹出动画；数字 ≥100 显示 "99+"；展开后计数清零（FR-1.4，UX-DR6）
6. 消息列表：`role="log" aria-live="polite" aria-atomic="false"`（UX-DR23）；他人气泡 `border-radius: 16px 16px 16px 4px`，`background: var(--surface-2)`，左对齐；自己气泡 `border-radius: 16px 16px 4px 16px`，`background: var(--accent)`，白色文字，右对齐（UX-DR2）
7. 滚动锚定：新消息到来时，若距底部 ≤200px 自动滚到底；否则显示"↓ 有新消息"浮动按钮（`position: absolute; bottom: 56px`），点击跳底（UX-DR8）
8. 消息时间戳分组：同用户 5 分钟内连续消息不重复头像/昵称；≥5 分钟显示时间戳分隔线（UX-DR9）
9. 新消息到来动画：`translateY(8px)→0` + fade，160ms（UX-DR15）；`prefers-reduced-motion` 时所有动画降级为瞬切（UX-DR15）
10. 所有可交互元素：`outline: 2px solid var(--accent); outline-offset: 2px`（UX-DR17）；最小点击区域 44×44px（UX-DR18）
11. 移动端响应式：≤400px 视口下 `width: calc(100vw - 32px)`（UX-DR21）

---

#### Story 2.7: Widget 输入区交互 & 多语言

**As a** 访客，
**I want** 能在输入框中输入并发送消息，看到发送状态反馈，并能以我偏好的语言使用 Widget，
**So that** 聊天体验流畅且易于理解。

**Acceptance Criteria:**

1. 输入框：`min-height: 44px`，左右 padding 12px，auto-expand 最多 4 行后出现内部滚动条（UX-DR10）；超过 channel.max_message_length 时边框变红，右下角显示剩余字符计数（红色）（UX-DR10）
2. Enter 发送消息；Shift+Enter 插入换行；发送后输入框清空并重新获焦
3. 右侧发送按钮：`background: var(--accent)`，SVG 发送图标（UX-DR3）；无内容时按钮禁用（`opacity: 0.5`）
4. 消息发送状态（UX-DR13）：发送中 → 气泡 opacity 0.6 + 右下角转圈 SVG 图标；已发送 → opacity 1 + 图标消失；失败 → 气泡底部红色"发送失败，重试"链接（点击重发）
5. 速率限制冷却：发送按钮禁用并显示倒计时 "3s..."（UX-DR14 相关）
6. WS 连接状态指示（UX-DR14）：连接中 → 在线人数区显示 `...`；已连接 → 在线人数 + 绿色波形动画；断开/重连中 → 顶部 4px 黄色细条 + "重新连接中…" 文字，输入区 disabled，指数退避重连（最大间隔 30s，NFR-2）；重连失败 → "连接失败，点击重试"按钮
7. i18n：`data-lang` 属性或 `init({lang})` 选择语言包（FR-7.1）；内置 `en`（默认）和 `zh-CN` 两套翻译（FR-7.2）；翻译 key 包含：sendButton, sendPlaceholder, connecting, reconnecting, reconnectFailed, rateLimitPrefix, mentionNotification 等所有面向用户文案
8. 自定义文案覆盖：`init({ labels: { sendButton: "发送", ... } })`，只覆盖指定 key，未覆盖 key 回退到语言默认值（FR-7.3）
9. 翻译文件为 JS 对象格式，可轻松增加新语言 key，社区贡献者只需添加一个对象（FR-7.4）
10. 所有 SVG 图标：`viewBox="0 0 16 16"`，`stroke-width="1.5"`，`fill="none"`，`stroke-linecap="round"`，`stroke-linejoin="round"`，颜色继承 `currentColor`（UX-DR3）

---

### Epic 3：用户账号 & 身份系统

#### Story 3.1: 用户注册 & JWT 登录

**As a** 访客，
**I want** 通过用户名和密码注册账号并登录，获得 JWT token，
**So that** 我能拥有在所有频道都可识别的持久身份。

**Acceptance Criteria:**

1. POST /api/auth/register：接受 `{username, password, email(可选)}`；返回 `{token, user: {id, username, role, avatarColor}}`；失败返回 `{"error":"...","code":"..."}` 格式（AD-12）
2. POST /api/auth/login：接受 `{username, password}`；验证 bcrypt hash；返回同上格式；用户不存在或密码错误统一返回 `{"error":"invalid credentials","code":"auth_failed"}` + HTTP 401
3. JWT payload：`{sub: userID, username, role, iat, exp}`；有效期 30 天；密钥从 config.yaml 读取（FR-2.2）
4. 注册验证：username 3-20 字符，仅允许 `^[a-zA-Z0-9_]+$`；password ≥8 字符；username 重复 → HTTP 409 `{"error":"username already taken","code":"username_taken"}`
5. 密码以 bcrypt（cost=12）哈希后存储，不存储明文（FR-2.2）
6. User 模型字段：id, username(唯一索引), password_hash, email, role(default "member"), avatar_color, country_code, reputation(default 0), banned_at(nullable), last_seen_at, created_at, updated_at
7. avatarColor：`SHA-256(username) % 12` 映射到 12 色调色板中的一个颜色（hex string），调色板定义在 `internal/model/user.go`（FR-2.4）
8. country_code：从 HTTP 请求头 `X-Forwarded-For` 或 RemoteAddr 解析 IP，查询 MaxMind GeoLite2（本地 DB）或返回空字符串（best-effort，不阻塞注册）

---

#### Story 3.2: Widget 内登录/注册覆盖层

**As a** 访客，
**I want** 在 Widget 内直接完成登录或注册，不跳转到任何外部页面，
**So that** 我能在不离开当前页面的情况下获得持久账号。

**Acceptance Criteria:**

1. Widget 频道头部区域展示"登录"按钮（SVG 用户图标 + 文字，UX-DR3）；已登录时显示用户名和头像色点
2. 点击"登录"按钮：覆盖层从底部 `translateY(100%)→translateY(0)` 滑入，280ms `cubic-bezier(0.4,0,0.2,1)`；Tab 切换"登录"和"注册"两个 tab（UX-DR20）
3. 登录 Tab：username 输入框、password 输入框（含显示/隐藏切换 SVG 图标）、"登录"按钮
4. 注册 Tab：username 输入框、password 输入框（密码强度进度条）、确认密码输入框、email 输入框（可选）、"注册"按钮
5. 表单验证状态：focus → `border-color: var(--accent)` + `box-shadow: 0 0 0 3px var(--accent-dim)`；error → `border-color: var(--red)` + 红色错误文字（UX-DR25）；API 返回错误时在表单顶部显示 inline 错误提示
6. 登录/注册成功后：覆盖层以反向动画下滑消失；**按接缝约定 A**：Widget JS 立即断开当前 WS 连接并携带新 JWT token 重连；输入框自动获焦（UX-DR20）
7. 覆盖层开启时 focus trap（Tab 键在覆盖层内部循环）（UX-DR23）；右上角关闭 SVG 按钮（点击或 Escape 键关闭）
8. 所有输入框 focus outline：`outline: 2px solid var(--accent); outline-offset: 2px`（UX-DR17）；最小点击区域 44×44px（UX-DR18）

---

#### Story 3.3: 用户 Profile & 声誉积分系统

**As a** 注册用户，
**I want** 拥有包含声誉积分和徽章的用户 Profile，管理员能够调整我的声誉，
**So that** 社区活跃贡献能够得到可见的认可。

**Acceptance Criteria:**

1. GET /api/users/:id/profile：返回 `{username, avatarColor, countryCode, reputation, role, lastSeenAt, reputationBadge}`；无需 JWT（公开接口）
2. ReputationTier GORM 模型：id, label, min_score, color(hex), sort_order；安装时 `internal/database/seed.go` 插入默认4段（无徽章: min_score=0；BRONZE: min_score=10；SILVER: min_score=50；GOLD: min_score=100）
3. GET /api/admin/reputation-tiers：列出全部积分段，需管理员 JWT
4. POST/PUT/DELETE /api/admin/reputation-tiers：创建/更新/删除积分段，需管理员 JWT；至少保留 1 段，低于此数量时返回 400
5. PUT /api/admin/users/:id/reputation `{delta: N}`：在原分值基础上加减 N，记录到 User.reputation；需管理员 JWT（FR-2.4）
6. PUT /api/admin/users/:id `{banned_at: "2026-11-08T..."}` 或 `{banned_at: null}`：封禁/解封用户（FR-5.4）；封禁时立即关闭该用户所有活跃 WS 连接（遍历 HubRegistry 踢出）
7. lastSeenAt 在每次 WS connect 和 disconnect 时更新（`UPDATE users SET last_seen_at=NOW() WHERE id=?`）
8. reputationBadge 字段根据 reputation 分值和当前 ReputationTier 表计算，取 max min_score ≤ reputation 的那一段

---

#### Story 3.4: 消息国旗 & 声誉徽章显示

**As a** 注册用户，
**I want** 在每条消息旁看到发送者的国家国旗和声誉徽章，
**So that** 我能直观了解对话的参与者身份和可信度。

**Acceptance Criteria:**

1. 每条消息广播 payload 新增字段：`countryCode`（ISO 3166-1 alpha-2），`reputationBadge`（"ANON"/"BRONZE"/"SILVER"/"GOLD" 或管理员自定义标签），`badgeColor`（来自 ReputationTier.color）
2. Widget 渲染消息时，在发送者名称左侧显示国旗 emoji（从 countryCode 转换为区域指示符号 emoji，如 "CN" → "🇨🇳"）；countryCode 为空时不显示国旗（FR-1.7）
3. 声誉徽章显示在发送者名称右侧（FR-1.7，UX-DR4）：
   - ANON：`background: var(--surface-3)`，`color: var(--fg-muted)`
   - BRONZE（默认 10+）：`background: rgba(205,127,50,0.2)`，`color: #cd7f32`
   - SILVER（默认 50+）：`background: rgba(148,163,184,0.2)`，`color: #94a3b8`
   - GOLD（默认 100+）：`background: rgba(251,191,36,0.2)`，`color: #fbbf24`
4. 徽章显示文字标签（非仅颜色），满足颜色非唯一信息传达（UX-DR24 无障碍）
5. 自己的消息布局：右对齐，时间戳 · 徽章 · 国旗 · 名称 · 头像色圆点（右侧）
6. 匿名用户：头像色圆点为 `var(--surface-3)` 灰色，显示名为服务端生成的英文昵称，徽章为 "ANON"
7. 注册用户头像：24px 圆形，背景色为 User.avatarColor（来自消息 payload），内显示用户名首字母（大写）
8. 管理员自定义了 ReputationTier 后，新消息立即使用更新后的徽章颜色/标签（消息广播时查库）

---

### Epic 4：社交互动功能

#### Story 4.1: @ 提及通知系统

**As a** 注册用户，
**I want** 被 @ 提及时在折叠的 Widget 上看到红色通知角标并听到提示音，展开后自动定位到提及消息，
**So that** 我不会错过直接对我说的消息。

**Acceptance Criteria:**

1. 消息内容经过 `@username` 正则匹配（`@([a-zA-Z0-9_]+)`），识别被提及用户（FR-2.5）
2. 服务端：从在线 Client 中找到匹配用户名的注册用户连接，向其推送 `{"type":"mention","messageID":<uint>,"channelSlug":"...","senderName":"...","preview":"<前50字>"}`（FR-2.5）
3. Widget 折叠态：`mentionCount` 自增，折叠按钮右上角显示红色角标（`scale(0)→scale(1)` 弹出动画）（FR-2.5，UX-DR6，KF-2）
4. 提及音效：页面首次用户交互后，收到 mention 事件时播放内置音效文件（0.3s 短提示音，base64 内嵌到 JS 中），仅遵从浏览器自动播放策略（FR-2.5）
5. Widget 内设置图标（SVG 齿轮，UX-DR3）→ 展开设置面板 → 声音通知开关（toggle，默认开启）；用户关闭后不播放音效，设置持久化到 localStorage（FR-2.5）
6. Widget 展开后：消息列表自动滚动到提及消息；被提及气泡显示 `box-shadow: 0 0 0 2px var(--accent)` 高亮边框
7. 消息中 `@username` 文字样式：他人气泡中 `color: var(--accent)` + `font-weight: 600`；自己气泡中 `color: rgba(255,255,255,0.9)` + `font-weight: 600`
8. mentionCount 在用户查看到提及消息后清零：消息滚动进入视口时触发 IntersectionObserver 清零（FR-2.5）

---

#### Story 4.2: Emoji 选择器 & 语音输入

**As a** 注册用户，
**I want** 使用 Emoji 选择器快速插入表情，并能用语音输入内容，
**So that** 我能更生动地表达自己。

**Acceptance Criteria:**

1. 输入区左侧 Emoji 按钮（SVG 笑脸图标，UX-DR3）；点击后在输入框上方弹出 Emoji 面板（`position: absolute; bottom: 100%`）（FR-1.6）
2. Emoji 面板：6×5 网格，30 个常用 emoji；每个 emoji 44×44px 触摸区域（UX-DR18）；点击 emoji 插入到输入框光标位置，面板关闭（FR-1.6）
3. 点击面板外部区域（document click）关闭 Emoji 面板（UX-DR 通用交互模式）
4. 语音输入按钮（SVG 麦克风图标）：仅当 `navigator.mediaDevices` 和 `window.SpeechRecognition`（或 `webkitSpeechRecognition`）可用时显示（FR-1.6）
5. 点击语音按钮：请求麦克风权限（如未授权）；启动 SpeechRecognition continuous mode；interim 结果实时追加到输入框；静默 3s 或再次点击停止
6. 麦克风权限被拒绝：语音按钮永久隐藏，不显示任何错误提示（FR-1.6）
7. `@` 提及补全（FR-1.5，UX-DR11）：输入 `@` 后出现在线用户下拉列表（取 Hub clients 中注册用户）；键盘上下键选择，Enter 确认插入 `@username`，Escape 关闭；下拉列表最多显示 10 个匹配用户
8. 所有按钮 SVG 图标满足 UX-DR3 规范；最小点击区域 44×44px（UX-DR18）

---

#### Story 4.3: 消息举报功能

**As a** 注册用户，
**I want** 对违规消息进行举报，举报后消息在我的视图中消失，
**So that** 我能帮助维护频道内容质量。

**Acceptance Criteria:**

1. 右键点击（桌面端）或长按（移动端，400ms）他人消息气泡：显示操作菜单「回复 | @ 他 | 举报」（UX-DR19，FR-1.8）；自己的消息不显示该菜单（无"举报"选项）
2. 操作菜单定位：在气泡上方弹出，`position: absolute`；点击菜单外部区域关闭
3. 点击"举报"：POST /api/reports `{"messageID": N}`；JWT 鉴权必须，匿名用户调用返回 401（FR-1.8）
4. Report GORM 模型：id, message_id(FK), reporter_id(FK), status(enum: pending/handled/ignored), created_at
5. 举报成功：消息在举报者视图中立即本地隐藏（从 DOM 移除，不影响其他用户视图）；不显示遮挡 overlay（FR-1.8）
6. 重复举报同一消息：API 返回 HTTP 409，前端静默处理（不显示错误弹窗）
7. 举报成功：页面底部短暂显示 toast "已举报"（2s 后自动消失），`background: var(--surface-3)`，SVG ✓ 图标
8. 未登录用户的菜单中不显示"举报"选项（FR-1.8 注：仅注册用户可举报）

---

#### Story 4.4: 私信 DM 后端

**As a** 注册用户，
**I want** 能向其他注册用户发送私信并持久化记录，
**So that** 我们可以在公共频道之外进行一对一交流。

**Acceptance Criteria:**

1. DirectMessage GORM 模型：id, sender_id(FK), receiver_id(FK), content, is_read(default false), created_at（FR-2.6）
2. POST /api/dm/:recipientID：创建 DM 消息，content HTML 转义，需 JWT；Bot 用户（role="bot"）作为收件人返回 HTTP 403（FR-6.3）
3. GET /api/dm/:otherUserID：返回双方 DM 对话历史，按 created_at ASC，分页 50 条/页（?page=N）；需 JWT
4. DM 发送时，若收件人在线（遍历 HubRegistry 中的活跃 Client）：推送 WS 事件 `{"type":"dm_received","senderID":<uint>,"senderName":"...","preview":"<前30字>"}`（FR-2.6）
5. Widget FAB（折叠按钮）DM 未读角标：收到 dm_received 且 Widget 未在 DM 抽屉视图时，蓝色角标（`background: #3b82f6`）叠加显示在 FAB，与红色 @ 角标并列（FR-2.6）
6. PUT /api/dm/:conversationUserID/read：将与该用户的全部未读消息标记为 is_read=true；清空对应 DM 未读角标计数；需 JWT
7. v1 明文存储：content 字段直接存储 HTML 转义后的文本（AD-13）
8. 管理员 PUT /api/admin/settings `{"dmEncryption": true/false}`：写入 config 字段，不实际加密，返回 `{"message":"此功能将在后续版本启用"}`（AD-13）

---

#### Story 4.5: 私信抽屉 UI

**As a** 注册用户，
**I want** 点击用户名后能打开私信抽屉，在不离开频道的情况下与对方私聊，
**So that** 私密交流体验流畅无缝。

**Acceptance Criteria:**

1. 点击频道消息列表中他人的用户名/头像：弹出 mini 操作菜单「发私信 | @ 他」
2. 点击"发私信"：DM 抽屉从 Widget 左侧 `translateX(-100%)→translateX(0)` 滑入，280ms `cubic-bezier(0.4,0,0.2,1)`；宽度等于 Widget 宽度；遮住频道聊天视图但不销毁它（UX-DR7，FR-2.6）
3. 抽屉头部：左侧"← 返回"按钮（SVG 箭头，UX-DR3）+ 对方用户名 + 24px 头像色圆点 + 在线状态点（绿色=在线，灰色=离线）
4. DM 消息列表：同频道消息气泡样式（自己右对齐橙色，对方左对齐深色），历史通过 GET /api/dm/:userID 加载，向上滚动分页
5. 输入区：与频道输入区样式一致，Enter 发送，POST /api/dm/:recipientID
6. "← 返回"按钮：DM 抽屉以反向动画 `translateX(0)→translateX(-100%)` 收起，频道聊天视图完整恢复（UX-DR7）
7. DM 抽屉开启时 focus trap（Tab 循环在抽屉内）（UX-DR23）；"← 返回"最小点击区域 44×44px（UX-DR18）
8. 抽屉打开时调用 PUT /api/dm/:userID/read 清除未读计数，FAB 蓝色角标消失

---

### Epic 5：管理后台

#### Story 5.1: 管理后台框架 & Dashboard

**As a** 管理员，
**I want** 访问一个安全的管理后台，看到实时系统统计 Dashboard，
**So that** 我能快速了解 ChatSphere 实例的运行状态。

**Acceptance Criteria:**

1. 管理后台路由：`/{admin.path}/*`（来自 config.yaml，默认 /admin）；JWT 中间件保护所有子路由；未认证访问重定向到 `/{admin.path}/login`（FR-5.1）
2. POST /{admin.path}/login：接受 `{username, password}` 表单（HTML form）；验证 JWT；设置 `httpOnly; Secure; SameSite=Strict` Cookie；重定向到 Dashboard（AD-8）
3. 所有 Admin 页面服务端 `html/template` 渲染，无 React/Vue/HTMX（AD-8）
4. 侧边栏：180px 宽，含图标 + 文字导航项：Dashboard、Channels、Messages、Users、Reports、Settings；活跃项 `background: var(--accent-dim)`，`color: var(--accent)`；SVG 图标（UX-DR3）
5. 移动端（<520px）侧边栏折叠为 44px 图标条，悬停/焦点时 tooltip 显示文字（UX-DR22）
6. 顶部栏：48px 高，左侧品牌名，中间当前页面标题，右侧管理员用户名 + 登出按钮
7. Dashboard 页面（FR-5.1）：4 个统计卡片——总在线用户数、今日消息数、活跃频道数、待处理举报数；数字以千位分隔符格式化
8. 统计数据通过 GET /api/admin/stats 获取（需管理员 JWT），页面加载时 JS fetch，每 30s 轮询刷新

---

#### Story 5.2: 频道管理

**As a** 管理员，
**I want** 在管理后台对频道进行完整 CRUD 操作，
**So that** 我能控制社区频道的结构和行为。

**Acceptance Criteria:**

1. Channels 页面：表格列出所有频道（名称、slug、是否默认、是否启用、在线人数、消息总数、操作列）（FR-5.2）
2. "新建频道"按钮：展开 inline 表单（服务端 partial 渲染，原生 JS fetch 局部插入 DOM，AD-8）
3. 频道表单字段覆盖 FR-3.2 全部配置：名称（必填）、slug（必填，auto-generate from name，可手动编辑，唯一校验）、描述、是否为默认频道（radio，同一时刻只能有一个默认频道）、是否启用（toggle）、最大消息长度（数字，默认 500）、发言速率限制（秒，默认 0=禁用）、是否允许匿名用户（toggle，默认 true）、欢迎消息（textarea）、Telegram 链接（URL，可选）
4. slug 自动生成规则：name 转小写，空格替换为 `-`，去掉非 `[a-z0-9-]` 字符；保存时服务端验证唯一性，重复时 inline 返回 `{"error":"slug already taken"}`（FR-3.1）
5. POST /api/admin/channels、PUT /api/admin/channels/:id、DELETE /api/admin/channels/:id；均需管理员 JWT
6. 列表行内快速启用/停用 toggle（JS fetch PATCH 局部更新）
7. 删除确认弹窗（`window.confirm()`...不，需要 inline confirm dialog）：`"删除频道将删除其所有消息，此操作不可撤销"`；用户点击"确认删除"后执行 DELETE
8. 频道配置变更立即生效：新建 WS 连接时读取最新配置，已连接用户在下条消息时使用新 rate_limit/max_length 配置

---

#### Story 5.3: 消息管理

**As a** 管理员，
**I want** 搜索筛选消息并软删除违规内容，
**So that** 我能保持频道内容整洁。

**Acceptance Criteria:**

1. Messages 页面：分页列表（20条/页），列：时间（相对+绝对 tooltip）、频道、发送者、内容预览（50字，截断 "..."）、状态标签、操作（FR-5.3）
2. 过滤控件：频道下拉（全部/各频道）、关键词文本搜索、用户名文本搜索、时间范围（from/to date input）；过滤参数通过 URL query string 传递（GET /api/admin/messages?channel=&keyword=&username=&from=&to=&page=N）
3. GET /api/admin/messages 接口：服务端根据过滤条件构建 GORM Where 子句；keyword 和 username 均使用 LIKE `%keyword%`；需管理员 JWT（FR-5.3）
4. 软删除单条：DELETE /api/admin/messages/:id → `is_deleted=true`；已删除消息在 Widget 中不再广播，历史加载时过滤（FR-5.3）
5. 批量软删除：列表每行有 checkbox；顶部"批量删除"按钮（选中 ≥1 条时启用）；POST /api/admin/messages/batch-delete `{ids: [N,N,...]}`（FR-5.3）
6. 已删除消息在管理列表中以 ~~strikethrough~~ 样式 + "已删除"红色标签显示；"恢复"按钮：PUT /api/admin/messages/:id `{is_deleted:false}`
7. 软删除不删除数据库行，保留原始内容用于审计（FR-5.3）

---

#### Story 5.4: 用户管理

**As a** 管理员，
**I want** 搜索用户并管理其封禁状态和声誉积分，
**So that** 我能维护社区秩序和激励体系。

**Acceptance Criteria:**

1. Users 页面：分页列表（20条/页），列：用户名（带头像色点）、国家国旗、声誉积分、徽章、角色、最后在线时间、状态、操作（FR-5.4）
2. 顶部搜索框：实时 JS fetch（300ms debounce）GET /api/admin/users?q=keyword；按 username LIKE `%q%` 筛选
3. 封禁/解封：用户行内 toggle 开关；点击触发 PUT /api/admin/users/:id（banned_at 写入/清空）；确认弹窗 inline："封禁后该用户将无法在所有频道发言，确认封禁？"；封禁后立即踢出所有活跃 WS 连接（FR-5.4）
4. 封禁用户：用户名旁显示"已封禁"红色徽章
5. 声誉积分调整：用户行内点击积分数字 → inline 编辑变为 input；输入 delta 值（如 +10 或 -5）→ 失焦/Enter 触发 PUT /api/admin/users/:id/reputation `{delta: N}`（FR-5.4）
6. 在线用户：用户名旁绿色圆点指示器（通过 HubRegistry 实时查询）
7. "查看消息"链接：跳转到 Messages 页面并预填 username 过滤条件（FR-5.4）
8. GET /api/admin/users 分页接口；需管理员 JWT

---

#### Story 5.5: 举报管理

**As a** 管理员，
**I want** 在三次点击内处理举报并对消息采取行动，
**So that** 举报处理流程高效不积压。

**Acceptance Criteria:**

1. Reports 页面：三个标签页（待处理 / 已处理 / 已忽略），待处理 tab 显示数量角标（FR-5.5）
2. 每行：举报时间、举报者用户名、被举报消息内容（前80字）、消息发送者、操作
3. "删除消息并处理"操作（1次点击）：PUT /api/admin/reports/:id `{action:"handle"}` → 同时设置 message.is_deleted=true + report.status=handled；不需要二次确认弹窗（≤2次点击，FR-5.5）
4. "忽略举报"操作（1次点击）：PUT /api/admin/reports/:id `{action:"ignore"}` → report.status=ignored；无需确认（FR-5.5）
5. 操作后对应行从当前 tab 消失，自动移入相应 tab（JS 局部 DOM 更新，无整页刷新）
6. 批量忽略：多选 checkbox + "批量忽略"按钮；POST /api/admin/reports/batch-ignore `{ids: [N,...]}`
7. 侧边栏 Reports 导航项旁显示待处理举报数量角标（来自 GET /api/admin/stats 的 pending_reports 字段）
8. GET /api/admin/reports?status=pending|handled|ignored；需管理员 JWT（FR-5.5）

---

#### Story 5.6: Widget 嵌入代码生成器

**As a** 管理员，
**I want** 通过可视化工具生成和复制 Widget 嵌入代码，
**So that** 我能将 ChatSphere 轻松嵌入到任何网站而不需要查阅文档。

**Acceptance Criteria:**

1. Settings → Embed Code 页面：配置控件 + 实时代码预览（FR-5.6）
2. 配置控件：服务器地址（文本输入，默认当前 host origin）、频道选择（已启用频道 dropdown）、展示位置（bottom-right/bottom-left radio，图示说明）
3. 控件任意改动时，下方代码块实时更新（无需点击生成，JS oninput/onchange）（FR-5.6）
4. 生成的最小嵌入代码：`<script src="{server}/widget/chatsphere.js" data-server="{server}" data-channel="{slug}" data-position="{position}"></script>`
5. 代码块：等宽字体，语法高亮（keyword/attribute/string 三色），一键复制按钮（SVG 复制图标）
6. 复制按钮：`navigator.clipboard.writeText()`；失败时 fallback 选中代码文本；成功后图标变为 ✓，文字变为"已复制!"，2s 后恢复（FR-5.6）
7. 页面同时展示完整 HTML 示例（包含 `<!DOCTYPE html>` 骨架，代码块中标注插入位置）以供参考

---

### Epic 6：内容审核 & 个性化定制

#### Story 6.1: 屏蔽词规则配置

**As a** 管理员，
**I want** 为每个频道配置精确匹配和正则表达式屏蔽词规则，
**So that** 频道能自动过滤违禁内容。

**Acceptance Criteria:**

1. FilterRule GORM 模型：id, channel_id(FK), rule_type(enum: exact/regex), pattern, is_enabled(default true), created_at（FR-6.1）
2. Settings → Channels → 编辑频道 → 屏蔽词规则 Tab：显示该频道的规则列表（FR-6.1）
3. 规则列表：每行显示规则类型标签（"精确" 灰色 / "正则" 橙色）、规则内容（`font-family: monospace`）、启用/停用 toggle、删除按钮
4. 新增规则表单：规则类型 radio（精确匹配 | 正则表达式）、规则内容 input；正则规则以 `/pattern/flags` 格式输入（FR-6.1）
5. 正则规则提交前：前端用 JS 执行 `new RegExp(pattern)` 验证语法；语法错误时 input 边框变红 + 错误文字 "正则表达式语法错误"
6. 服务端保存时同样验证 `regexp.Compile(pattern)`；失败返回 HTTP 400（FR-6.1）
7. GET/POST /api/admin/channels/:id/rules；PUT /api/admin/channels/:id/rules/:ruleID；DELETE /api/admin/channels/:id/rules/:ruleID；均需管理员 JWT
8. 编译后的正则表达式在 MessageService 启动时预编译并缓存（`sync.Map[ruleID→*regexp.Regexp]`），避免每条消息重复编译

---

#### Story 6.2: 消息过滤 & Bot 系统消息

**As a** 访客，
**I want** 违禁内容被自动拦截，频道内 Bot 以系统消息提示过滤事件，
**So that** 频道内容始终符合社区规范，同时用户能理解为什么某条消息没有出现。

**Acceptance Criteria:**

1. MessageService.CreateMessage 在写入 MySQL 前检查内容（FR-6.2）：加载频道启用的 FilterRule；精确匹配：`strings.Contains(strings.ToLower(content), strings.ToLower(pattern))`；正则匹配：使用缓存编译的 `*regexp.Regexp`
2. 命中任意规则：消息以 `is_filtered=true` 写入 MySQL（保留审计）但不广播，不写入 Redis（FR-6.2）
3. Bot 用户：安装时 `seed.go` 创建 User{username: "ChatSphere Bot", role: "bot", avatar_color: "#6366f1"}；不可登录（无 password_hash）（FR-6.3）
4. 命中过滤后：Bot 以 ChatSphere Bot 身份向频道广播系统消息 `{"type":"system_message","content":"该消息因包含违禁词已被过滤","isBotMessage":true}`（FR-6.2）
5. Widget 渲染系统消息：居中显示，`color: var(--fg-subtle)`，`font-size: 12px`，无气泡背景，前缀 Bot SVG 图标（FR-6.3，UX 系统消息样式）
6. Bot 不可被 @ 提及（@ 解析时跳过 role="bot" 的用户）；POST /api/dm/{botUserID} 返回 HTTP 403（FR-6.3）
7. 已过滤消息在管理后台 Messages 列表中可见，状态标签"已过滤"（橙色）；管理员"恢复"操作：`is_filtered=false` → 重新广播到频道（FR-6.4）
8. 管理员"永久删除"已过滤消息：`is_deleted=true`，content 覆写为 `[已删除]`（FR-6.4）

---

#### Story 6.3: Widget 主题色定制

**As a** 管理员，
**I want** 在管理后台修改 Widget 强调色并立即预览效果，保存后新访客立即看到新主题，
**So that** Widget 能与我的网站品牌风格保持一致。

**Acceptance Criteria:**

1. Settings → Widget 主题页面（FR-5.8）：当前 accent color 色块展示 + 原生 `<input type="color">` 颜色选择器（默认值 `#f5a623`）
2. 颜色变化时（oninput 事件）< 100ms 内更新右侧 Widget 预览面板（FR-5.8）：预览面板渲染折叠按钮（56×56px 圆形）+ 一条他人消息气泡 + 一条自己消息气泡 + 输入区域；全部使用选中颜色的 CSS token（UX-DR5）
3. `--accent-dim = rgba(r,g,b,0.15)`，`--accent-glow = rgba(r,g,b,0.35)`：由 hex 颜色 JS 实时计算（FR-5.8）
4. 保存按钮：POST /api/admin/settings/theme `{"accentColor":"#xxxxxx"}`；服务端将颜色存储到 DB settings 表（key: "accent_color"）（FR-5.8）
5. 颜色保存后：服务端在 `GET /widget/chatsphere.js` 响应中注入 CSS 变量覆盖块（`<accent-css-inject>` → `[data-cs="root"]{--accent:#xxx;--accent-dim:rgba(...);--accent-glow:rgba(...)}`）到 JS 文件头部（FR-5.8）
6. 已连接客户端使用旧颜色直到页面刷新；新访客连接时获取注入了新颜色的 JS 文件（FR-5.8）
7. "恢复默认"按钮：重置为 `#f5a623`，同步更新预览和保存

---

#### Story 6.4: 声誉积分规则 & @ 提醒音效

**As a** 管理员，
**I want** 自定义声誉积分分段的阈值/标签/颜色，并可上传自定义 @ 提醒音效，
**So that** 社区激励体系能反映我们的社区文化。

**Acceptance Criteria:**

1. Settings → 声誉积分规则页面：列出所有 ReputationTier，可编辑每段的 label（文本）、color（原生 color picker）、min_score（数字输入）（UX-DR4，FR-2.4）
2. 每行显示 badge 预览（使用当前 label 和 color 渲染样式，实时更新）
3. 新增积分段：表单行内追加新行；删除按钮（至少保留 1 段，低于此数量时 toast 提示 "至少需要一个声誉等级"）
4. 保存：POST /api/admin/reputation-tiers `{tiers: [{label, min_score, color, sort_order}, ...]}`（全量替换）；服务端写入 DB；Widget 广播的新消息立即使用更新后的 tier 计算徽章
5. Settings → @ 提醒音效区域：当前音效标签（"默认内置音效" 或已上传文件名）；"播放预览"按钮（SVG 播放图标）（FR-2.5）
6. 上传自定义音效：`<input type="file" accept="audio/*">`；POST /api/admin/settings/sound（multipart/form-data）；服务端存储至 `data/sounds/notification.mp3`（或原格式）；Widget 在初始化时检查 GET /api/settings/sound-url 确定使用内置或自定义 URL（FR-2.5）
7. "恢复默认音效"按钮：DELETE /api/admin/settings/sound；Widget 回退到内置 base64 音效
8. Settings → 私信加密区域（AD-13）：toggle 开关 + 文字说明"端对端加密将在后续版本启用"；toggle 状态变化时 PUT /api/admin/settings `{"dmEncryption":true/false}`，仅写入 config flag，不改变任何数据存储行为；toggle 旁固定显示灰色 "v2 功能"标签
