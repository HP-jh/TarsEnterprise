# TarsEnterprise 常见问题

## Q: 启动后无法访问 http://localhost:8888？

A: 检查防火墙是否放行了 8888 端口。如果绑定的是 127.0.0.1，只有本机能访问；如果需要局域网访问，启动时指定 `-port 0.0.0.0:8888`。

## Q: 提示"管理员令牌尚未配置"怎么办？

A: 在 `config.json` 中添加 `"admin_token": "你的随机字符串"`，然后重启服务。建议用 `openssl rand -hex 32` 生成。

## Q: 沙箱执行被拦截了，如何调整？

A: 当前版本内置了危险命令黑名单（rm、format、shutdown 等）。如需放行特定命令，请联系管理员修改 `pkg/sandbox/guard.go` 中的黑名单规则。

## Q: 与 TSG 个人版互通失败？

A: 按以下顺序排查：
1. TSG 是否已启动并在监听对应地址
2. `config.json` 中 `tsg_bridge.address` 是否正确
3. 如果启用了 mTLS，证书和密钥是否匹配
4. NATS 是否已启动（如果使用 NATS 模式）

## Q: 支持哪些公司类型？

A: 6 种内置类型：`education`（学校）、`tech`（互联网）、`kpi`（KPI 考核）、`consulting`（咨询）、`finance`（金融）、`gov`（政企）。

## Q: 自动更新失败怎么办？

A: 检查网络是否能访问 GitHub。如果无法访问，系统会自动回退到 `~/.tars/stable_versions.json` 中缓存的版本。也可以手动下载二进制替换。
