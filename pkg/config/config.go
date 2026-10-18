package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

var Version = "v4.34.0"

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

type MTLSConfig struct {
	Enabled      bool   `json:"enabled"`
	Mode         string `json:"mode"`          // "self_signed" | "custom_ca"
	CertFile     string `json:"cert_file"`     // 服务端证书路径
	KeyFile      string `json:"key_file"`      // 服务端密钥路径
	CAFile       string `json:"ca_file"`       // CA 证书路径（自定义 CA 模式）
	ClientAuth   string `json:"client_auth"`   // "require" | "verify" | "optional"
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
		MTLS:        MTLSConfig{Enabled: false, Mode: "self_signed", ClientAuth: "require"},
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

// ValidationError holds a config validation issue with a user-friendly message.
type ValidationError struct {
	Field       string `json:"field"`
	Message     string `json:"message"`
	FriendlyMsg string `json:"friendly_msg"`
	Suggestion  string `json:"suggestion"`
}

// Validate checks the configuration for common problems and returns
// user-friendly errors (小白化提示).
func (c *Config) Validate() []ValidationError {
	var errs []ValidationError

	// Admin token should be set for non-first-run
	if !c.FirstRun && c.AdminToken == "" {
		errs = append(errs, ValidationError{
			Field:       "admin_token",
			Message:     "admin token is empty",
			FriendlyMsg: "管理员令牌还没设置呢，这会让自动更新等管理功能无法使用",
			Suggestion:  "在配置文件中添加 admin_token 字段，或者运行安装向导重新生成",
		})
	}

	// mTLS consistency
	if c.MTLS.Enabled {
		if c.MTLS.Mode != "self_signed" && c.MTLS.Mode != "custom_ca" {
			errs = append(errs, ValidationError{
				Field:       "mtls.mode",
				Message:     "invalid mTLS mode: " + c.MTLS.Mode,
				FriendlyMsg: "mTLS 模式设置不对，当前值不认识",
				Suggestion:  "请使用 self_signed（自动生成证书）或 custom_ca（使用企业自有证书）",
			})
		}
		if c.MTLS.Mode == "custom_ca" {
			if c.MTLS.CertFile == "" || c.MTLS.KeyFile == "" || c.MTLS.CAFile == "" {
				errs = append(errs, ValidationError{
					Field:       "mtls.cert_file",
					Message:     "custom_ca mode requires cert_file, key_file, and ca_file",
					FriendlyMsg: "选择了自定义证书模式，但证书文件路径没有填全",
					Suggestion:  "请检查 cert_file、key_file、ca_file 三个路径是否都已正确配置",
				})
			}
		}
		if c.MTLS.ClientAuth == "" {
			errs = append(errs, ValidationError{
				Field:       "mtls.client_auth",
				Message:     "client_auth is empty",
				FriendlyMsg: "客户端证书验证方式没设置",
				Suggestion:  "建议设为 require（强制要求客户端证书）",
			})
		}
	}

	// TSG bridge address format
	if c.TSGBridge.Enabled && c.TSGBridge.Address == "" {
		errs = append(errs, ValidationError{
			Field:       "tsg_bridge.address",
			Message:     "TSG bridge enabled but address is empty",
			FriendlyMsg: "TSG 网关互通已开启，但目标地址还没填",
			Suggestion:  "在 tsg_bridge.address 中填写 TSG 网关的地址，例如 http://localhost:8080",
		})
	}

	// Company type validity
	if c.CompanyType == "" {
		errs = append(errs, ValidationError{
			Field:       "company_type",
			Message:     "company_type is empty",
			FriendlyMsg: "公司类型还没选，这会影响角色矩阵和知识库的默认配置",
			Suggestion:  "可选值：tech（科技）、education（教育）、finance（金融）、healthcare（医疗）",
		})
	}

	// Metrics port sanity
	if c.Metrics.Enabled && c.Metrics.Port <= 0 {
		errs = append(errs, ValidationError{
			Field:       "metrics.port",
			Message:     "metrics port must be positive",
			FriendlyMsg: "监控指标端口设置不对，端口号必须是正数",
			Suggestion:  "建议使用 8888 或 9090 等常用端口",
		})
	}

	return errs
}
