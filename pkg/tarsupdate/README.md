# tars-update

独立 Go module：Tars 系列项目的自动更新引擎。

## 功能

- **版本检查**：查询 GitHub Releases API，semver 比对
- **下载校验**：下载对应平台二进制 + SHA256SUMS.txt，哈希校验
- **原子替换**：rename 原子替换，旧二进制备份保留 7 天
- **失败回退**：GitHub API 不可用时回退到本地 `stable_versions.json`
- **跨平台**：linux / darwin / windows / android（amd64 / arm64）
- **多仓库**：同一模块支持 TarsSecureGuard / TarsWingGuard / TarsEnterprise 等 6 仓库

## 使用

```go
import "github.com/HP-jh/tars-update"

u := update.NewUpdater("twg", "v0.1.0", "HP-jh", "TarsWingGuard")

// Check
result := u.Check()
if result.HasUpdate {
    fmt.Printf("New version available: %s\n", result.Latest.String())
}

// Apply
if err := u.Apply(result); err != nil {
    log.Fatal(err)
}

// Rollback
if err := u.Rollback(); err != nil {
    log.Fatal(err)
}
```

## CLI 集成

在 `wingctl` 中集成：

```bash
wing update check    # 检查新版本
wing update apply    # 下载并应用更新
wing update rollback # 回退到上一版本
```

## 安全

- SHA256 校验必做
- 旧二进制备份保留，支持手动 rollback
- 升级日志写 `~/.tars/update.log`
- 极致档用户可配置 2FA 确认（由调用方实现）

## License

Apache-2.0
