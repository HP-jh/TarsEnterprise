package sandbox

import (
	"fmt"
	"path/filepath"
	"strings"
)

// commandBlacklist 危险命令黑名单（大小写不敏感匹配）
var commandBlacklist = []string{
	"rm", "del", "rd", "format", "mkfs", "dd",
	"shutdown", "reboot", "halt", "poweroff",
	">", ">>", "|",
}

// pathBlacklist 危险路径模式（防止目录遍历和敏感文件访问）
var pathBlacklist = []string{
	"..", "~", "/etc/passwd", "/etc/shadow", "/root", "/proc", "/sys",
}

// ValidateCommand 检查命令和参数是否包含危险操作
func ValidateCommand(command string, args []string) error {
	check := strings.ToLower(command)
	for _, blk := range commandBlacklist {
		if check == blk || strings.HasPrefix(check, blk+" ") {
			return fmt.Errorf("检测到危险命令 '%s'，已被安全策略拦截。如需执行该操作，请联系管理员调整沙箱策略", command)
		}
	}
	for _, arg := range args {
		lowArg := strings.ToLower(arg)
		for _, pblk := range pathBlacklist {
			if strings.Contains(lowArg, pblk) {
				return fmt.Errorf("参数包含受限路径 '%s'，已被安全策略拦截", arg)
			}
		}
	}
	// 防止绝对路径指向敏感目录
	if filepath.IsAbs(command) {
		lowCmd := strings.ToLower(command)
		for _, pblk := range pathBlacklist {
			if strings.Contains(lowCmd, pblk) {
				return fmt.Errorf("命令路径包含受限目录，已被安全策略拦截")
			}
		}
	}
	return nil
}
