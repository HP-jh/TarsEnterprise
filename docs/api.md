# TarsEnterprise API 文档

## 基础信息

- 基础地址：`http://localhost:8888`
- 所有 API 返回 `Content-Type: application/json`
- 错误信息已人话化，无需猜测技术黑话

## 通用响应格式

成功：
```json
{"status": "ok", ...}
```

错误：
```json
{"error_code": "METHOD_NOT_ALLOWED", "message": "这个接口需要用 POST 方法访问..."}
```

## 端点列表

### 健康检查

```
GET /health
```

返回服务状态和版本号。

### 指标

```
GET /metrics
```

Prometheus 格式指标。

### 公司类型

```
GET /api/v1/company/schema?type=tech
```

获取指定公司类型的角色、资源、动作定义。

### 角色权限检查

```
POST /api/v1/role/check
```

请求体：
```json
{"user_id": "alice", "resource_id": "dashboard", "action": "read"}
```

### 工具调用

```
POST /api/v1/tools/call
```

请求体：
```json
{"tool_id": "openclaw-1", "params": {"query": "*.go"}, "client_id": "tsg-001"}
```

### MCP 调用

```
POST /api/v1/mcp/call
```

请求体：
```json
{"server": "openclaw", "tool": "search", "params": {"q": "hello"}}
```

### 沙箱执行

```
POST /api/v1/sandbox/run
```

请求体：
```json
{"lang": "python", "command": "echo", "args": ["hello"]}
```

**安全策略**：危险命令（rm、format、shutdown 等）和路径遍历会被自动拦截。

### TSG 互通

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/tsg/role/bind` | POST | 绑定跨网关角色 |
| `/api/v1/tsg/command` | POST | 向 TSG 发送指令 |
| `/api/v1/tsg/commands` | GET | 获取 TSG 下发的指令 |
| `/api/v1/tsg/events` | GET | SSE 事件流 |
| `/api/v1/tsg/state/sync` | POST | 状态同步 |
| `/api/v1/tsg/state` | GET | 拉取本地状态 |

### 自动更新

```
GET /api/v1/update/check
POST /api/v1/update/apply  （需要 X-Admin-Token）
```

### 知识库

```
POST /api/v1/kb/search
GET /api/v1/kb/docs
POST /api/v1/kb/import
```

### Obsidian 同步

```
POST /api/v1/obsidian/sync
POST /api/v1/obsidian/writeback
```

### 语音纠错

```
POST /api/v1/voice/correct
```

请求体：
```json
{"text": "这是一个侧试"}
```
