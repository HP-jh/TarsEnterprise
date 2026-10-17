# TarsEnterprise 配置指南

## 配置文件位置

- Linux/macOS：`~/.tarsenterprise/config.json`
- Windows：`%USERPROFILE%\.tarsenterprise\config.json`

## 完整配置示例

```json
{
  "first_run": false,
  "company_type": "tech",
  "roles": ["admin", "manager", "employee"],
  "clients": [
    {"id": "tsg-001", "type": "tsg_personal", "endpoint": "localhost:8443", "token": ""}
  ],
  "tools": [
    {"id": "openclaw-1", "name": "OpenClaw", "lang": "go", "entry": "openclaw", "clients": ["tsg-001"], "sandboxed": true}
  ],
  "mcp": {
    "enabled": true,
    "servers": ["openclaw"],
    "openclaw": true
  },
  "tsg_bridge": {
    "enabled": true,
    "address": "localhost:8443",
    "mtls_cert": "",
    "mtls_key": "",
    "nats_url": "nats://localhost:4222"
  },
  "sandbox": {
    "enabled": true,
    "runtime": "gvisor",
    "seccomp": true
  },
  "metrics": {
    "enabled": true,
    "port": 8888
  },
  "admin_token": "change-me-in-production"
}
```

## 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `company_type` | string | 是 | 公司类型：`education`、`tech`、`kpi`、`consulting`、`finance`、`gov` |
| `admin_token` | string | 否 | 管理员令牌，执行敏感操作（如应用更新）时需要 |
| `tsg_bridge.enabled` | bool | 否 | 是否启用与 TSG 个人版的互通 |
| `sandbox.enabled` | bool | 否 | 是否启用沙箱 |
| `sandbox.runtime` | string | 否 | 沙箱运行时：`gvisor`、`seccomp`、`none` |

## 首次启动

不指定配置文件启动时，会自动进入配置向导：

```bash
./te-v0.4.0-linux-amd64
# 打开 http://localhost:8888 完成 6 步向导
```

## 安全配置建议

1. **生产环境必须设置 `admin_token`**，长度建议 ≥ 32 位随机字符串
2. **关闭不需要的端口**：如果只在内网使用，绑定 `127.0.0.1:8888`
3. **启用 mTLS**：TSG 互通时配置 `mtls_cert` 和 `mtls_key`
4. **沙箱不要设为 `none`**：生产环境至少使用 `seccomp`
