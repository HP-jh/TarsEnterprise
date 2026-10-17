# TarsEnterprise 故障排查

## 1. 服务无法启动

### 症状：端口被占用
```
listen tcp :8888: bind: address already in use
```

**解决**：
```bash
# 查找占用端口的进程
lsof -i :8888
# 或更换端口启动
./te-v0.4.0-linux-amd64 -port 8889
```

### 症状：配置文件解析失败
```
parse config: invalid character
```

**解决**：检查 `~/.tarsenterprise/config.json` 是否为合法 JSON，可用 `python3 -m json.tool ~/.tarsenterprise/config.json` 验证。

## 2. API 返回 429（请求太快）

**原因**：v0.4.0 新增了速率限制，每个 IP 默认 2 QPS / 10 burst。

**解决**：降低请求频率，或等待 1 秒后重试。Response Header 中会有 `Retry-After`。

## 3. API 返回 403（危险命令被拦截）

**原因**：沙箱安全策略拦截了危险命令。

**解决**：确认命令不在黑名单中。如需临时测试，可改用本地直接执行（不推荐生产环境）。

## 4. 与 TSG 互通时状态不同步

**排查步骤**：
1. 检查 TE 和 TSG 是否都能访问 NATS 服务器
2. 查看 TE 日志中是否有 `[TSG] mTLS configured but CA missing`
3. 调用 `/api/v1/tsg/state` 查看本地状态时间戳
4. 调用 `/api/v1/tsg/commands` 查看是否有堆积指令

## 5. 知识库搜索无结果

**排查步骤**：
1. 确认 `/api/v1/kb/docs` 返回的文档列表不为空
2. 如为空，先调用 `/api/v1/kb/import` 导入文档
3. 检查 `company_type` 是否与导入时一致

## 6. 更新后服务异常

**回滚步骤**：
1. 停止当前服务
2. 恢复旧版本二进制（旧版本保留在同级目录，文件名带时间戳）
3. 从 `~/.tars/stable_versions.json` 读取推荐稳定版本
4. 重新启动
