package sandbox

import (
	"context"
	"testing"
	"time"
)

func TestManager_Run(t *testing.T) {
	m := NewManager()
	if !m.IsAvailable() {
		t.Skip("sandbox runtime not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := m.Run(ctx, "go", "go", []string{"version"})
	if err != nil {
		t.Logf("run error: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}
	if result["runtime"] == "" {
		t.Error("runtime not set")
	}
}

func TestManager_Records(t *testing.T) {
	m := NewManager()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = m.Run(ctx, "go", "go", []string{"version"})
	records := m.Records()
	if len(records) == 0 {
		t.Error("expected records")
	}
}

func TestWrapCommand(t *testing.T) {
	ctx := context.Background()
	cmd, err := WrapCommand(ctx, "go", "go", false)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == nil {
		t.Error("nil cmd")
	}
}
