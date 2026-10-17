package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Server 表示一个 MCP Server 端点
type Server struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Endpoint string            `json:"endpoint"`
	Tools    []Tool            `json:"tools"`
	Meta     map[string]string `json:"meta"`
}

// Tool MCP 工具定义
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Hub MCP Server 管理中心
type Hub struct {
	servers map[string]*Server
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{servers: make(map[string]*Server)}
}

func (h *Hub) Register(s *Server) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.servers[s.ID] = s
}

func (h *Hub) ListServers() []*Server {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var out []*Server
	for _, s := range h.servers {
		out = append(out, s)
	}
	return out
}

func (h *Hub) GetServer(id string) (*Server, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s, ok := h.servers[id]
	return s, ok
}

func (h *Hub) Call(ctx context.Context, serverID, toolName string, params map[string]interface{}) (map[string]interface{}, error) {
	s, ok := h.GetServer(serverID)
	if !ok {
		return nil, fmt.Errorf("mcp server %s not found", serverID)
	}
	var found bool
	for _, t := range s.Tools {
		if t.Name == toolName {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("tool %s not found in server %s", toolName, serverID)
	}
	// In v0.1.0, we return a simulated result
	return map[string]interface{}{
		"server":  serverID,
		"tool":    toolName,
		"params":  params,
		"status":  "simulated",
		"version": "v0.1.0",
	}, nil
}

// OpenClawAdapter OpenClaw MCP 化适配器
type OpenClawAdapter struct {
	hub      *Hub
	vulnDB   map[string]string // CVE -> mitigation
}

func NewOpenClawAdapter(hub *Hub) *OpenClawAdapter {
	return &OpenClawAdapter{
		hub: hub,
		vulnDB: map[string]string{
			"CVE-2025-XXXX": "upgrade-to-v2.1",
			"CVE-2025-YYYY": "enable-sandbox",
			"CVE-2026-258":  "apply-patch-2026-03",
		},
	}
}

func (oca *OpenClawAdapter) RegisterAsMCP(id, endpoint string) {
	oca.hub.Register(&Server{
		ID:       id,
		Name:     "OpenClaw Gateway " + id,
		Endpoint: endpoint,
		Tools: []Tool{
			{Name: "send_message", Description: "Send message to platform", InputSchema: map[string]interface{}{"type": "object"}},
			{Name: "list_channels", Description: "List available channels", InputSchema: map[string]interface{}{"type": "object"}},
			{Name: "check_vuln", Description: "Check OpenClaw vulnerability", InputSchema: map[string]interface{}{"type": "object"}},
		},
		Meta: map[string]string{"provider": "openclaw", "mcp_version": "1.0"},
	})
}

func (oca *OpenClawAdapter) Mitigate(cve string) (string, bool) {
	m, ok := oca.vulnDB[cve]
	return m, ok
}

// ParseOpenAITools 将 OpenAI Function Calling 格式转换为 MCP Tool 格式
func ParseOpenAITools(openAITools []json.RawMessage) ([]Tool, error) {
	var tools []Tool
	for _, raw := range openAITools {
		var t Tool
		if err := json.Unmarshal(raw, &t); err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}
	return tools, nil
}
