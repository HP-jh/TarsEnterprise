package lang

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// EnvInfo 本地环境信息
type EnvInfo struct {
	Go         bool              `json:"go"`
	Python     bool              `json:"python"`
	NodeJS     bool              `json:"nodejs"`
	Rust       bool              `json:"rust"`
	Java       bool              `json:"java"`
	PowerShell bool              `json:"powershell"`
	Versions   map[string]string `json:"versions"`
}

func Detect() *EnvInfo {
	e := &EnvInfo{Versions: make(map[string]string)}
	// Go 永远可用（自研 fallback）
	e.Go = true
	e.Versions["go"] = runtime.Version()

	if v, ok := checkVersion("python3", "--version"); ok {
		e.Python = true
		e.Versions["python"] = v
	} else if v, ok := checkVersion("python", "--version"); ok {
		e.Python = true
		e.Versions["python"] = v
	}

	if v, ok := checkVersion("node", "--version"); ok {
		e.NodeJS = true
		e.Versions["nodejs"] = v
	}

	if v, ok := checkVersion("rustc", "--version"); ok {
		e.Rust = true
		e.Versions["rust"] = v
	}

	if v, ok := checkVersion("java", "-version"); ok {
		e.Java = true
		e.Versions["java"] = v
	}

	if v, ok := checkVersion("pwsh", "--version"); ok {
		e.PowerShell = true
		e.Versions["powershell"] = v
	} else if v, ok := checkVersion("powershell", "-Command", "$PSVersionTable.PSVersion"); ok {
		e.PowerShell = true
		e.Versions["powershell"] = v
	}

	return e
}

func (e *EnvInfo) Summary() []string {
	var out []string
	if e.Go {
		out = append(out, fmt.Sprintf("Go %s", e.Versions["go"]))
	}
	if e.Python {
		out = append(out, fmt.Sprintf("Python %s", e.Versions["python"]))
	}
	if e.NodeJS {
		out = append(out, fmt.Sprintf("Node.js %s", e.Versions["nodejs"]))
	}
	if e.Rust {
		out = append(out, fmt.Sprintf("Rust %s", e.Versions["rust"]))
	}
	if e.Java {
		out = append(out, fmt.Sprintf("Java %s", e.Versions["java"]))
	}
	if e.PowerShell {
		out = append(out, fmt.Sprintf("PowerShell %s", e.Versions["powershell"]))
	}
	return out
}

func checkVersion(name string, args ...string) (string, bool) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", false
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "", false
	}
	// Usually version output is on first line
	lines := strings.SplitN(v, "\n", 2)
	return strings.TrimSpace(lines[0]), true
}
