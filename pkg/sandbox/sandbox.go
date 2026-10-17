package sandbox

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Manager 沙箱管理器
type Manager struct {
	runtime string // "gvisor", "seccomp", "none"
	mu      sync.Mutex
	runs    map[string]*RunRecord
}

type RunRecord struct {
	ID        string
	Lang      string
	Command   string
	StartedAt time.Time
	Status    string
}

func NewManager() *Manager {
	rt := "seccomp"
	if hasGVisor() {
		rt = "gvisor"
	}
	return &Manager{
		runtime: rt,
		runs:    make(map[string]*RunRecord),
	}
}

func hasGVisor() bool {
	_, err := exec.LookPath("runsc")
	return err == nil
}

// Run 在沙箱中执行命令
func (m *Manager) Run(ctx context.Context, lang, command string, args []string) (map[string]interface{}, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	record := &RunRecord{
		ID:        fmt.Sprintf("run-%d", time.Now().UnixNano()),
		Lang:      lang,
		Command:   command,
		StartedAt: time.Now(),
		Status:    "running",
	}
	m.mu.Lock()
	m.runs[record.ID] = record
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		record.Status = "completed"
		m.mu.Unlock()
	}()

	// gVisor 路径
	if m.runtime == "gvisor" && lang != "" {
		return m.runWithGVisor(ctx, lang, command, args)
	}

	// seccomp 路径
	if m.runtime == "seccomp" {
		return m.runWithSeccomp(ctx, lang, command, args)
	}

	// 降级：直接执行（带超时）
	return m.runDirect(ctx, lang, command, args)
}

func (m *Manager) runWithGVisor(ctx context.Context, lang, command string, args []string) (map[string]interface{}, error) {
	// v0.1.0: 模拟 gVisor 调用，实际部署需 runsc run 配置
	cmdArgs := []string{"run", "--rootless", "-bundle", "/tmp/te-sandbox/" + lang}
	if command != "" {
		cmdArgs = append(cmdArgs, command)
		cmdArgs = append(cmdArgs, args...)
	}
	return m.execContext(ctx, "runsc", cmdArgs)
}

func (m *Manager) runWithSeccomp(ctx context.Context, lang, command string, args []string) (map[string]interface{}, error) {
	// v0.1.0: 使用 seccomp-bpf 过滤系统调用（通过 go-seccomp-bpf 或 syscall）
	// 这里用带受限环境变量的子进程模拟
	return m.runDirect(ctx, lang, command, args)
}

func (m *Manager) runDirect(ctx context.Context, lang, command string, args []string) (map[string]interface{}, error) {
	var cmd *exec.Cmd
	if command == "" {
		// 直接执行语言运行时
		rt := langRuntime(lang)
		if rt == "" {
			return nil, fmt.Errorf("unsupported language: %s", lang)
		}
		cmd = exec.CommandContext(ctx, rt, args...)
	} else {
		cmd = exec.CommandContext(ctx, command, args...)
	}
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=/tmp", "LANG=C"}
	cmd.Dir = "/tmp"

	out, err := cmd.CombinedOutput()
	result := map[string]interface{}{
		"stdout":   string(out),
		"exitCode": 0,
		"runtime":  m.runtime,
	}
	if err != nil {
		result["exitCode"] = 1
		result["error"] = err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result["exitCode"] = exitErr.ExitCode()
		}
	}
	return result, nil
}

func (m *Manager) execContext(ctx context.Context, name string, args []string) (map[string]interface{}, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	result := map[string]interface{}{
		"stdout":   string(out),
		"exitCode": 0,
		"runtime":  m.runtime,
	}
	if err != nil {
		result["exitCode"] = 1
		result["error"] = err.Error()
	}
	return result, nil
}

func langRuntime(lang string) string {
	mapping := map[string]string{
		"go":         "go",
		"python":     "python3",
		"py":         "python3",
		"nodejs":     "node",
		"node":       "node",
		"javascript": "node",
		"rust":       "rustc",
		"java":       "java",
		"powershell": "pwsh",
		"ps":         "pwsh",
	}
	if rt, ok := mapping[strings.ToLower(lang)]; ok {
		return rt
	}
	return ""
}

// IsAvailable 检查沙箱运行时是否可用
func (m *Manager) IsAvailable() bool {
	return m.runtime != "none"
}

// Records 返回运行记录
func (m *Manager) Records() []*RunRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*RunRecord
	for _, r := range m.runs {
		out = append(out, r)
	}
	return out
}

// WrapCommand 为任意命令包装沙箱执行器
func WrapCommand(ctx context.Context, lang, entry string, sandboxed bool) (*exec.Cmd, error) {
	if !sandboxed {
		return exec.CommandContext(ctx, entry), nil
	}
	if runtime.GOOS == "linux" && hasGVisor() {
		return exec.CommandContext(ctx, "runsc", "run", "--rootless", "-bundle", filepath.Join("/tmp/te-sandbox", lang), entry), nil
	}
	return exec.CommandContext(ctx, entry), nil
}
