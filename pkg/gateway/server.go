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
	"github.com/HP-jh/TarsEnterprise/pkg/knowledge"
	"github.com/HP-jh/TarsEnterprise/pkg/mcp"
	"github.com/HP-jh/TarsEnterprise/pkg/metrics"
	"github.com/HP-jh/TarsEnterprise/pkg/nats"
	"github.com/HP-jh/TarsEnterprise/pkg/obsidian"
	"github.com/HP-jh/TarsEnterprise/pkg/role"
	"github.com/HP-jh/TarsEnterprise/pkg/sandbox"
	"github.com/HP-jh/TarsEnterprise/pkg/tsgclient"
	"github.com/HP-jh/TarsEnterprise/pkg/voicecorrection"
)

type Gateway struct {
	cfg       *config.Config
	roleMgr   *role.Manager
	mcpHub    *mcp.Hub
	sandbox   *sandbox.Manager
	kbMgr     *knowledge.Manager
	obsCfg    *obsidian.ConnectorConfig
	voiceCorr *voicecorrection.Corrector
	// v0.3.0 TSG bridge
	tsgClient *tsgclient.Client
	natsTrans *nats.Transport
	stateCache map[string]interface{}
	stateMu    sync.RWMutex
	cmdHistory []map[string]interface{}
	cmdMu      sync.Mutex
}

func New(cfg *config.Config) *Gateway {
	gw := &Gateway{
		cfg:        cfg,
		roleMgr:    role.NewManager(),
		mcpHub:     mcp.NewHub(),
		sandbox:    sandbox.NewManager(),
		kbMgr:      knowledge.NewManager(),
		obsCfg:     obsidian.DefaultConnectorConfig(),
		voiceCorr:  voicecorrection.NewCorrector(""),
		stateCache: make(map[string]interface{}),
		cmdHistory: make([]map[string]interface{}, 0),
	}
	if cfg.CompanyType != "" {
		_ = gw.roleMgr.BuildMatrix(company.CompanyType(cfg.CompanyType))
		gw.kbMgr.InitCompanyBase(cfg.CompanyType)
	}
	// TSG bridge init
	if cfg.TSGBridge.Enabled && cfg.TSGBridge.Address != "" {
		gw.tsgClient = tsgclient.New(cfg.TSGBridge.Address)
	}
	if cfg.TSGBridge.NATSURL != "" || cfg.TSGBridge.Enabled {
		url := cfg.TSGBridge.NATSURL
		if url == "" {
			url = "nats://localhost:4222"
		}
		gw.natsTrans = nats.NewTransport(url)
		_ = gw.natsTrans.Connect() // graceful: memory-backed if real NATS unavailable
	}
	return gw
}

func (g *Gateway) RegisterRoutes(mux *http.ServeMux) {
	// Health
	mux.HandleFunc("/health", g.handleHealth)
	// Metrics (Prometheus)
	mux.HandleFunc("/metrics", metrics.HTTPHandler())
	// API v1
	mux.HandleFunc("/api/v1/company/schema", g.handleCompanySchema)
	mux.HandleFunc("/api/v1/role/check", g.handleRoleCheck)
	mux.HandleFunc("/api/v1/tools", g.handleTools)
	mux.HandleFunc("/api/v1/tools/call", g.handleToolCall)
	mux.HandleFunc("/api/v1/mcp/servers", g.handleMCPServers)
	mux.HandleFunc("/api/v1/mcp/call", g.handleMCPCall)
	mux.HandleFunc("/api/v1/sandbox/run", g.handleSandboxRun)
	mux.HandleFunc("/api/v1/config/merge", g.handleConfigMerge)
	mux.HandleFunc("/api/v1/lang/detect", g.handleLangDetect)
	mux.HandleFunc("/api/v1/tsg/bridge", g.handleTSGBridge)
	// Knowledge Base
	mux.HandleFunc("/api/v1/kb/search", g.handleKBSearch)
	mux.HandleFunc("/api/v1/kb/docs", g.handleKBDocs)
	mux.HandleFunc("/api/v1/kb/import", g.handleKBImport)
	// Obsidian
	mux.HandleFunc("/api/v1/obsidian/sync", g.handleObsidianSync)
	mux.HandleFunc("/api/v1/obsidian/writeback", g.handleObsidianWriteBack)
	// Voice Correction
	mux.HandleFunc("/api/v1/voice/correct", g.handleVoiceCorrect)
	// v0.3.0 TSG cross-gateway routes
	mux.HandleFunc("/api/v1/tsg/command", g.handleTSGCommand)
	mux.HandleFunc("/api/v1/tsg/commands", g.handleTSGCommands)
	mux.HandleFunc("/api/v1/tsg/events", g.handleTSGEvents)
	mux.HandleFunc("/api/v1/tsg/state/sync", g.handleTSGStateSync)
	mux.HandleFunc("/api/v1/tsg/state", g.handleTSGState)
	mux.HandleFunc("/api/v1/tsg/role/bind", g.handleTSGRoleBind)
	mux.HandleFunc("/api/v1/tsg/role/verify", g.handleTSGRoleVerify)
	// Wizard static files
	mux.Handle("/wizard/", http.StripPrefix("/wizard/", http.FileServer(http.Dir("./web"))))
	// Root redirect to wizard
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/wizard/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
}

func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"status":  "ok",
		"version": config.Version,
	})
}

func (g *Gateway) handleCompanySchema(w http.ResponseWriter, r *http.Request) {
	ctype := r.URL.Query().Get("type")
	if ctype == "" {
		ctype = g.cfg.CompanyType
	}
	schema, err := company.GetSchema(company.CompanyType(ctype))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, schema)
}

func (g *Gateway) handleRoleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		UserID     string `json:"user_id"`
		ResourceID string `json:"resource_id"`
		Action     string `json:"action"`
		RoleToken  string `json:"role_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ok := g.roleMgr.Can(req.UserID, req.ResourceID, company.Action(req.Action))
	// Cross-gateway mode: if role_token present, also verify via TSG
	if ok && req.RoleToken != "" && g.tsgClient != nil {
		g.tsgClient.SetRoleToken(req.RoleToken)
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		tsgoK, err := g.tsgClient.VerifyRole(ctx, req.UserID, req.ResourceID, req.Action)
		if err == nil {
			ok = ok && tsgoK
		}
	}
	writeJSON(w, map[string]bool{"allowed": ok})
}

func (g *Gateway) handleTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, g.cfg.Tools)
}

func (g *Gateway) handleToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ToolID   string                 `json:"tool_id"`
		Params   map[string]interface{} `json:"params"`
		ClientID string                 `json:"client_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var tool *config.ToolConfig
	for i := range g.cfg.Tools {
		if g.cfg.Tools[i].ID == req.ToolID {
			tool = &g.cfg.Tools[i]
			break
		}
	}
	if tool == nil {
		http.Error(w, "tool not found", http.StatusNotFound)
		return
	}
	metrics.RecordAgentCall(req.ToolID, req.ClientID)
	result, err := g.executeTool(tool, req.Params)
	if err != nil {
		metrics.RecordAgentError(req.ToolID)
		http.Error(w, fmt.Sprintf("tool execution failed: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{
		"tool_id": req.ToolID,
		"result":  result,
	})
}

func (g *Gateway) handleMCPServers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, g.mcpHub.ListServers())
}

func (g *Gateway) handleMCPCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Server string                 `json:"server"`
		Tool   string                 `json:"tool"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := g.mcpHub.Call(r.Context(), req.Server, req.Tool, req.Params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

func (g *Gateway) handleSandboxRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Lang    string   `json:"lang"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := g.sandbox.Run(r.Context(), req.Lang, req.Command, req.Args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

func (g *Gateway) handleConfigMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Modules []map[string]interface{} `json:"modules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	merged, collisions, err := config.MergeModules(req.Modules)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]interface{}{
		"merged":     merged,
		"collisions": collisions,
		"ok":         len(collisions) == 0,
	})
}

func (g *Gateway) handleLangDetect(w http.ResponseWriter, r *http.Request) {
	env := langDetect()
	writeJSON(w, env)
}

func (g *Gateway) handleTSGBridge(w http.ResponseWriter, r *http.Request) {
	status := "disconnected"
	if g.natsTrans != nil && g.natsTrans.Connected() {
		status = "connected"
	} else if g.tsgClient != nil {
		status = "http_ready"
	}
	writeJSON(w, map[string]interface{}{
		"enabled":    g.cfg.TSGBridge.Enabled,
		"address":    g.cfg.TSGBridge.Address,
		"nats_url":   g.cfg.TSGBridge.NATSURL,
		"status":     status,
		"version":    config.Version,
	})
}

// v0.3.0 TSG cross-gateway handlers

func (g *Gateway) handleTSGCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if g.tsgClient == nil {
		http.Error(w, "TSG bridge not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		ToolID string                 `json:"tool_id"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	resp, err := g.tsgClient.SendCommand(ctx, req.ToolID, req.Params)
	if err != nil {
		http.Error(w, fmt.Sprintf("tsg command failed: %v", err), http.StatusBadGateway)
		return
	}
	// Record locally
	g.cmdMu.Lock()
	g.cmdHistory = append(g.cmdHistory, map[string]interface{}{
		"tool_id":   req.ToolID,
		"params":    req.Params,
		"tsg_resp":  resp,
		"timestamp": time.Now().Unix(),
	})
	g.cmdMu.Unlock()
	writeJSON(w, map[string]interface{}{"sent": true, "tsg_response": resp})
}

func (g *Gateway) handleTSGCommands(w http.ResponseWriter, r *http.Request) {
	if g.tsgClient == nil {
		writeJSON(w, map[string]interface{}{"commands": []interface{}{}})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	cmds, err := g.tsgClient.PullCommands(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("pull failed: %v", err), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]interface{}{"commands": cmds})
}

func (g *Gateway) handleTSGEvents(w http.ResponseWriter, r *http.Request) {
	// Simple SSE-style event stream (v0.3.0 baseline; full WS in v0.4.0)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	// Send bridge status as first event
	status := map[string]interface{}{"event": "bridge.status", "tsg_enabled": g.cfg.TSGBridge.Enabled}
	data, _ := json.Marshal(status)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
	// Keep connection open for 3 seconds then close (demo)
	time.Sleep(3 * time.Second)
}

func (g *Gateway) handleTSGStateSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		State map[string]interface{} `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Update local cache
	g.stateMu.Lock()
	for k, v := range req.State {
		g.stateCache[k] = v
	}
	g.stateMu.Unlock()
	// Push to TSG
	if g.tsgClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		_, _ = g.tsgClient.StateSync(ctx, req.State)
	}
	// Also publish to NATS if available
	if g.natsTrans != nil && g.natsTrans.Connected() {
		_ = g.natsTrans.StateSync("push", req.State)
	}
	writeJSON(w, map[string]interface{}{"acked": true})
}

func (g *Gateway) handleTSGState(w http.ResponseWriter, r *http.Request) {
	// Pull from TSG if configured
	var tsgState map[string]interface{}
	if g.tsgClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		s, err := g.tsgClient.PullState(ctx)
		if err == nil {
			tsgState = s
		}
	}
	g.stateMu.RLock()
	local := make(map[string]interface{})
	for k, v := range g.stateCache {
		local[k] = v
	}
	g.stateMu.RUnlock()
	writeJSON(w, map[string]interface{}{
		"local": local,
		"tsg":   tsgState,
		"ts":    time.Now().Unix(),
	})
}

func (g *Gateway) handleTSGRoleBind(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		RoleID     string `json:"role_id"`
		Profession string `json:"profession"`
		RoleToken  string `json:"role_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Bind locally
	_ = g.roleMgr.RegisterIdentity(&role.Identity{
		UserID:      "tsg-client",
		RoleID:      req.RoleID,
		CompanyType: company.CompanyType(g.cfg.CompanyType),
	})
	// Push to TSG
	if g.tsgClient != nil {
		if req.RoleToken != "" {
			g.tsgClient.SetRoleToken(req.RoleToken)
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		resp, err := g.tsgClient.BindRole(ctx, req.RoleID, req.Profession)
		if err != nil {
			http.Error(w, fmt.Sprintf("tsg bind failed: %v", err), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]interface{}{"bound": true, "tsg_response": resp})
		return
	}
	writeJSON(w, map[string]interface{}{"bound": true, "local_only": true})
}

func (g *Gateway) handleTSGRoleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		UserID     string `json:"user_id"`
		ResourceID string `json:"resource_id"`
		Action     string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	localOK := g.roleMgr.Can(req.UserID, req.ResourceID, company.Action(req.Action))
	var tsgOK bool
	if g.tsgClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		ok, err := g.tsgClient.VerifyRole(ctx, req.UserID, req.ResourceID, req.Action)
		if err == nil {
			tsgOK = ok
		}
	}
	writeJSON(w, map[string]interface{}{
		"allowed":   localOK || tsgOK,
		"local":     localOK,
		"tsg":       tsgOK,
		"cross":     localOK && tsgOK,
	})
}

// Knowledge Base handlers

func (g *Gateway) handleKBSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Query       string `json:"query"`
		TopK        int    `json:"top_k"`
		CompanyType string `json:"company_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.TopK <= 0 {
		req.TopK = 5
	}
	ctype := req.CompanyType
	if ctype == "" {
		ctype = g.cfg.CompanyType
	}
	kb := g.kbMgr.InitCompanyBase(ctype)
	results, err := kb.Search(req.Query, req.TopK)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{
		"query":   req.Query,
		"results": results,
		"count":   len(results),
	})
}

func (g *Gateway) handleKBDocs(w http.ResponseWriter, r *http.Request) {
	ctype := r.URL.Query().Get("company_type")
	if ctype == "" {
		ctype = g.cfg.CompanyType
	}
	kb := g.kbMgr.InitCompanyBase(ctype)
	writeJSON(w, kb.ListDocuments())
}

func (g *Gateway) handleKBImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Dir         string `json:"dir"`
		CompanyType string `json:"company_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctype := req.CompanyType
	if ctype == "" {
		ctype = g.cfg.CompanyType
	}
	kb := g.kbMgr.InitCompanyBase(ctype)
	if err := kb.ImportFromDir(req.Dir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{
		"imported": len(kb.ListDocuments()),
		"dir":      req.Dir,
	})
}

// Obsidian handlers

func (g *Gateway) handleObsidianSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		VaultPath   string `json:"vault_path"`
		CompanyType string `json:"company_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path := req.VaultPath
	if path == "" {
		path = g.obsCfg.VaultPath
	}
	ctype := req.CompanyType
	if ctype == "" {
		ctype = g.cfg.CompanyType
	}
	kb := g.kbMgr.InitCompanyBase(ctype)
	se := obsidian.NewSyncEngine(path, kb)
	if err := se.SyncOnce(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{
		"vault":  path,
		"synced": len(kb.ListDocuments()),
		"status": "ok",
	})
}

func (g *Gateway) handleObsidianWriteBack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		DocID     string `json:"doc_id"`
		VaultPath string `json:"vault_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path := req.VaultPath
	if path == "" {
		path = g.obsCfg.VaultPath
	}
	kb, ok := g.kbMgr.GetBase("kb-" + g.cfg.CompanyType)
	if !ok {
		http.Error(w, "knowledge base not found", http.StatusNotFound)
		return
	}
	for _, doc := range kb.ListDocuments() {
		if doc.ID == req.DocID || req.DocID == "" {
			se := obsidian.NewSyncEngine(path, kb)
			_ = se.WriteBack(doc)
		}
	}
	writeJSON(w, map[string]interface{}{"status": "ok"})
}

// Voice Correction handler

func (g *Gateway) handleVoiceCorrect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := g.voiceCorr.Correct(r.Context(), req.Text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

// ReconcileOnce 执行一次双向同步：推本地 state → TSG，拉 TSG state → 本地，拉 TSG 指令
func (g *Gateway) ReconcileOnce() {
	if g.tsgClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Push local state
	g.stateMu.RLock()
	localState := make(map[string]interface{})
	for k, v := range g.stateCache {
		localState[k] = v
	}
	g.stateMu.RUnlock()
	if len(localState) > 0 {
		_, _ = g.tsgClient.StateSync(ctx, localState)
	}
	// Pull TSG state
	if s, err := g.tsgClient.PullState(ctx); err == nil && s != nil {
		g.stateMu.Lock()
		for k, v := range s {
			g.stateCache[k] = v
		}
		g.stateMu.Unlock()
	}
	// Pull commands
	if cmds, err := g.tsgClient.PullCommands(ctx); err == nil {
		g.cmdMu.Lock()
		for _, c := range cmds {
			c["pulled_at"] = time.Now().Unix()
			g.cmdHistory = append(g.cmdHistory, c)
		}
		g.cmdMu.Unlock()
	}
}

func (g *Gateway) executeTool(tool *config.ToolConfig, params map[string]interface{}) (interface{}, error) {
	if tool.Sandboxed {
		return g.sandbox.Run(nil, tool.Lang, tool.Entry, []string{})
	}
	return map[string]string{"note": "non-sandbox execution not implemented in v0.3.0"}, nil
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func langDetect() map[string]interface{} {
	return map[string]interface{}{
		"go":     true,
		"python": fileExists("python3") || fileExists("python"),
		"node":   fileExists("node"),
		"rust":   fileExists("rustc"),
		"java":   fileExists("java"),
	}
}

func fileExists(name string) bool {
	paths := []string{"/usr/bin/", "/usr/local/bin/", "/bin/"}
	for _, p := range paths {
		if _, err := http.Get("file://" + p + name); err == nil {
			return true
		}
	}
	return false
}
