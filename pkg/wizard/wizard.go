package wizard

import (
	"fmt"

	"github.com/HP-jh/TarsEnterprise/pkg/company"
	"github.com/HP-jh/TarsEnterprise/pkg/config"
)

// Step 向导步骤
type Step struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Desc    string `json:"desc"`
	Fields  []Field `json:"fields"`
}

type Field struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // "select", "text", "multi"
	Options  []string `json:"options,omitempty"`
	Required bool     `json:"required"`
}

// Steps 返回 6 步首次配置向导定义
func Steps() []Step {
	companyOpts := make([]string, 0, len(company.AllTypes))
	for _, t := range company.AllTypes {
		if s, err := company.GetSchema(t); err == nil {
			companyOpts = append(companyOpts, fmt.Sprintf("%s|%s", t, s.Name))
		}
	}
	return []Step{
		{
			ID: "step1", Title: "第 1 步：选择公司类型", Desc: "告诉我们你的组织类型，我们会自动配置合适的角色和权限",
			Fields: []Field{
				{ID: "company_type", Label: "公司类型", Type: "select", Options: companyOpts, Required: true},
			},
		},
		{
			ID: "step2", Title: "第 2 步：配置角色", Desc: "选择组织中需要使用的角色",
			Fields: []Field{
				{ID: "roles", Label: "角色", Type: "multi", Options: []string{"admin", "manager", "employee", "guest"}, Required: true},
			},
		},
		{
			ID: "step3", Title: "第 3 步：接入客户端", Desc: "配置需要接入的客户端（个人版 TSG、第三方网关等）",
			Fields: []Field{
				{ID: "clients", Label: "客户端类型", Type: "multi", Options: []string{"tsg_personal", "discord", "telegram", "feishu", "custom"}, Required: false},
			},
		},
		{
			ID: "step4", Title: "第 4 步：挂载工具", Desc: "选择需要挂载的 AI 工具 / MCP Server",
			Fields: []Field{
				{ID: "tools", Label: "工具", Type: "multi", Options: []string{"openclaw", "openai_fc", "custom_python", "custom_nodejs"}, Required: false},
			},
		},
		{
			ID: "step5", Title: "第 5 步：安全与沙箱", Desc: "配置沙箱和互通安全选项",
			Fields: []Field{
				{ID: "sandbox", Label: "启用沙箱", Type: "select", Options: []string{"gvisor|gVisor（推荐）", "seccomp|Seccomp", "none|不启用"}, Required: true},
				{ID: "tsg_bridge", Label: "启用 TSG 互通", Type: "select", Options: []string{"true|启用", "false|暂不启用"}, Required: true},
			},
		},
		{
			ID: "step6", Title: "第 6 步：完成", Desc: "配置完成！点击完成启动网关",
			Fields: []Field{
				{ID: "confirm", Label: "确认启动", Type: "select", Options: []string{"true|立即启动"}, Required: true},
			},
		},
	}
}

// Run 运行向导并返回配置（v0.1.0 返回默认配置 + 模拟向导选择）
func Run() *config.Config {
	// 在实际 UI 中，前端会逐步收集用户选择并 POST 到 /api/v1/wizard/submit
	// 这里返回一个预填充的教育类型配置作为演示
	cfg := &config.Config{
		FirstRun:    false,
		CompanyType: string(company.Education),
		Roles:       []string{"principal", "teacher", "student"},
		Clients: []config.ClientConfig{
			{ID: "tsg-001", Type: "tsg_personal", Endpoint: "localhost:8443", Token: ""},
		},
		Tools: []config.ToolConfig{
			{ID: "openclaw-1", Name: "OpenClaw Gateway", Lang: "go", Entry: "openclaw", Clients: []string{"tsg-001"}, Sandboxed: true},
		},
		MCP:       config.MCPConfig{Enabled: true, Servers: []string{"openclaw"}, OpenClaw: true},
		TSGBridge: config.TSGBridgeConfig{Enabled: true, Address: "localhost:8443"},
		Sandbox:   config.SandboxConfig{Enabled: true, Runtime: "gvisor", Seccomp: true},
		Metrics:   config.MetricsConfig{Enabled: true, Port: 8888},
	}
	return cfg
}

// ApplyAnswers 将向导答案应用到配置
func ApplyAnswers(cfg *config.Config, answers map[string]string) error {
	if ct, ok := answers["company_type"]; ok {
		cfg.CompanyType = ct
	}
	if sandbox, ok := answers["sandbox"]; ok {
		cfg.Sandbox.Enabled = sandbox != "none"
		cfg.Sandbox.Runtime = sandbox
	}
	if bridge, ok := answers["tsg_bridge"]; ok {
		cfg.TSGBridge.Enabled = bridge == "true"
	}
	cfg.FirstRun = false
	return nil
}
