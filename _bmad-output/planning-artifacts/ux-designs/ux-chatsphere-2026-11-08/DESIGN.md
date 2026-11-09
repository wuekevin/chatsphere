---
title: "ChatSphere Design System"
status: final
created: 2026-11-08
updated: 2026-11-08
colors:
  # Dark theme (default)
  background:        "#0e1017"
  surface:           "#13151e"
  surface-elevated:  "#1a1d28"
  surface-high:      "#222638"
  border:            "rgba(255,255,255,0.07)"
  border-bright:     "rgba(255,255,255,0.12)"
  accent:            "#f5a623"
  accent-dim:        "rgba(245,166,35,0.15)"
  accent-glow:       "rgba(245,166,35,0.35)"
  fg:                "#f0f0f8"
  fg-muted:          "rgba(240,240,248,0.45)"
  fg-subtle:         "rgba(240,240,248,0.22)"
  green:             "#34d399"
  red:               "#f87171"
  amber:             "#fbbf24"
  # Light theme overrides
  light-background:       "#f5f1e8"
  light-surface:          "#ffffff"
  light-surface-elevated: "#faf7f1"
  light-surface-high:     "#f0ece0"
  light-accent:           "#d4891a"
  light-accent-dim:       "rgba(212,137,26,0.12)"
  light-fg:               "#12141c"
  light-fg-muted:         "rgba(18,20,28,0.5)"
  light-fg-subtle:        "rgba(18,20,28,0.25)"
typography:
  primary: "Inter, system-ui, -apple-system, sans-serif"
  mono:    "JetBrains Mono, 'Cascadia Code', monospace"
  scale:
    display: "44px / 700 / -0.02em"
    h1:      "28px / 700 / -0.02em"
    h2:      "20px / 600 / -0.015em"
    h3:      "16px / 600 / -0.01em"
    body:    "14px / 400 / 0"
    small:   "12px / 400 / 0"
    label:   "11px / 600 / 0.08em / uppercase"
    micro:   "10px / 500 / 0.04em"
    bubble:  "12.5px / 400 / 0 / line-height 1.5"
rounded:
  sm:   "8px"
  md:   "14px"
  lg:   "20px"
  bubble-other: "16px 16px 16px 4px"
  bubble-self:  "16px 16px 4px 16px"
  full: "9999px"
spacing:
  xs:   "4px"
  sm:   "8px"
  md:   "12px"
  lg:   "16px"
  xl:   "24px"
  2xl:  "32px"
  3xl:  "48px"
components:
  widget-width:  "340px"
  widget-height: "480px (expanded), 56px (collapsed)"
  header-height: "56px"
  input-height:  "44px"
  avatar-sm:     "24px"
  avatar-md:     "32px"
  badge-height:  "16px"
---

# ChatSphere Design System

## Brand & Style

ChatSphere 的视觉语言参考金帝影视（jdmv.net）——一款面向华人影视观众的流媒体平台——将其「深黑底 + 琥珀橙强调」的设计基因移植到嵌入式聊天场景中。整体调性：**沉浸、高密度、夜间友好**。

**核心关键词：** 沉浸（Immersive）· 精准（Precise）· 温暖（Warm）

Widget 是一个与宿主页面共存的小型信息层，不抢占视觉焦点，但在激活时传递强烈的品牌感。橙色是唯一强调色，所有交互反馈、发送按钮、自己的消息气泡、在线状态、active 徽章均使用同一橙色，形成一致的视觉节奏。

**Admin 后台**采用相同调色板，给站长一种「同一产品不同视角」的统一感，而非另立一套企业蓝风格。

**禁止：**
- 使用 Emoji 作为 UI 图标（消息内容中的 emoji 文字除外）
- 使用纯紫色、蓝色作为主强调色
- 添加装饰性渐变背景色块（仅限头部细线光晕 `::before`）
- 圆角超过 `radius-lg (20px)` 的卡片

---

## Colors

### 暗色主题（默认）

| Token | 值 | 用途 |
|-------|----|------|
| `--bg` | `#0e1017` | 页面/宿主容器背景 |
| `--surface` | `#13151e` | Widget 主体背景、卡片 |
| `--surface-2` | `#1a1d28` | 次级面板（输入区、侧边栏） |
| `--surface-3` | `#222638` | 输入框内部、高亮行 |
| `--border` | `rgba(255,255,255,0.07)` | 通用分隔线 |
| `--border-bright` | `rgba(255,255,255,0.12)` | 卡片外框、输入框 |
| `--accent` | `#f5a623` | 强调色（唯一主色）|
| `--accent-dim` | `rgba(245,166,35,0.15)` | 强调色背景填充 |
| `--accent-glow` | `rgba(245,166,35,0.35)` | 阴影发光效果 |
| `--fg` | `#f0f0f8` | 主文字 |
| `--fg-muted` | `rgba(240,240,248,0.45)` | 次要文字、占位符 |
| `--fg-subtle` | `rgba(240,240,248,0.22)` | 分隔线标签、时间戳 |
| `--green` | `#34d399` | 在线状态、成功状态 |
| `--red` | `#f87171` | 错误、举报角标 |
| `--amber` | `#fbbf24` | Gold 声誉徽章 |

### 亮色主题

底色使用暖米色 `#f5f1e8`（非纯白，轻微奶油感），强调色降饱和至 `#d4891a`，避免亮底下橙色刺眼。

### 管理员自定义主题

Widget 的 `--accent` / `--accent-dim` / `--accent-glow` 三个 token 由管理员在后台「Widget 主题」页面配置。服务端生成一段内联 CSS 注入 widget 初始化脚本。其余结构色（背景、边框）不可自定义（v1）。

---

## Typography

主字体 **Inter**，代码/哈希值使用 **JetBrains Mono**，通过 Google Fonts 加载。

| 层级 | 大小 | 字重 | 行高 | 间距 | 场景 |
|------|------|------|------|------|------|
| Display | 44px | 700 | 1.15 | -0.02em | 安装向导大标题 |
| H1 | 28px | 700 | 1.2 | -0.02em | 页面主标题 |
| H2 | 20px | 600 | 1.3 | -0.015em | 章节标题 |
| H3 | 16px | 600 | 1.4 | -0.01em | 卡片标题 |
| Body | 14px | 400 | 1.6 | 0 | 管理后台正文 |
| Bubble | 12.5px | 400 | 1.5 | 0 | Widget 气泡消息正文 |
| Small | 12px | 400 | 1.5 | 0 | 辅助说明 |
| Label | 11px | 600 | 1 | 0.08em | 大写区块标签，`text-transform: uppercase` |
| Micro | 10px | 500 | 1 | 0.04em | 徽章文字、时间戳 |

**排版规则：**
- 标题类使用 `text-wrap: balance`
- 气泡文字使用 `word-break: break-word` 防止长 URL 溢出
- 管理后台正文行宽建议 ≤ 65 字符

---

## Layout & Spacing

间距基础单位 **4px**，所有间距取 4 的倍数。

**Widget 布局（340px 宽）：**
- 头部：高 56px，左右 padding 16px
- 消息列表：左右 padding 12px，气泡间 `gap: 8px`
- 输入区：高 44px（含外边距），左右 padding 12px
- 气泡最大宽度：72% 容器宽

**Admin 后台布局：**
- 侧边栏宽：180px（移动端折叠为 44px）
- 内容区：左右 padding 16px
- 卡片 grid `gap: 10px`
- 顶栏高：48px

**Gutter：** 任何表面在 ≤ 400px 视口下最小侧边距 16px，使用 `max(16px, env(safe-area-inset-*, 0px))`。

---

## Elevation & Depth

三级投影体系，从低到高叠加层次感：

| 层级 | 使用场景 | Shadow |
|------|---------|--------|
| Level 0 | 页面背景 | 无 |
| Level 1 | 卡片、侧边栏 | `0 2px 8px rgba(0,0,0,.3)` |
| Level 2 | Admin 预览卡片 | `0 16px 48px rgba(0,0,0,.4)` |
| Level 3 | Widget 浮层 | `0 24px 80px rgba(0,0,0,.55), 0 0 0 1px rgba(245,166,35,.1)` |

**发光效果：** 仅用于强调色相关元素（发送按钮、active 步骤圆圈、自己的气泡）。发光值为 `box-shadow: 0 2px 10px var(--accent-glow)`。头部细线光晕通过 `::before` 伪元素实现，不用 `box-shadow`。

---

## Shapes

- Widget 卡片：`border-radius: 20px`（`radius-lg`）
- 消息气泡（他人）：`border-radius: 16px 16px 16px 4px`（左下角收紧，指向发言者）
- 消息气泡（自己）：`border-radius: 16px 16px 4px 16px`（右下角收紧）
- 输入框：`border-radius: 10px`
- 按钮/徽章：`border-radius: 8px`（`radius-sm`）
- 头像/在线点：`border-radius: 50%`

---

## Components

### Widget 气泡消息

```
[他人]  头像(24px) · 昵称(label) · 国旗 · 声誉徽章 · 时间戳
        └── 气泡(其他色，左对齐)

[自己]                        时间戳 · 声誉徽章 · 国旗 · 昵称
                              气泡(accent色，右对齐) ──┘ 头像(24px，accent色边框)
```

- 自己的气泡背景使用 `--accent`，文字白色
- @提及文字在自己的气泡中：`rgba(255,255,255,0.75)` + 600 字重
- @提及文字在他人气泡中：`var(--accent)` + 600 字重
- 高亮消息（被@时）：气泡外加 `box-shadow: 0 0 0 2px var(--accent)`

### 声誉徽章

| 段位 | 显示文字 | 颜色 |
|------|---------|------|
| 无徽章 | — | — |
| 铜（10+积分）| BRONZE | `rgba(205,127,50,0.2)` / `#cd7f32` |
| 银（50+）| SILVER | `rgba(148,163,184,0.2)` / `#94a3b8` |
| 金（100+）| GOLD | `rgba(251,191,36,0.2)` / `#fbbf24` |
| 匿名 | ANON | `var(--surface-3)` / `var(--fg-muted)` |

管理员可自定义阈值、颜色和文案；以上为系统默认值。

### SVG 图标规范

**仅使用 SVG 线条图标**，不使用 Emoji 或任何图片图标。

- viewBox: `0 0 16 16`
- stroke-width: `1.5`
- stroke-linecap / linejoin: `round`
- fill: `none`（需要填充的圆点除外）
- 颜色继承 `currentColor`，通过父元素的 `color` 控制
- 非激活态：`opacity: 0.6`；激活态：`opacity: 1`

### Widget 折叠按钮

圆形悬浮按钮，56×56px，`background: var(--accent)`，底部右对齐，`box-shadow: 0 4px 16px var(--accent-glow)`。未读消息数以红色圆形角标（右上角）显示，≥100 显示 "99+"。

### 私信抽屉

从 widget 左侧（`transform: translateX(-100%)`）滑入，宽度等于 widget 宽，覆盖频道视图。顶部有返回按钮（← 对话者用户名）。与主视图共享底部输入区。

### 安装向导步骤圆圈

- 已完成：绿色背景 + 黑色 ✓
- 进行中：`--accent` 背景 + 白色数字 + `box-shadow: 0 0 12px var(--accent-glow)`
- 待完成：`--surface-3` 背景 + 细边框 + `--fg-muted` 数字

---

## Do's and Don'ts

**Do:**
- 在强调色元素上统一使用 `#f5a623`；不要用不同深浅的橙色混搭
- SVG 图标始终继承 `currentColor`，不硬编码颜色
- 气泡最大宽度 72%，让两侧气泡各有呼吸空间
- 使用 `gap` 布局间距，不用 `margin` 堆叠

**Don't:**
- 不要在 widget 内使用 Emoji 作为 UI 功能图标
- 不要给 `--bg` / `--surface` 增加不透明度变体（会破坏深色叠加逻辑）
- 不要在气泡以外的场景使用带方向的异形圆角
- 不要在非交互元素上添加 hover 态
- 不要将管理后台的卡片阴影（Level 2）用在 widget 内部组件上
