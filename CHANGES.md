## v4.32.1 (2026-10-18)

### 历史优点整合
- 整合此前所有版本的稳定性与可靠性改进
- 统一错误处理与日志规范
- 强化安全基线（输入校验、边界检查）

### 本版改进
- 代码质量：补充边界条件注释与文档
- 运维友好：CHANGES.md 持续维护，版本追踪更清晰
- 新增文档：docs/DEPLOYMENT.md 部署指南

### 测试
- 现有测试套件通过验证

# TarsEnterprise Changelog

## v4.33.0 (2026-10-18)

### 集成版本 — 历史优点整合 + 健康检查与配置验证

本次版本将 TarsEnterprise 自 v0.2.0 至 v4.32.0 的所有核心能力整合到统一的运行时可观测体系中，并新增两项面向运维与小白用户的增强功能：

**整合的历史优点（自 v0.2.0 ~ v4.32.0）**
- **知识库（KnowledgeBase）** — 文档分片、嵌入向量、语义搜索、按公司类型初始化
- **Obsidian 双向同步** — Vault 同步引擎、Frontmatter 解析、WriteBack 回写
- **语音纠错（VoiceCorrection）** — 文本纠错接口，支持上下文感知
- **NATS 消息传输** — TE↔TSG 双向指令通道、状态同步、内存 fallback
- **TSG 客户端 SDK（pkg/tsgclient）** — 7 端点封装、角色令牌验证、健康检查
- **自动更新（tarsupdate）** — GitHub Release 检测、稳定版本 fallback、SHA256 校验、平台适配
- **MCP Hub** — 多服务器注册、工具调用代理
- **沙箱管理（Sandbox）** — gvisor/seccomp 运行时、命令黑名单校验
- **角色管理（Role）** — 公司类型角色矩阵、跨网关身份绑定
- **mTLS 零信任网络接入（v4.32.0）** — 自签名/自定义 CA 双模式、证书动态重载、HTTP 中间件
- **安装向导（Wizard）** — 首次启动 TUI 引导
- **国际化错误（i18n）** — 小白化 HTTP 错误提示
- **安全中间件（security）** — 限流、CORS、安全头部、请求体大小限制、管理员认证
- **指标监控（metrics）** — Prometheus 格式导出
- **OpenClaw 适配器** — 平台互通适配层
- **配置合并（config/merge）** — 多模块配置去冲突合并

**新增功能**

- **统一健康检查端点 `/api/v1/health`**
  - 聚合 10 个子系统实时状态：mTLS、TSG Bridge、KnowledgeBase、MCP Hub、Sandbox、Metrics、Role Manager、VoiceCorrection、Config
  - 每个子系统返回 `status`（healthy/degraded/unhealthy/unknown）、`message`、`friendly_msg`（小白化描述）
  - 整体状态自动降级：任一 unhealthy → 整体 unhealthy；任一 degraded → 整体 degraded
  - 保留旧 `/health` 端点向后兼容

- **启动时配置验证（小白化提示）**
  - `config.Validate()` 检查 5 类常见问题：AdminToken 缺失、mTLS 配置不一致、TSG Bridge 地址缺失、CompanyType 未设置、Metrics 端口异常
  - 每条错误包含 `friendly_msg`（人话）+ `suggestion`（修复建议）
  - 验证失败不阻塞启动，仅日志提示，降低新手门槛

**测试**
- `pkg/health` 新增 4 个测试用例（Registry、Healthy、Unhealthy、SimpleCheck）
- `pkg/config` 新增 5 个验证场景测试（全绿配置、AdminToken 缺失、mTLS 文件缺失、CompanyType 空、Metrics 端口异常）
- `go test -race ./...` 全量通过

## v4.32.0 (2026-10-18)

### 零信任网络接入 — mTLS 双向认证

- **CertificateManager** (`pkg/mtls/manager.go`)
  - 服务端证书、CA 证书池与客户端证书验证的全生命周期管理
  - 支持从文件加载和内存 PEM 两种初始化方式
  - 证书动态重载（Reload）
  - 服务端证书信息查询（用于健康检查与状态展示）

- **自签名 CA + 自定义 CA 双模式**
  - `self_signed`：自动生成 CA + 服务端证书，开箱即用
  - `custom_ca`：从指定文件路径加载企业自有 CA 与服务端证书

- **证书过期自动拒绝 + 小白化错误提示**
  - 服务端证书过期 → "服务器安全证书已过期，请联系管理员更新证书后再试"
  - 客户端未提供证书 → "连接需要客户端证书，请确保已正确配置并上传您的身份证书"
  - 客户端证书过期 → "您的身份证书已过期，请联系管理员重新签发证书"
  - 客户端证书尚未生效 → "您的身份证书尚未生效，请确认系统时间是否正确，或联系管理员"

- **HTTP 中间件**
  - `ClientCertMiddleware`：从 TLS 连接提取客户端证书并注入请求上下文
  - `RequireClientCert`：强制要求客户端证书（HTTP 495）

- **API 端点**
  - `GET /api/v1/mtls/status` — 查询当前 mTLS 配置状态与客户端身份信息

- **启动支持**
  - `cmd/te` 根据配置自动选择明文或 mTLS 模式启动
  - TLS 1.3 最低版本，强制双向认证

- **测试**
  - 20 个测试用例，`go test -race ./pkg/mtls/...` 全绿
