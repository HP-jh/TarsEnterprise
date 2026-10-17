package tsgclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/HP-jh/TarsEnterprise/pkg/mtls"
)

// Client TSG 个人网关 HTTP/mTLS 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	roleToken  string
}

// NewClient 创建 TSG 客户端
func NewClient(tsgAddr string, tlsCfg *tls.Config) *Client {
	transport := &http.Transport{
		TLSClientConfig: tlsCfg,
	}
	return &Client{
		baseURL:    "https://" + tsgAddr,
		httpClient: &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}
}

// NewInsecureClient 创建无 TLS 的 TSG 客户端（开发/测试用）
func NewInsecureClient(tsgAddr string) *Client {
	return &Client{
		baseURL:    "http://" + tsgAddr,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetRoleToken 设置角色令牌
func (c *Client) SetRoleToken(token string) {
	c.roleToken = token
}

// SendCommand 向 TSG 发送指令
func (c *Client) SendCommand(ctx context.Context, toolID string, params map[string]interface{}) (*CommandResponse, error) {
	reqBody := map[string]interface{}{
		"tool_id": toolID,
		"params":  params,
	}
	if c.roleToken != "" {
		reqBody["role_token"] = c.roleToken
	}
	return c.post(ctx, "/api/v1/interop/receive-command", reqBody)
}

// StateSync 向 TSG 同步状态
func (c *Client) StateSync(ctx context.Context, state map[string]interface{}) (*CommandResponse, error) {
	reqBody := map[string]interface{}{"state": state}
	if c.roleToken != "" {
		reqBody["role_token"] = c.roleToken
	}
	return c.post(ctx, "/api/v1/interop/receive-state", reqBody)
}

// HealthCheck 检查 TSG 健康状态
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	var hr HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		return nil, err
	}
	hr.StatusCode = resp.StatusCode
	return &hr, nil
}

// SubscribeCommand 订阅 TSG 下发的指令（WebSocket 长连接）
func (c *Client) SubscribeCommand(handler func(CommandMessage)) error {
	// v0.3.0: 使用 HTTP 轮询作为 WebSocket fallback
	// 实际 WebSocket 在 gateway 端通过 /api/v1/tsg/events 实现
	go c.pollLoop(handler)
	return nil
}

// pollLoop 轮询拉取 TSG 下发的指令
func (c *Client) pollLoop(handler func(CommandMessage)) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		cmds, err := c.FetchCommands(context.Background())
		if err != nil {
			continue
		}
		for _, cmd := range cmds {
			handler(cmd)
		}
	}
}

// FetchCommands 拉取 TSG 下发的待处理指令列表
func (c *Client) FetchCommands(ctx context.Context) ([]CommandMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/interop/commands", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch commands %d: %s", resp.StatusCode, string(body))
	}
	var result struct {
		Commands []CommandMessage `json:"commands"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Commands, nil
}

// ValidateRoleToken 向 TSG 验证角色令牌
func (c *Client) ValidateRoleToken(ctx context.Context, token string) (*RoleValidation, error) {
	return c.postRoleCheck(ctx, token)
}

// --- 内部 HTTP 辅助 ---

func (c *Client) post(ctx context.Context, path string, body interface{}) (*CommandResponse, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.roleToken != "" {
		req.Header.Set("X-Role-Token", c.roleToken)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tsg request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyData, _ := io.ReadAll(resp.Body)
	cr := &CommandResponse{
		StatusCode: resp.StatusCode,
		RawBody:    bodyData,
	}
	_ = json.Unmarshal(bodyData, &cr.Data)
	return cr, nil
}

func (c *Client) postRoleCheck(ctx context.Context, token string) (*RoleValidation, error) {
	reqBody := map[string]string{"role_token": token}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/role/validate", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var rv RoleValidation
	if err := json.NewDecoder(resp.Body).Decode(&rv); err != nil {
		return nil, err
	}
	return &rv, nil
}

// --- 数据类型 ---

// CommandResponse TSG 响应
type CommandResponse struct {
	StatusCode int
	RawBody    []byte
	Data       map[string]interface{}
}

// HealthResponse TSG 健康响应
type HealthResponse struct {
	Status     string `json:"status"`
	Version    string `json:"version"`
	StatusCode int    `json:"-"`
}

// CommandMessage TSG 下发指令
type CommandMessage struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"`
	ToolID    string                 `json:"tool_id"`
	Params    map[string]interface{} `json:"params"`
	Timestamp int64                  `json:"timestamp"`
}

// RoleValidation 角色验证结果
type RoleValidation struct {
	Valid     bool   `json:"valid"`
	RoleID    string `json:"role_id,omitempty"`
	CompanyID string `json:"company_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

// BuildTLSConfig 从 PEM 构建 mTLS 配置
func BuildTLSConfig(certPEM, keyPEM, caPEM []byte) (*tls.Config, error) {
	return mtls.LoadTLSConfig(certPEM, keyPEM, caPEM)
}

// GenerateSelfSigned 生成自签名证书
func GenerateSelfSigned(commonName string) (*mtls.Pair, error) {
	return mtls.GenerateSelfSigned(commonName)
}
