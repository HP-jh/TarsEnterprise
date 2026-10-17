package tsgclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/HP-jh/TarsEnterprise/pkg/mtls"
)

// Client TSG 个人网关客户端 SDK
type Client struct {
	baseURL    string
	httpClient *http.Client
	roleToken  string
	mu         sync.RWMutex
}

// New 创建 TSG 客户端（HTTP 模式）
func New(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// NewWithMTLS 创建带 mTLS 双向认证的 TSG 客户端
func NewWithMTLS(baseURL string, certPEM, keyPEM, caPEM []byte) (*Client, error) {
	tlsCfg, err := mtls.LoadTLSConfig(certPEM, keyPEM, caPEM)
	if err != nil {
		return nil, fmt.Errorf("load tls: %w", err)
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsCfg,
			},
		},
	}, nil
}

// SetRoleToken 设置角色令牌（后续请求自动携带）
func (c *Client) SetRoleToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.roleToken = token
}

func (c *Client) roleHeader() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.roleToken
}

// SendCommand 向 TSG 发送指令（TE → TSG）
func (c *Client) SendCommand(ctx context.Context, toolID string, params map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"tool_id": toolID,
		"params":  params,
	}
	return c.post(ctx, "/api/v1/command", body)
}

// StateSync 推送状态到 TSG（TE → TSG）
func (c *Client) StateSync(ctx context.Context, state map[string]interface{}) (map[string]interface{}, error) {
	return c.post(ctx, "/api/v1/state/sync", state)
}

// HealthCheck 检查 TSG 健康状态
func (c *Client) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	return c.get(ctx, "/health")
}

// PullCommands 拉取 TSG → TE 的待处理指令列表
func (c *Client) PullCommands(ctx context.Context) ([]map[string]interface{}, error) {
	resp, err := c.get(ctx, "/api/v1/commands/outbound")
	if err != nil {
		return nil, err
	}
	items, ok := resp["commands"].([]interface{})
	if !ok {
		return nil, nil
	}
	var out []map[string]interface{}
	for _, it := range items {
		if m, ok := it.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out, nil
}

// PullState 拉取 TSG 的当前状态
func (c *Client) PullState(ctx context.Context) (map[string]interface{}, error) {
	return c.get(ctx, "/api/v1/state")
}

// VerifyRole 跨网关角色验证（带 role_token 调用 TSG 验签）
func (c *Client) VerifyRole(ctx context.Context, userID, resourceID, action string) (bool, error) {
	body := map[string]interface{}{
		"user_id":     userID,
		"resource_id": resourceID,
		"action":      action,
		"role_token":  c.roleHeader(),
	}
	resp, err := c.post(ctx, "/api/v1/role/verify", body)
	if err != nil {
		return false, err
	}
	allowed, _ := resp["allowed"].(bool)
	return allowed, nil
}

// BindRole 将企业版角色绑定到 TSG 客户端
func (c *Client) BindRole(ctx context.Context, roleID, profession string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"role_id":    roleID,
		"profession": profession,
		"role_token": c.roleHeader(),
	}
	return c.post(ctx, "/api/v1/role/bind", body)
}

func (c *Client) post(ctx context.Context, path string, body interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if rt := c.roleHeader(); rt != "" {
		req.Header.Set("X-Role-Token", rt)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return c.decode(resp)
}

func (c *Client) get(ctx context.Context, path string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if rt := c.roleHeader(); rt != "" {
		req.Header.Set("X-Role-Token", rt)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return c.decode(resp)
}

func (c *Client) decode(resp *http.Response) (map[string]interface{}, error) {
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
