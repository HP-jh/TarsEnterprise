package gateway

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/HP-jh/TarsEnterprise/pkg/company"
	"github.com/HP-jh/TarsEnterprise/pkg/config"
	"github.com/HP-jh/TarsEnterprise/pkg/health"
	"github.com/HP-jh/TarsEnterprise/pkg/i18n"
	"github.com/HP-jh/TarsEnterprise/pkg/knowledge"
	"github.com/HP-jh/TarsEnterprise/pkg/mcp"
	"github.com/HP-jh/TarsEnterprise/pkg/metrics"
	"github.com/HP-jh/TarsEnterprise/pkg/obsidian"
	"github.com/HP-jh/TarsEnterprise/pkg/role"
	"github.com/HP-jh/TarsEnterprise/pkg/sandbox"
	"github.com/HP-jh/TarsEnterprise/pkg/security"
	"github.com/HP-jh/TarsEnterprise/pkg/voicecorrection"
)

type Gateway struct {
	cfg           *config.Config
	roleMgr       *role.Manager
	mcpHub        *mcp.Hub
	sandbox       *sandbox.Manager
	kbMgr         *knowledge.Manager
	obsCfg        *obsidian.ConnectorConfig
	voiceCorr     *voicecorrection.Corrector
	tsgInterop    *TSGInterop
	healthReg     *health.Registry
}

func New(cfg *config.Config) *Gateway {
	gw := &Gateway{
		cfg:       cfg,
		roleMgr:   role.NewManager(),
		mcpHub:    mcp.NewHub(),
		sandbox:   sandbox.NewManager(),
		kbMgr:     knowledge.NewManager(),
		obsCfg:    obsidian.DefaultConnectorConfig(),
		voiceCorr: voicecorrection.NewCorrector(""),
	}
	// Pre-build role matrix for configured company type
	if cfg.CompanyType != "" {
		_ = gw.roleMgr.BuildMatrix(company.CompanyType(cfg.CompanyType))
		// Init knowledge base for company type
		gw.kbMgr.InitCompanyBase(cfg.CompanyType)
	}
	// Init TSG interop
	gw.tsgInterop = newTSGInterop(cfg.TSGBridge)
	// Build health check registry
	gw.healthReg = gw.buildHealthRegistry()
	return gw
}

func (g *Gateway) RegisterRoutes(mux *http.ServeMux) {
	// Common middleware stack for API endpoints
	apiChain := func(h http.HandlerFunc) http.HandlerFunc {
		return security.Chain(h,
			security.MaxBody(8<<20),
			security.SecureHeaders,
			security.CORS,
			security.RateLimit,
		)
	}

	// Health
	mux.HandleFunc("/health", apiChain(g.handleHealth))
	mux.HandleFunc("/api/v1/health", apiChain(g.handleHealthV1))
	// Metrics (Prometheus)
	mux.HandleFunc("/metrics", apiChain(metrics.HTTPHandler()))
	// API v1
	mux.HandleFunc("/api/v1/company/schema", apiChain(g.handleCompanySchema))
	mux.HandleFunc("/api/v1/role/check", apiChain(g.handleRoleCheckCross))
	mux.HandleFunc("/api/v1/tools", apiChain(g.handleTools))
	mux.HandleFunc("/api/v1/tools/call", apiChain(g.handleToolCall))
	mux.HandleFunc("/api/v1/mcp/servers", apiChain(g.handleMCPServers))
	mux.HandleFunc("/api/v1/mcp/call", apiChain(g.handleMCPCall))
	mux.HandleFunc("/api/v1/sandbox/run", apiChain(g.handleSandboxRun))
	mux.HandleFunc("/api/v1/config/merge", apiChain(g.handleConfigMerge))
	mux.HandleFunc("/api/v1/lang/detect", apiChain(g.handleLangDetect))
	mux.HandleFunc("/api/v1/tsg/bridge", apiChain(g.handleTSGBridge))
	// TSG interop v0.3.0 endpoints
	mux.HandleFunc("/api/v1/tsg/command", apiChain(g.handleTSGCommand))
	mux.HandleFunc("/api/v1/tsg/commands", apiChain(g.handleTSGCommands))
	mux.HandleFunc("/api/v1/tsg/events", apiChain(g.handleTSGEvents))
	mux.HandleFunc("/api/v1/tsg/state/sync", apiChain(g.handleTSGStateSync))
	mux.HandleFunc("/api/v1/tsg/state", apiChain(g.handleTSGState))
	mux.HandleFunc("/api/v1/tsg/role/bind", apiChain(g.handleTSGRoleBind))
	// Knowledge Base
	mux.HandleFunc("/api/v1/kb/search", apiChain(g.handleKBSearch))
	mux.HandleFunc("/api/v1/kb/docs", apiChain(g.handleKBDocs))
	mux.HandleFunc("/api/v1/kb/import", apiChain(g.handleKBImport))
	// Obsidian
	mux.HandleFunc("/api/v1/obsidian/sync", apiChain(g.handleObsidianSync))
	mux.HandleFunc("/api/v1/obsidian/writeback", apiChain(g.handleObsidianWriteBack))
	// Voice Correction
	mux.HandleFunc("/api/v1/voice/correct", apiChain(g.handleVoiceCorrect))
	// mTLS 零信任状态（公开查询当前 mTLS 配置）
	mux.HandleFunc("/api/v1/mtls/status", apiChain(g.handleMTLSStatus))
	// Auto-update endpoints (check public, apply admin-only)
	mux.HandleFunc("/api/v1/update/check", apiChain(g.handleUpdateCheck))
	mux.HandleFunc("/api/v1/update/apply", security.Chain(g.handleUpdateApply,
		security.MaxBody(1<<20),
		security.SecureHeaders,
		security.CORS,
		security.RateLimit,
		security.AdminAuth(g.cfg.AdminToken),
	))
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

func (g *Gateway) handleHealthV1(w http.ResponseWriter, r *http.Request) {
	report := g.healthReg.Run(config.Version)
	writeJSON(w, report)
}

func (g *Gateway) buildHealthRegistry() *health.Registry {
	reg := health.NewRegistry()

	// mTLS health
	reg.Register(func() health.Check {
		if !g.cfg.MTLS.Enabled {
			return health.Healthy("mTLS", "mTLS 未启用，使用明文传输（生产环境建议开启）")
		}
		return health.Healthy("mTLS", fmt.Sprintf("双向认证已启用，模式: %s", g.cfg.MTLS.Mode))
	})

	// TSG Bridge health
	reg.Register(func() health.Check {
		if !g.cfg.TSGBridge.Enabled {
			return health.Healthy("TSG_Bridge", "TSG 网关互通未启用")
		}
		if g.tsgInterop == nil || g.tsgInterop.tsClient == nil {
			return health.Unhealthy("TSG_Bridge", fmt.Errorf("TSG 客户端未初始化"))
		}
		return health.Healthy("TSG_Bridge", fmt.Sprintf("TSG 互通已连接: %s", g.cfg.TSGBridge.Address))
	})

	// Knowledge Base health
	reg.Register(func() health.Check {
		if g.cfg.CompanyType == "" {
			return health.Unknown("KnowledgeBase", "公司类型未设置，知识库未初始化")
		}
		return health.Healthy("KnowledgeBase", fmt.Sprintf("知识库已初始化，公司类型: %s", g.cfg.CompanyType))
	})

	// MCP Hub health
	reg.Register(func() health.Check {
		if !g.cfg.MCP.Enabled {
			return health.Healthy("MCP_Hub", "MCP 未启用")
		}
		servers := g.mcpHub.ListServers()
		return health.Healthy("MCP_Hub", fmt.Sprintf("MCP Hub 运行中，已注册 %d 个服务器", len(servers)))
	})

	// Sandbox health
	reg.Register(func() health.Check {
		if !g.cfg.Sandbox.Enabled {
			return health.Healthy("Sandbox", "沙箱未启用")
		}
		return health.Healthy("Sandbox", fmt.Sprintf("沙箱已启用，运行时: %s", g.cfg.Sandbox.Runtime))
	})

	// Metrics health
	reg.Register(func() health.Check {
		if !g.cfg.Metrics.Enabled {
			return health.Healthy("Metrics", "监控指标未启用")
		}
		return health.Healthy("Metrics", fmt.Sprintf("监控指标已启用，端口: %d", g.cfg.Metrics.Port))
	})

	// Role Manager health
	reg.Register(func() health.Check {
		return health.Healthy("Role_Manager", "角色管理器运行正常")
	})

	// Voice Correction health
	reg.Register(func() health.Check {
		return health.Healthy("VoiceCorrection", "语音纠错模块就绪")
	})

	// Config validation health
	reg.Register(func() health.Check {
		validationErrs := g.cfg.Validate()
		if len(validationErrs) == 0 {
			return health.Healthy("Config", "配置验证通过，一切正常")
		}
		msgs := make([]string, 0, len(validationErrs))
		for _, e := range validationErrs {
			msgs = append(msgs, e.FriendlyMsg)
		}
		return health.Check{
			Name:        "Config",
			Status:      health.StatusDegraded,
			Message:     fmt.Sprintf("发现 %d 个配置建议", len(validationErrs)),
			FriendlyMsg: fmt.Sprintf("配置检查发现了 %d 个可以优化的地方：%v", len(validationErrs), msgs),
			CheckedAt:   health.Healthy("").CheckedAt,
		}
	})

	return reg
}

func (g *Gateway) handleMTLSStatus(w http.ResponseWriter, r *http.Request) {
	// 从 TLS 连接中提取客户端身份信息（如果存在）
	clientInfo := map[string]interface{}{"present": false}
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		cert := r.TLS.PeerCertificates[0]
		clientInfo = map[string]interface{}{
			"present":    true,
			"subject":    cert.Subject.String(),
			"issuer":     cert.Issuer.String(),
			"serial":     cert.SerialNumber.String(),
			"not_after":  cert.NotAfter.Unix(),
			"dns_names":  cert.DNSNames,
		}
	}
	writeJSON(w, map[string]interface{}{
		"enabled":     g.cfg.MTLS.Enabled,
		"mode":        g.cfg.MTLS.Mode,
		"client_auth": g.cfg.MTLS.ClientAuth,
		"tls_version": func() string {
			if r.TLS != nil {
				switch r.TLS.Version {
				case tls.VersionTLS13:
					return "TLS 1.3"
				case tls.VersionTLS12:
					return "TLS 1.2"
				default:
					return "unknown"
				}
			}
			return "none"
		}(),
		"client": clientInfo,
	})
}

func (g *Gateway) handleCompanySchema(w http.ResponseWriter, r *http.Request) {
	ctype := r.URL.Query().Get("type")
	if ctype == "" {
		ctype = g.cfg.CompanyType
	}
	schema, err := company.GetSchema(company.CompanyType(ctype))
	if err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, schema)
}

func (g *Gateway) handleRoleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		security.FriendlyMethodError(w, "POST")
		return
	}
	var req struct {
		UserID     string `json:"user_id"`
		ResourceID string `json:"resource_id"`
		Action     string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
		return
	}
	ok := g.roleMgr.Can(req.UserID, req.ResourceID, company.Action(req.Action))
	writeJSON(w, map[string]bool{"allowed": ok})
}

func (g *Gateway) handleTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, g.cfg.Tools)
}

func (g *Gateway) handleToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		security.FriendlyMethodError(w, "POST")
		return
	}
	var req struct {
		ToolID   string                 `json:"tool_id"`
		Params   map[string]interface{} `json:"params"`
		ClientID string                 `json:"client_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Find tool
	var tool *config.ToolConfig
	for i := range g.cfg.Tools {
		if g.cfg.Tools[i].ID == req.ToolID {
			tool = &g.cfg.Tools[i]
			break
		}
	}
	if tool == nil {
		i18n.FriendlyError(w, http.StatusNotFound, "tool not found")
		return
	}
	// Record metrics
	metrics.RecordAgentCall(req.ToolID, req.ClientID)
	// Execute (sandboxed if configured)
	result, err := g.executeTool(tool, req.Params)
	if err != nil {
		metrics.RecordAgentError(req.ToolID)
		i18n.FriendlyError(w, http.StatusInternalServerError, fmt.Sprintf("tool execution failed: %v", err))
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
		security.FriendlyMethodError(w, "POST")
		return
	}
	var req struct {
		Server string                 `json:"server"`
		Tool   string                 `json:"tool"`
		Params map[string]interface{} `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := g.mcpHub.Call(r.Context(), req.Server, req.Tool, req.Params)
	if err != nil {
		i18n.FriendlyError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, result)
}

func (g *Gateway) handleSandboxRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		security.FriendlyMethodError(w, "POST")
		return
	}
	var req struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Lang    string   `json:"lang"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Security guard: validate command against blacklist
	if err := sandbox.ValidateCommand(req.Command, req.Args); err != nil {
		i18n.FriendlyError(w, http.StatusForbidden, err.Error())
		return
	}
	result, err := g.sandbox.Run(r.Context(), req.Lang, req.Command, req.Args)
	if err != nil {
		i18n.FriendlyError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, result)
}

func (g *Gateway) handleConfigMerge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		security.FriendlyMethodError(w, "POST")
		return
	}
	var req struct {
		Modules []map[string]interface{} `json:"modules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
		return
	}
	merged, collisions, err := config.MergeModules(req.Modules)
	if err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
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
	writeJSON(w, map[string]interface{}{
		"enabled": g.cfg.TSGBridge.Enabled,
		"address": g.cfg.TSGBridge.Address,
		"status":  "connected", // simplified
	})
}

func (g *Gateway) executeTool(tool *config.ToolConfig, params map[string]interface{}) (interface{}, error) {
	if tool.Sandboxed {
		return g.sandbox.Run(nil, tool.Lang, tool.Entry, []string{})
	}
	return map[string]string{"note": "non-sandbox execution not implemented in v0.2.0"}, nil
}

// Knowledge Base handlers
func (g *Gateway) handleKBSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		security.FriendlyMethodError(w, "POST")
		return
	}
	var req struct {
		Query       string `json:"query"`
		TopK        int    `json:"top_k"`
		CompanyType string `json:"company_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
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
		i18n.FriendlyError(w, http.StatusInternalServerError, err.Error())
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
		security.FriendlyMethodError(w, "POST")
		return
	}
	var req struct {
		Dir         string `json:"dir"`
		CompanyType string `json:"company_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctype := req.CompanyType
	if ctype == "" {
		ctype = g.cfg.CompanyType
	}
	kb := g.kbMgr.InitCompanyBase(ctype)
	if err := kb.ImportFromDir(req.Dir); err != nil {
		i18n.FriendlyError(w, http.StatusInternalServerError, err.Error())
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
		security.FriendlyMethodError(w, "POST")
		return
	}
	var req struct {
		VaultPath   string `json:"vault_path"`
		CompanyType string `json:"company_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
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
		i18n.FriendlyError(w, http.StatusInternalServerError, err.Error())
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
		security.FriendlyMethodError(w, "POST")
		return
	}
	var req struct {
		DocID     string `json:"doc_id"`
		VaultPath string `json:"vault_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
		return
	}
	path := req.VaultPath
	if path == "" {
		path = g.obsCfg.VaultPath
	}
	kb, ok := g.kbMgr.GetBase("kb-" + g.cfg.CompanyType)
	if !ok {
		i18n.FriendlyError(w, http.StatusNotFound, "knowledge base not found")
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
		security.FriendlyMethodError(w, "POST")
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		i18n.FriendlyError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := g.voiceCorr.Correct(r.Context(), req.Text)
	if err != nil {
		i18n.FriendlyError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, result)
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
		if _, err := exec.LookPath(name); err == nil {
			return true
		}
		if _, err := os.Stat(p + name); err == nil {
			return true
		}
	}
	return false
}
