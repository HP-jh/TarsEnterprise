package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var Version = "v0.5.0"

type Config struct {
	FirstRun    bool              `json:"first_run"`
	CompanyType string            `json:"company_type"`
	Roles       []string          `json:"roles"`
	Clients     []ClientConfig    `json:"clients"`
	Tools       []ToolConfig      `json:"tools"`
	MCP         MCPConfig         `json:"mcp"`
	TSGBridge   TSGBridgeConfig   `json:"tsg_bridge"`
	Sandbox     SandboxConfig     `json:"sandbox"`
	Metrics     MetricsConfig     `json:"metrics"`
	AdminToken  string            `json:"admin_token"`
	MTLS        MTLSConfig        `json:"mtls"`
}

type MTLSConfig struct {
	Enabled        bool     `json:"enabled"`
	CertPath       string   `json:"cert_path"`
	KeyPath        string   `json:"key_path"`
	CAPath         string   `json:"ca_path"`
	AutoRotate     bool     `json:"auto_rotate"`
	RotateInterval string   `json:"rotate_interval"` // 解析为 duration，如 "720h"
	RotateWindow   string   `json:"rotate_window"`   // 证书过期前多久开始轮换，如 "168h"
	AllowRoles     []string `json:"allow_roles"`     // 零信任：允许访问的角色列表（证书 CN/OU 匹配）
	Enforce        bool     `json:"enforce"`         // 零信任：是否强制拒绝无有效客户端证书的连接
}

type ClientConfig struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Endpoint string `json:"endpoint"`
	Token    string `json:"token"`
}

type ToolConfig struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Lang        string   `json:"lang"`
	Entry       string   `json:"entry"`
	Clients     []string `json:"clients"`
	Sandboxed   bool     `json:"sandboxed"`
}

type MCPConfig struct {
	Enabled    bool     `json:"enabled"`
	Servers    []string `json:"servers"`
	OpenClaw   bool     `json:"openclaw"`
}

type TSGBridgeConfig struct {
	Enabled   bool   `json:"enabled"`
	Address   string `json:"address"`
	MTLSCert  string `json:"mtls_cert"`
	MTLSKey   string `json:"mtls_key"`
	NATSURL   string `json:"nats_url"`
}

type SandboxConfig struct {
	Enabled   bool   `json:"enabled"`
	Runtime   string `json:"runtime"`
	Seccomp   bool   `json:"seccomp"`
}

type MetricsConfig struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

func defaultConfig() *Config {
	return &Config{
		FirstRun:    true,
		CompanyType: "tech",
		Roles:       []string{"admin"},
		Clients:     []ClientConfig{},
		Tools:       []ToolConfig{},
		MCP:         MCPConfig{Enabled: true},
		TSGBridge:   TSGBridgeConfig{Enabled: false},
		Sandbox:     SandboxConfig{Enabled: true, Runtime: "gvisor", Seccomp: true},
		Metrics:     MetricsConfig{Enabled: true, Port: 8888},
		MTLS:        MTLSConfig{Enabled: false, AutoRotate: true, RotateInterval: "720h", RotateWindow: "168h", Enforce: true},
	}
}

func configPath(path string) string {
	if path != "" {
		return path
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tarsenterprise", "config.json")
}

func LoadOrInit(path string) (*Config, error) {
	p := configPath(path)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	p := configPath("")
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}

func (c *Config) SaveTo(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
