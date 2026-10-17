package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/HP-jh/TarsEnterprise/pkg/company"
	"github.com/HP-jh/TarsEnterprise/pkg/config"
	"github.com/HP-jh/TarsEnterprise/pkg/nats"
	"github.com/HP-jh/TarsEnterprise/pkg/role"
	"github.com/HP-jh/TarsEnterprise/pkg/tsgclient"
)

// TSGInterop TSG 互通状态机
type TSGInterop struct {
	mu        sync.RWMutex
	commands  []TSGCommand      // TE → TSG 发过的指令
	inbox     []TSGCommand      // TSG → TE 收到的指令
	state     map[string]interface{}
	lastSync  time.Time
	tsClient  *tsgclient.Client
	transport *nats.Transport
}

// TSGCommand 跨网关指令
type TSGCommand struct {
	ID        string                 `json:"id"`
	Direction string                 `json:"direction"` // "te_to_tsg" | "tsg_to_te"
	ToolID    string                 `json:"tool_id,omitempty"`
	Params    map[string]interface{} `json:"params,omitempty"`
	State     map[string]interface{} `json:"state,omitempty"`
	RoleToken string                 `json:"role_token,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Status    string                 `json:"status"` // "pending" | "delivered" | "acked"
}

func newTSGInterop(cfg config.TSGBridgeConfig) *TSGInterop {
	ti := &TSGInterop{
		commands: make([]TSGCommand, 0),
		inbox:    make([]TSGCommand, 0),
		state:    make(map[string]interface{}),
	}

	// 初始化 NATS transport
	url := cfg.NATSURL
	if url == "" {
		url = "nats://localhost:4222"
	}
	ti.transport = nats.NewTransport(url)
	_ = ti.transport.Connect()

	// 初始化 TSG HTTP 客户端
	if cfg.Enabled && cfg.Address != "" {
		if cfg.MTLSCert != "" && cfg.MTLSKey != "" {
			fmt.Println("[TSG] mTLS configured but CA missing, using HTTP fallback")
		}
		ti.tsClient = tsgclient.NewInsecureClient(cfg.Address)
	}

	// 订阅 NATS 主题
	_ = ti.transport.Subscribe("te.commands.tsg", func(msg nats.Message) {
		if msg.From == "tsg" && msg.Type == "command" {
			cmd := TSGCommand{
				ID:        msg.ID,
				Direction: "tsg_to_te",
				RoleToken: msg.RoleToken,
				Timestamp: msg.Timestamp,
				Status:    "pending",
			}
			if toolID, ok := msg.Payload["tool_id"].(string); ok {
				cmd.ToolID = toolID
			}
			if params, ok := msg.Payload["params"].(map[string]interface{}); ok {
				cmd.Params = params
			}
			ti.mu.Lock()
			ti.inbox = append(ti.inbox, cmd)
			ti.mu.Unlock()
		}
	})

	_ = ti.transport.Subscribe("te.state.tsg", func(msg nats.Message) {
		if msg.From == "tsg" && msg.Type == "state_sync" {
			ti.mu.Lock()
			for k, v := range msg.Payload {
				ti.state[k] = v
			}
			ti.lastSync = time.Now()
			ti.mu.Unlock()
		}
	})

	return ti
}

// --- HTTP Handlers ---

func (g *Gateway) handleTSGCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ToolID    string                 `json:"tool_id"`
		Params    map[string]interface{} `json:"params"`
		RoleToken string                 `json:"role_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := TSGCommand{
		ID:        fmt.Sprintf("cmd-%d", time.Now().UnixNano()),
		Direction: "te_to_tsg",
		ToolID:    req.ToolID,
		Params:    req.Params,
		RoleToken: req.RoleToken,
		Timestamp: time.Now().Unix(),
		Status:    "pending",
	}

	g.tsgInterop.mu.Lock()
	g.tsgInterop.commands = append(g.tsgInterop.commands, cmd)
	g.tsgInterop.mu.Unlock()

	// NATS 投递
	_ = g.tsgInterop.transport.Publish("te.commands.tsg", nats.Message{
		From:      "te",
		To:        "tsg",
		Type:      "command",
		RoleToken: req.RoleToken,
		Payload: map[string]interface{}{
			"tool_id": req.ToolID,
			"params":  req.Params,
			"cmd_id":  cmd.ID,
		},
	})

	// HTTP 投递到 TSG
	if g.tsgInterop.tsClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		resp, err := g.tsgInterop.tsClient.SendCommand(ctx, req.ToolID, req.Params)
		cancel()
		if err == nil && resp.StatusCode == http.StatusOK {
			cmd.Status = "delivered"
			g.tsgInterop.mu.Lock()
			for i := range g.tsgInterop.commands {
				if g.tsgInterop.commands[i].ID == cmd.ID {
					g.tsgInterop.commands[i].Status = "delivered"
					break
				}
			}
			g.tsgInterop.mu.Unlock()
		}
	}

	writeJSON(w, map[string]interface{}{
		"cmd_id": cmd.ID,
		"status": cmd.Status,
	})
}

func (g *Gateway) handleTSGCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	g.tsgInterop.mu.RLock()
	cmds := make([]TSGCommand, len(g.tsgInterop.inbox))
	copy(cmds, g.tsgInterop.inbox)
	g.tsgInterop.mu.RUnlock()
	writeJSON(w, map[string]interface{}{
		"commands": cmds,
		"count":    len(cmds),
	})
}

func (g *Gateway) handleTSGEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"event": "connected", "version": "v0.3.0"})
	flusher.Flush()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	lastCount := 0
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			g.tsgInterop.mu.RLock()
			count := len(g.tsgInterop.inbox)
			g.tsgInterop.mu.RUnlock()
			if count != lastCount {
				lastCount = count
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"event": "commands_updated",
					"count": count,
				})
				flusher.Flush()
			}
		}
	}
}

func (g *Gateway) handleTSGStateSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		State     map[string]interface{} `json:"state"`
		RoleToken string                 `json:"role_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	g.tsgInterop.mu.Lock()
	for k, v := range req.State {
		g.tsgInterop.state[k] = v
	}
	g.tsgInterop.lastSync = time.Now()
	g.tsgInterop.mu.Unlock()

	_ = g.tsgInterop.transport.StateSync("push", req.State)

	if g.tsgInterop.tsClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = g.tsgInterop.tsClient.StateSync(ctx, req.State)
		cancel()
	}

	writeJSON(w, map[string]interface{}{"status": "synced", "timestamp": time.Now().Unix()})
}

func (g *Gateway) handleTSGState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	g.tsgInterop.mu.RLock()
	state := make(map[string]interface{})
	for k, v := range g.tsgInterop.state {
		state[k] = v
	}
	lastSync := g.tsgInterop.lastSync
	g.tsgInterop.mu.RUnlock()
	writeJSON(w, map[string]interface{}{
		"state":     state,
		"last_sync": lastSync.Unix(),
		"sync_age":  time.Since(lastSync).Seconds(),
	})
}

func (g *Gateway) handleTSGRoleBind(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		UserID    string `json:"user_id"`
		RoleID    string `json:"role_id"`
		TSGToken  string `json:"tsg_token"`
		CompanyID string `json:"company_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.UserID == "" || req.RoleID == "" {
		http.Error(w, "user_id and role_id required", http.StatusBadRequest)
		return
	}

	id := &role.Identity{
		UserID:      req.UserID,
		RoleID:      req.RoleID,
		CompanyType: company.CompanyType(g.cfg.CompanyType),
		Attributes: map[string]string{
			"tsg_token":  req.TSGToken,
			"company_id": req.CompanyID,
		},
	}
	_ = g.roleMgr.RegisterIdentity(id)

	if g.tsgInterop.tsClient != nil && req.TSGToken != "" {
		g.tsgInterop.tsClient.SetRoleToken(req.TSGToken)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = g.tsgInterop.tsClient.StateSync(ctx, map[string]interface{}{
			"event":      "role_bound",
			"user_id":    req.UserID,
			"role_id":    req.RoleID,
			"company_id": req.CompanyID,
		})
		cancel()
	}

	writeJSON(w, map[string]interface{}{
		"bound":   true,
		"user_id": req.UserID,
		"role_id": req.RoleID,
	})
}

// handleRoleCheckCross 扩展：支持跨网关模式（带 role_token 调用 TSG 验签）
func (g *Gateway) handleRoleCheckCross(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		UserID     string `json:"user_id"`
		ResourceID string `json:"resource_id"`
		Action     string `json:"action"`
		RoleToken  string `json:"role_token,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	localOK := g.roleMgr.Can(req.UserID, req.ResourceID, company.Action(req.Action))

	crossOK := false
	if req.RoleToken != "" && g.tsgInterop.tsClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		vr, err := g.tsgInterop.tsClient.ValidateRoleToken(ctx, req.RoleToken)
		cancel()
		if err == nil && vr.Valid {
			crossOK = true
		}
	}

	writeJSON(w, map[string]interface{}{
		"allowed":    localOK || crossOK,
		"local":      localOK,
		"cross":      crossOK,
		"role_token": req.RoleToken != "",
	})
}
