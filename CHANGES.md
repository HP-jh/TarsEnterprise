# TarsEnterprise 变更日志

## v0.5.0 — 零信任网络接入（mTLS 双向认证）

**发布日期：** 2026-10-18

### 新增功能
- **mTLS 双向认证配置档** (`config.MTLSConfig`)：支持通过配置文件统一管理证书路径、自动轮换策略、零信任角色白名单
- **证书自动轮换逻辑** (`pkg/mtls/rotation.go`)：
  - `RotationManager` 后台监控证书有效期
  - 在过期前自动重新生成并持久化自签名证书（默认 30 天有效期，过期前 7 天轮换）
  - 支持从现有 PEM 文件加载，不存在时自动生成
  - 每小时检查一次，轮换失败仅记录日志、不中断服务
- **零信任身份验证** (`pkg/mtls/zerotrust.go`)：
  - TLS 握手层强制 `RequireAndVerifyClientCert`
  - `VerifyPeerCertificate` 回调校验客户端证书 CN/OU 是否在允许角色列表中
  - 无有效客户端证书的连接直接拒绝（人话化错误提示）
  - HTTP 中间件 `RequireMTLSMiddleware` 供应用层二次防护
- **服务端集成** (`cmd/te/main.go`)：
  - mTLS 启用时自动切换为 `tls.Listen` 启动 HTTPS
  - 自动轮换守护线程与主服务生命周期绑定

### 安全强化
- 最低 TLS 版本锁定为 1.3
- 无客户端证书 = 零信任拒绝，不降级到普通 HTTP
- 自签名证书文件权限：cert 0644 / key 0600 / ca 0644

### 测试覆盖
- `pkg/mtls/rotation_test.go`：初始化、轮换、加载已有证书、过期检测、起停、解析
- `pkg/mtls/zerotrust_test.go`：enforce 策略、身份提取、中间件行为

---

## v0.4.1 — 维护更新

- 版本号同步

## v0.4.0 — 完整性能评价体系

- 基准测试 harness、回归基线、趋势看板
