# Epic 3 Context: 用户账号 & 身份系统

## Goal

访客可以在 Widget 内完成注册/登录（不跳转页面），获得持久账号身份；消息旁展示国旗、声誉徽章（ANON/BRONZE/SILVER/GOLD）；管理员可在后台调整声誉积分。

## Stories

- Story 3.1: 用户注册 & JWT 登录（后端 API）
- Story 3.2: Widget 内登录/注册覆盖层
- Story 3.3: 用户 Profile & 声誉积分系统
- Story 3.4: 消息国旗 & 声誉徽章显示

## Technical Decisions

**JWT 传递方式（WS）：**
浏览器 WebSocket API 不支持自定义 header，故注册用户连接 WS 时通过 query param 传递 token：
`/ws/:channel?token=<JWT>`
OptionalAuth 中间件同时检查 Authorization header 和 `?token` query param。

**声誉分段（4 级徽章）：**
- is_anonymous = true → ANON（无徽章）
- is_anonymous = false && reputation 0–99 → BRONZE 🥉
- is_anonymous = false && reputation 100–499 → SILVER 🥈
- is_anonymous = false && reputation 500+ → GOLD 🥇

**Widget JWT 本地存储：**
JWT 存储于宿主页面的 localStorage，key 为 `cs:jwt:<server_origin>`，支持同一页面多频道实例共享同一登录态。

**国家码来源：**
注册时从请求头 CF-IPCountry（Cloudflare CDN）或 X-Country 读取；无法检测时为空，用户不可手动设置（v1 简化）。
