# TarsEnterprise — 万物皆 MCP 🚀

> **"Everything is MCP"** — Enterprise Gateway for AI Agents, MCP Servers & Multi-Client Orchestration

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/HP-jh/TarsEnterprise)](https://goreportcard.com/report/github.com/HP-jh/TarsEnterprise)

TarsEnterprise (TE) 是企业级 AI 网关，支持多进程并行接入多个客户端、AI 通过 OpenAI Function Calling 直接调用工具、一个工具作用于多个客户端、多工具并发发起。基于 OpenClaw 优化，具备智能体沙箱环境，可直接覆盖 OpenClaw 主流漏洞。

## ✨ 核心特性

- **多进程并行客户端接入**：企业级高并发网关，端口 `8888`
- **万物皆 MCP**：OpenClaw MCP 化接入，工具即 MCP Server
- **TSG ↔ TE 互通**：个人版与企业版双向消息投递（mTLS + 角色 Token）
- **分布式/边缘路由**：NOB 内置路由、TE-GW 网关调度、OpenAI Realtime Agent 直调
- **6 种公司类型**：学校 / 互联网 / KPI考核 / 咨询 / 金融 / 政企，角色-资源-动作三轴权限
- **智能体沙箱**：gVisor + seccomp + 网络隔离，复杂工具全部沙箱化
- **TOP 应用统计**：智能体级 Prometheus 指标 + Grafana 看板
- **多语言适配**：Python / Node.js / Rust / PowerShell / Java + Go 完整最低级 fallback
- **极致小白化**：6 步首次配置向导，错误信息全人话化

## 🏢 公司类型

| 类型 | 标识 | 典型角色 | 核心资源 |
|------|------|---------|---------|
| 学校 | `education` | 教师、学生、家长 | 课程、成绩、家校 |
| 互联网公司 | `tech` | 产品、研发、运营、HR | 项目、迭代、事故 |
| KPI 考核公司 | `kpi` | 部门经理、小组长、员工 | 日报、周报、月报 |
| 咨询公司 | `consulting` | 顾问、项目经理、客户 | 项目、客户、计费 |
| 金融机构 | `finance` | 风控、合规、审计 | 交易、账户、报告 |
| 政企机构 | `gov` | 科员、处长、审批人 | 公文、流程、权限 |

## 🚀 快速开始

```bash
# 1. 下载对应平台二进制
curl -LO https://github.com/HP-jh/TarsEnterprise/releases/latest/download/te-v0.1.0-linux-amd64
chmod +x te-v0.1.0-linux-amd64

# 2. 启动网关（首次启动自动进入配置向导）
./te-v0.1.0-linux-amd64

# 3. 打开 http://localhost:8888 完成 6 步配置
```

## 🛡️ 安全

- mTLS 双向认证（TSG ↔ TE 互通）
- 角色 Token 二次确认机制
- 智能体沙箱：gVisor runsc + seccomp BPF
- OpenClaw 漏洞覆盖：258 CVE 签名库
- 审计哈希链：全操作不可篡改日志

## 📊 监控

Prometheus 端点：`http://localhost:8888/metrics`

```bash
curl http://localhost:8888/metrics
# te_agent_calls_total, te_agent_latency_seconds, te_agent_errors_total
```

Grafana 看板导入 `dashboards/grafana.json`。

## 🌐 交互方式

1. **NOB 内置路由**：沿用 TSG 自研 NOB 协议
2. **TE-GW 网关调度**：企业版自研调度协议
3. **OpenAI Realtime Agent**：双工语音 + Function Call 直调

## 📦 架构

```
┌─────────────────────────────────────────────┐
│           TarsEnterprise Gateway            │
│              (Port 8888)                     │
├─────────────┬─────────────┬─────────────────┤
│  Company    │   MCP Hub   │   TSG Bridge    │
│  Schema     │   (Servers) │   (mTLS/NATS)   │
├─────────────┼─────────────┼─────────────────┤
│  Role/RBAC  │  Sandbox    │   Metrics       │
│  (KingdoM)  │  (gVisor)   │  (Prometheus)   │
├─────────────┴─────────────┴─────────────────┤
│         Lang Adapters (Py/Node/Rust/...)     │
└─────────────────────────────────────────────┘
```

## 🤝 与 TSG 互通

```
TSG (Personal)  ←──mTLS──→  TE (Enterprise)
     │                          │
   NATS JetStream  ←──State Sync──→  NATS JetStream
```

- 个人版 → 企业版：消息投递（带角色 Token）
- 企业版 → 个人版：反向命令（带审计 + 二次确认）
- State 同步：NATS JetStream 或 Redis Streams

### 真闭环示例（v0.3.0）

启动 TE 和 TSG 两个进程：

```bash
# Terminal 1: 启动 TE（已配置 TSG 地址）
./te-v0.3.0-linux-amd64 -config te.json

# Terminal 2: 启动 TSG（个人网关，端口 8443）
./tsg -port 8443
```

演示完整命令往返：

```bash
# 1. 绑定角色（企业角色 → TSG 令牌）
curl -X POST http://localhost:8888/api/v1/tsg/role/bind \
  -d '{"user_id":"alice","role_id":"admin","tsg_token":"tok123"}'
# → {"bound":true,"user_id":"alice","role_id":"admin"}

# 2. 企业向个人网关发指令
curl -X POST http://localhost:8888/api/v1/tsg/command \
  -d '{"tool_id":"file_search","params":{"query":"*.go"},"role_token":"tok123"}'
# → {"cmd_id":"cmd-xxx","status":"delivered"}

# 3. 推送状态同步
curl -X POST http://localhost:8888/api/v1/tsg/state/sync \
  -d '{"state":{"mode":"active","version":"v0.3.0"}}'
# → {"status":"synced","timestamp":1234567890}

# 4. 拉取本地状态
curl http://localhost:8888/api/v1/tsg/state
# → {"state":{"mode":"active"},"last_sync":1234567890,"sync_age":5.2}

# 5. 拉取 TSG 下发指令列表
curl http://localhost:8888/api/v1/tsg/commands
# → {"commands":[],"count":0}

# 6. 跨网关角色验证
curl -X POST http://localhost:8888/api/v1/role/check \
  -d '{"user_id":"alice","resource_id":"dashboard","action":"read","role_token":"tok123"}'
# → {"allowed":true,"local":false,"cross":true,"role_token":true}
```

实时事件流：

```bash
# SSE 事件流（每 5s 心跳 + 新指令通知）
curl -N http://localhost:8888/api/v1/tsg/events
# → {"event":"connected","version":"v0.3.0"}
# → {"event":"commands_updated","count":1}
```

## 🔄 自动更新（v0.3.0）

TarsEnterprise 内置自动更新引擎，启动后可随时检查并应用最新版本。

```bash
# 检查是否有新版本（公开端点，无需认证）
curl http://localhost:8888/api/v1/update/check
# → {"has_update":false,"current":"v0.3.0","latest":"v0.3.0"}

# 应用更新（需要管理员 Token）
curl -X POST http://localhost:8888/api/v1/update/apply \
  -H "X-Admin-Token: your-admin-token"
# → {"ok":true,"message":"update applied, restart required","from":"v0.3.0","to":"v0.3.1"}
```

**特性**
- GitHub Release 自动比对（semver）
- SHA256SUMS 校验 + 原子替换
- 解析失败自动回退 `~/.tars/stable_versions.json` 缓存的稳定版本
- 旧版本保留 7 天，支持手动回滚
- 跨 7 平台统一更新（Linux / macOS / Windows / Android）

## 📄 License

Apache License 2.0 — 详见 [LICENSE](LICENSE)

## Changelog

### v1.0.1
- Version bump with stability improvements
