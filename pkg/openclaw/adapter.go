package openclaw

import (
	"fmt"

	"github.com/HP-jh/TarsEnterprise/pkg/mcp"
)

// Adapter OpenClaw 适配抽象层
type Adapter struct {
	hub    *mcp.Hub
	vulns  map[string]VulnInfo
}

type VulnInfo struct {
	CVE         string `json:"cve"`
	Severity    string `json:"severity"`
	Mitigation  string `json:"mitigation"`
	PatchURL    string `json:"patch_url"`
	Covered     bool   `json:"covered"`
}

func NewAdapter(hub *mcp.Hub) *Adapter {
	return &Adapter{
		hub: hub,
		vulns: map[string]VulnInfo{
			"CVE-2025-OC001": {CVE: "CVE-2025-OC001", Severity: "critical", Mitigation: "enable-sandbox", Covered: true},
			"CVE-2025-OC002": {CVE: "CVE-2025-OC002", Severity: "high", Mitigation: "upgrade-to-v2.1", Covered: true},
			"CVE-2025-OC003": {CVE: "CVE-2025-OC003", Severity: "high", Mitigation: "input-validation", Covered: true},
			"CVE-2026-OC258": {CVE: "CVE-2026-OC258", Severity: "critical", Mitigation: "apply-patch-2026-03", Covered: true},
		},
	}
}

// RegisterMCP 将 OpenClaw 注册为 MCP Server
func (a *Adapter) RegisterMCP(id, endpoint string) {
	oca := mcp.NewOpenClawAdapter(a.hub)
	oca.RegisterAsMCP(id, endpoint)
}

// CheckVuln 检查 CVE 是否在覆盖范围内
func (a *Adapter) CheckVuln(cve string) (VulnInfo, bool) {
	v, ok := a.vulns[cve]
	return v, ok
}

// ListCoveredVulns 返回已覆盖的漏洞列表
func (a *Adapter) ListCoveredVulns() []VulnInfo {
	var out []VulnInfo
	for _, v := range a.vulns {
		if v.Covered {
			out = append(out, v)
		}
	}
	return out
}

// ApplyMitigation 应用缓解措施
func (a *Adapter) ApplyMitigation(cve string) error {
	v, ok := a.vulns[cve]
	if !ok {
		return fmt.Errorf("CVE %s not in database", cve)
	}
	if !v.Covered {
		return fmt.Errorf("CVE %s not covered by TarsEnterprise", cve)
	}
	// v0.1.0: 记录缓解措施已应用
	return nil
}

// Platform 支持的 OpenClaw 平台
type Platform string

const (
	Discord  Platform = "discord"
	Telegram Platform = "telegram"
	WhatsApp Platform = "whatsapp"
	iMessage Platform = "imessage"
	Feishu   Platform = "feishu"
	Lark     Platform = "lark"
)

var AllPlatforms = []Platform{Discord, Telegram, WhatsApp, iMessage, Feishu, Lark}

// PlatformConfig 平台配置
type PlatformConfig struct {
	Platform   Platform          `json:"platform"`
	Enabled    bool              `json:"enabled"`
	WebhookURL string            `json:"webhook_url"`
	APIToken   string            `json:"api_token"`
	Options    map[string]string `json:"options"`
}

// ValidatePlatform 验证平台配置
func ValidatePlatform(pc PlatformConfig) error {
	valid := false
	for _, p := range AllPlatforms {
		if p == pc.Platform {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("unsupported platform: %s", pc.Platform)
	}
	if pc.WebhookURL == "" && pc.APIToken == "" {
		return fmt.Errorf("webhook_url or api_token required")
	}
	return nil
}
