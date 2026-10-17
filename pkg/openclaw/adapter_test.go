package openclaw

import (
	"testing"

	"github.com/HP-jh/TarsEnterprise/pkg/mcp"
)

func TestAdapter_RegisterMCP(t *testing.T) {
	hub := mcp.NewHub()
	a := NewAdapter(hub)
	a.RegisterMCP("oc1", "http://localhost:8080")

	list := hub.ListServers()
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
}

func TestAdapter_CheckVuln(t *testing.T) {
	a := NewAdapter(mcp.NewHub())
	v, ok := a.CheckVuln("CVE-2025-OC001")
	if !ok {
		t.Error("expected covered")
	}
	if !v.Covered {
		t.Error("covered flag")
	}
	_, ok = a.CheckVuln("CVE-9999-UNKNOWN")
	if ok {
		t.Error("unknown should not be found")
	}
}

func TestAdapter_ListCoveredVulns(t *testing.T) {
	a := NewAdapter(mcp.NewHub())
	list := a.ListCoveredVulns()
	if len(list) == 0 {
		t.Error("expected vulns")
	}
}

func TestValidatePlatform(t *testing.T) {
	if err := ValidatePlatform(PlatformConfig{Platform: Feishu, WebhookURL: "http://x"}); err != nil {
		t.Error("valid config")
	}
	if err := ValidatePlatform(PlatformConfig{Platform: "xxx"}); err == nil {
		t.Error("expected error for bad platform")
	}
	if err := ValidatePlatform(PlatformConfig{Platform: Discord}); err == nil {
		t.Error("expected error for missing credentials")
	}
}
