package sandbox

import (
	"strings"
	"testing"
)

func TestValidateCommand_AllowsSafe(t *testing.T) {
	if err := ValidateCommand("echo", []string{"hello"}); err != nil {
		t.Fatalf("expected safe command to pass, got %v", err)
	}
}

func TestValidateCommand_BlocksBlacklist(t *testing.T) {
	blocked := []string{"rm", "del", "format", "shutdown", "dd"}
	for _, cmd := range blocked {
		if err := ValidateCommand(cmd, []string{"-rf", "/"}); err == nil {
			t.Errorf("expected %q to be blocked", cmd)
		} else if !strings.Contains(err.Error(), "危险命令") && !strings.Contains(err.Error(), "安全策略拦截") {
			t.Errorf("expected friendly block message for %q, got %v", cmd, err)
		}
	}
}

func TestValidateCommand_BlocksPathTraversal(t *testing.T) {
	if err := ValidateCommand("cat", []string{"../../etc/passwd"}); err == nil {
		t.Error("expected path traversal to be blocked")
	}
	if err := ValidateCommand("cat", []string{"/etc/shadow"}); err == nil {
		t.Error("expected sensitive path to be blocked")
	}
}

func TestValidateCommand_BlocksAbsoluteSensitivePath(t *testing.T) {
	if err := ValidateCommand("/root/.bashrc", nil); err == nil {
		t.Error("expected absolute sensitive path to be blocked")
	}
}
