# Deferred Work

- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-项目基础启动-配置检测.md`
  summary: installHandler 中 db.Create(ch) 的错误被静默吞掉，安装成功响应可能掩盖频道创建失败
  evidence: 预存在 bug，不在 Story 1.1 范围；installer.go:214 `db.Create(ch)` 结果未检查

- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-项目基础启动-配置检测.md`
  summary: generateSecret() 中 rand.Read 错误未处理，极端情况下可能生成全零密钥
  evidence: 预存在 bug；installer.go:280 `rand.Read(b)` 返回值被忽略

- source_spec: `_bmad-output/implementation-artifacts/spec-1-3-管理员账号与站点信息配置-安装向导步骤-3-4.md`
  summary: 安装失败后无返回/重试按钮，用户只能刷新页面重新输入所有字段
  evidence: 预存在于步骤5面板；Story 1.4 重构安装流程（SSE）时一并处理

- source_spec: `_bmad-output/implementation-artifacts/spec-1-3-管理员账号与站点信息配置-安装向导步骤-3-4.md`
  summary: startInstall() 进度日志在 API 调用前伪造"Running migrations"等步骤完成信息
  evidence: 预存在；Story 1.4 改 SSE 实时推送时修复

- source_spec: `_bmad-output/implementation-artifacts/spec-1-3-管理员账号与站点信息配置-安装向导步骤-3-4.md`
  summary: 安装成功后 Embed Widget 链接错误指向 /admin 而非嵌入代码页面
  evidence: 预存在复制粘贴错误；Story 1.4 完善安装完成页面时修复

- source_spec: `_bmad-output/implementation-artifacts/spec-1-3-管理员账号与站点信息配置-安装向导步骤-3-4.md`
  summary: admin-link 硬编码 http:// 协议，HTTPS 部署或反向代理场景下链接错误
  evidence: 预存在；Story 1.4 处理安装完成后跳转逻辑时修复

- source_spec: `_bmad-output/implementation-artifacts/spec-1-3-管理员账号与站点信息配置-安装向导步骤-3-4.md`
  summary: channel slug 仅检查非空，允许全连字符值（如 ---）作为有效 slug
  evidence: 低影响；后端或 Story 2.1 频道 API 层可加正则校验

- source_spec: `_bmad-output/implementation-artifacts/spec-2-1-频道数据层-服务端-api.md`
  summary: ChannelService/MessageService 以 concrete type 注入 handler，阻碍测试替换
  evidence: RegisterChat 接受 *service.ChannelService 而非 interface；测试时无法注入 mock，需要在 Story 2.2 或专项重构中定义 service interface

- source_spec: `_bmad-output/implementation-artifacts/spec-2-1-频道数据层-服务端-api.md`
  summary: repository 层返回未包装的原始 GORM/DB 错误，调用方无法区分 not-found 以外的错误类型
  evidence: ListByChannel/ListActive 直接返回 gorm 错误；需系统性 error domain 设计，建议在 service 层引入包装工具
