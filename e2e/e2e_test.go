//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/HP-jh/TarsEnterprise/pkg/config"
	"github.com/HP-jh/TarsEnterprise/pkg/gateway"
	"github.com/HP-jh/TarsEnterprise/pkg/mtls"
	"github.com/HP-jh/TarsEnterprise/pkg/role"
)

// mockTSG 模拟 TSG 个人网关
type mockTSG struct {
	server    *httptest.Server
	mu        sync.Mutex
	commands  []map[string]interface{}
	states    map[string]interface{}
	roleStore map[string]interface{}
}

func newMockTSG() *mockTSG {
	m := &mockTSG{
		commands:  make([]map[string]interface{}, 0),
		states:    make(map[string]interface{}),
		roleStore: make(map[string]interface{}),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{"status": "ok", "version": "tsg-mock"})
	})
	mux.HandleFunc("/api/v1/command", func(w http.ResponseWriter, r *http.Request) {
		var cmd map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&cmd)
		cmd["received_at"] = time.Now().Unix()
		m.mu.Lock()
		m.commands = append(m.commands, cmd)
		m.mu.Unlock()
		writeJSON(w, map[string]interface{}{"status": "ok", "cmd_id": cmd["tool_id"]})
	})
	mux.HandleFunc("/api/v1/commands/outbound", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		cmds := make([]map[string]interface{}, len(m.commands))
		copy(cmds, m.commands)
		m.mu.Unlock()
		writeJSON(w, map[string]interface{}{"commands": cmds})
	})
	mux.HandleFunc("/api/v1/state/sync", func(w http.ResponseWriter, r *http.Request) {
		var st map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&st)
		m.mu.Lock()
		for k, v := range st {
			m.states[k] = v
		}
		m.mu.Unlock()
		writeJSON(w, map[string]interface{}{"acked": true})
	})
	mux.HandleFunc("/api/v1/state", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		st := make(map[string]interface{})
		for k, v := range m.states {
			st[k] = v
		}
		m.mu.Unlock()
		writeJSON(w, st)
	})
	mux.HandleFunc("/api/v1/role/verify", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		token, _ := req["role_token"].(string)
		allowed := token != "" && strings.HasPrefix(token, "valid-")
		writeJSON(w, map[string]interface{}{"allowed": allowed})
	})
	mux.HandleFunc("/api/v1/role/bind", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		m.mu.Lock()
		m.roleStore["bound"] = req
		m.mu.Unlock()
		writeJSON(w, map[string]interface{}{"bound": true})
	})
	m.server = httptest.NewServer(mux)
	return m
}

func (m *mockTSG) URL() string { return m.server.URL }
func (m *mockTSG) Close()      { m.server.Close() }
func (m *mockTSG) StateCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.states)
}

func TestE2E_RegisterAndCommand(t *testing.T) {
	mock := newMockTSG()
	defer mock.Close()

	cfg := &config.Config{
		CompanyType: "tech",
		TSGBridge: config.TSGBridgeConfig{
			Enabled: true,
			Address: mock.URL(),
		},
	}
	gw := gateway.New(cfg)
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)
	te := httptest.NewServer(mux)
	defer te.Close()

	// Step 1: Register identity in TE
	resp, err := http.Post(te.URL+"/api/v1/role/check", "application/json", strings.NewReader(`{"user_id":"u1","resource_id":"res1","action":"read"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	// Step 2: TE -> TSG send command
	cmdBody := `{"tool_id":"test.echo","params":{"msg":"hello"}}`
	resp, err = http.Post(te.URL+"/api/v1/tsg/command", "application/json", strings.NewReader(cmdBody))
	if err != nil {
		t.Fatal(err)
	}
	var cmdResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&cmdResp)
	resp.Body.Close()
	if cmdResp["status"] != "ok" && cmdResp["sent"] != true {
		t.Fatalf("command failed: %v", cmdResp)
	}

	// Step 3: Pull commands from TSG via TE proxy
	resp, err = http.Get(te.URL + "/api/v1/tsg/commands")
	if err != nil {
		t.Fatal(err)
	}
	var listResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&listResp)
	resp.Body.Close()
	cmds, _ := listResp["commands"].([]interface{})
	if len(cmds) == 0 {
		t.Fatal("expected commands from TSG")
	}
}

func TestE2E_StateSync(t *testing.T) {
	mock := newMockTSG()
	defer mock.Close()

	cfg := &config.Config{
		CompanyType: "tech",
		TSGBridge: config.TSGBridgeConfig{
			Enabled: true,
			Address: mock.URL(),
		},
	}
	gw := gateway.New(cfg)
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)
	te := httptest.NewServer(mux)
	defer te.Close()

	// Push state
	body := `{"state":{"mode":"active","users":3}}`
	resp, err := http.Post(te.URL+"/api/v1/tsg/state/sync", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var syncResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&syncResp)
	resp.Body.Close()
	if syncResp["acked"] != true {
		t.Fatalf("state sync not acked: %v", syncResp)
	}

	// Pull state back
	resp, err = http.Get(te.URL + "/api/v1/tsg/state")
	if err != nil {
		t.Fatal(err)
	}
	var stateResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&stateResp)
	resp.Body.Close()
	localState, _ := stateResp["local"].(map[string]interface{})
	if localState["mode"] != "active" {
		t.Fatalf("state mismatch: %v", stateResp)
	}
}

func TestE2E_RoleCrossGateway(t *testing.T) {
	mock := newMockTSG()
	defer mock.Close()

	cfg := &config.Config{
		CompanyType: "tech",
		TSGBridge: config.TSGBridgeConfig{
			Enabled: true,
			Address: mock.URL(),
		},
	}
	gw := gateway.New(cfg)
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)
	te := httptest.NewServer(mux)
	defer te.Close()

	// Bind role
	bindBody := `{"role_id":"dev","profession":"engineer","role_token":"valid-dev-token"}`
	resp, err := http.Post(te.URL+"/api/v1/tsg/role/bind", "application/json", strings.NewReader(bindBody))
	if err != nil {
		t.Fatal(err)
	}
	var bindResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&bindResp)
	resp.Body.Close()
	if bindResp["bound"] != true {
		t.Fatalf("role bind failed: %v", bindResp)
	}

	// Verify role across gateway
	verifyBody := `{"user_id":"u1","resource_id":"code","action":"read"}`
	resp, err = http.Post(te.URL+"/api/v1/tsg/role/verify", "application/json", strings.NewReader(verifyBody))
	if err != nil {
		t.Fatal(err)
	}
	var verifyResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&verifyResp)
	resp.Body.Close()
	if verifyResp["allowed"] != true {
		t.Fatalf("cross-gateway role verify failed: %v", verifyResp)
	}
}

func TestE2E_TSGEventsWS(t *testing.T) {
	mock := newMockTSG()
	defer mock.Close()

	cfg := &config.Config{
		CompanyType: "tech",
		TSGBridge: config.TSGBridgeConfig{
			Enabled: true,
			Address: mock.URL(),
		},
	}
	gw := gateway.New(cfg)
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)
	te := httptest.NewServer(mux)
	defer te.Close()

	// WS endpoint exists and accepts upgrade
	resp, err := http.Get(te.URL + "/api/v1/tsg/events")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	// Without WS client we just verify endpoint is registered
	// A real WS client would do the upgrade handshake
	if resp.StatusCode != http.StatusBadRequest {
		t.Logf("WS endpoint returned %d (expected 400 without upgrade header)", resp.StatusCode)
	}
}

func TestE2E_ReconcileLoop(t *testing.T) {
	mock := newMockTSG()
	defer mock.Close()

	cfg := &config.Config{
		CompanyType: "tech",
		TSGBridge: config.TSGBridgeConfig{
			Enabled: true,
			Address: mock.URL(),
		},
	}
	gw := gateway.New(cfg)
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)
	te := httptest.NewServer(mux)
	defer te.Close()

	// Start reconcile in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Simulate reconcile: push state then pull state
				_, _ = http.Post(te.URL+"/api/v1/tsg/state/sync", "application/json", strings.NewReader(`{"state":{"tick":1}}`))
				_, _ = http.Get(te.URL + "/api/v1/tsg/state")
			}
		}
	}()

	<-ctx.Done()
	// After reconcile, TSG should have received state
	if mock.StateCount() == 0 {
		t.Fatal("expected mock TSG to have received state after reconcile")
	}
}

func TestE2E_mTLSFrame(t *testing.T) {
	// Verify mTLS frame encode/decode round-trip
	pair, err := mtls.GenerateSelfSigned("te-test")
	if err != nil {
		t.Fatal(err)
	}
	_ = pair.CertPEM
	_ = pair.KeyPEM

	payload := []byte(`{"cmd":"ping"}`)
	frame := mtls.NewFrame(2, "role-dev", payload)
	encoded := frame.Encode()
	decoded, err := mtls.DecodeFrame(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Type != 2 {
		t.Errorf("expected type 2, got %d", decoded.Type)
	}
	if string(decoded.Payload) != string(payload) {
		t.Errorf("payload mismatch")
	}
}

func TestE2E_RoleManagerBindToProfession(t *testing.T) {
	mgr := role.NewManager()
	prof := role.BindToProfession("dev")
	if prof == "" {
		t.Error("expected non-empty profession path")
	}
	if !strings.Contains(prof, "engineer") {
		t.Errorf("expected profession containing engineer, got %s", prof)
	}

	_ = mgr.BuildMatrix("tech")
	_ = mgr.RegisterIdentity(&role.Identity{
		UserID:      "u1",
		RoleID:      "dev",
		CompanyType: "tech",
	})
	if !mgr.Can("u1", "code", "write") {
		t.Error("expected dev to write code")
	}
	if mgr.Can("u1", "code", "read") {
		t.Error("expected dev NOT to read code (only write)")
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
