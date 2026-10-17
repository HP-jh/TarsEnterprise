//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/HP-jh/TarsEnterprise/pkg/config"
	"github.com/HP-jh/TarsEnterprise/pkg/gateway"
)

// mockTSG 模拟 TSG 个人网关
type mockTSG struct {
	server   *httptest.Server
	commands []map[string]interface{}
	state    map[string]interface{}
}

func newMockTSG() *mockTSG {
	m := &mockTSG{
		commands: make([]map[string]interface{}, 0),
		state:    make(map[string]interface{}),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "v4.11.5"})
	})
	mux.HandleFunc("/api/v1/interop/receive-command", func(w http.ResponseWriter, r *http.Request) {
		var cmd map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&cmd)
		m.commands = append(m.commands, cmd)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "received"})
	})
	mux.HandleFunc("/api/v1/interop/receive-state", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if s, ok := body["state"].(map[string]interface{}); ok {
			for k, v := range s {
				m.state[k] = v
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
	})
	mux.HandleFunc("/api/v1/interop/commands", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"commands": m.commands})
	})
	mux.HandleFunc("/api/v1/role/validate", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		valid := body["role_token"] == "valid-tok"
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": valid, "role_id": "admin",
		})
	})
	m.server = httptest.NewServer(mux)
	return m
}

func (m *mockTSG) Close() {
	m.server.Close()
}

func TestInterop_FullLoop(t *testing.T) {
	// 1. 启动 mock TSG
	mock := newMockTSG()
	defer mock.Close()

	// 2. 启动 TE gateway（连接 mock TSG）
	cfg := &config.Config{
		CompanyType: "tech",
		TSGBridge: config.TSGBridgeConfig{
			Enabled: true,
			Address: mock.server.Listener.Addr().String(),
			NATSURL: "memory://",
		},
	}
	gw := gateway.New(cfg)
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)
	teServer := httptest.NewServer(mux)
	defer teServer.Close()

	teBase := teServer.URL
	client := &http.Client{Timeout: 5 * time.Second}

	// Step 1: 注册身份 + 角色绑定
	bindBody := map[string]interface{}{
		"user_id":    "u001",
		"role_id":    "admin",
		"tsg_token":  "valid-tok",
		"company_id": "c001",
	}
	resp := post(t, client, teBase+"/api/v1/tsg/role/bind", bindBody)
	assertStatus(t, resp, 200)
	var bindRes map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&bindRes)
	if bindRes["bound"] != true {
		t.Fatalf("expected bound=true, got %v", bindRes["bound"])
	}

	// Step 2: 跨网关发指令 (TE → TSG)
	cmdBody := map[string]interface{}{
		"tool_id":    "test_tool",
		"params":     map[string]interface{}{"action": "ping"},
		"role_token": "valid-tok",
	}
	resp = post(t, client, teBase+"/api/v1/tsg/command", cmdBody)
	assertStatus(t, resp, 200)
	var cmdRes map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&cmdRes)
	if cmdRes["status"] != "delivered" && cmdRes["status"] != "pending" {
		t.Fatalf("expected delivered or pending, got %v", cmdRes["status"])
	}

	// Step 3: 推 state (TE → TSG)
	stateBody := map[string]interface{}{
		"state": map[string]interface{}{
			"mode":    "active",
			"version": "v0.3.0",
		},
		"role_token": "valid-tok",
	}
	resp = post(t, client, teBase+"/api/v1/tsg/state/sync", stateBody)
	assertStatus(t, resp, 200)
	var syncRes map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&syncRes)
	if syncRes["status"] != "synced" {
		t.Fatalf("expected synced, got %v", syncRes["status"])
	}

	// Step 4: 拉 TSG 指令列表
	resp = get(t, client, teBase+"/api/v1/tsg/commands")
	assertStatus(t, resp, 200)
	var cmdsRes map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&cmdsRes)
	cmds := cmdsRes["commands"].([]interface{})
	// inbox 此时可能为空（TSG→TE 指令需要 TSG 主动发）
	_ = cmds

	// Step 5: 拉本地 state
	resp = get(t, client, teBase+"/api/v1/tsg/state")
	assertStatus(t, resp, 200)
	var stateRes map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&stateRes)
	state := stateRes["state"].(map[string]interface{})
	if state["mode"] != "active" {
		t.Fatalf("expected mode=active in state, got %v", state["mode"])
	}

	// Step 6: 跨网关角色验证
	roleBody := map[string]interface{}{
		"user_id":     "u001",
		"resource_id": "dashboard",
		"action":      "read",
		"role_token":  "valid-tok",
	}
	resp = post(t, client, teBase+"/api/v1/role/check", roleBody)
	assertStatus(t, resp, 200)
	var roleRes map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&roleRes)
	if roleRes["allowed"] != true {
		t.Fatalf("expected allowed=true, got %v", roleRes)
	}

	// Step 7: 验证 TSG 收到了指令和 state
	if len(mock.commands) == 0 {
		t.Fatal("expected TSG to have received at least 1 command")
	}
	if mock.state["mode"] != "active" {
		t.Fatalf("expected TSG state mode=active, got %v", mock.state["mode"])
	}

	fmt.Println("[e2e] Full interop loop verified: identity → command → state sync → role check → consistency")
}

func TestInterop_RoleCheck_LocalOnly(t *testing.T) {
	cfg := &config.Config{
		CompanyType: "tech",
		TSGBridge:   config.TSGBridgeConfig{Enabled: false},
	}
	gw := gateway.New(cfg)
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)
	teServer := httptest.NewServer(mux)
	defer teServer.Close()

	client := &http.Client{Timeout: 5 * time.Second}

	// 本地角色检查（无 role_token）
	roleBody := map[string]interface{}{
		"user_id":     "u001",
		"resource_id": "dashboard",
		"action":      "read",
	}
	resp := post(t, client, teServer.URL+"/api/v1/role/check", roleBody)
	assertStatus(t, resp, 200)
	var roleRes map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&roleRes)
	if roleRes["allowed"] != false {
		t.Fatalf("expected allowed=false for unregistered user, got %v", roleRes)
	}
}

func TestInterop_StateReconcile(t *testing.T) {
	mock := newMockTSG()
	defer mock.Close()

	cfg := &config.Config{
		CompanyType: "tech",
		TSGBridge: config.TSGBridgeConfig{
			Enabled: true,
			Address: mock.server.Listener.Addr().String(),
			NATSURL: "memory://",
		},
	}
	gw := gateway.New(cfg)
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)
	teServer := httptest.NewServer(mux)
	defer teServer.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	teBase := teServer.URL

	// 连续推送多个 state
	for i := 0; i < 3; i++ {
		stateBody := map[string]interface{}{
			"state": map[string]interface{}{
				"counter": float64(i),
				"ts":      time.Now().Unix(),
			},
		}
		resp := post(t, client, teBase+"/api/v1/tsg/state/sync", stateBody)
		assertStatus(t, resp, 200)
	}

	// 验证最终 state 一致
	resp := get(t, client, teBase+"/api/v1/tsg/state")
	assertStatus(t, resp, 200)
	var stateRes map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&stateRes)
	state := stateRes["state"].(map[string]interface{})
	if state["counter"] != float64(2) {
		t.Fatalf("expected counter=2, got %v", state["counter"])
	}
	if mock.state["counter"] != float64(2) {
		t.Fatalf("expected TSG counter=2, got %v", mock.state["counter"])
	}
}

// --- helpers ---

func post(t *testing.T, client *http.Client, url string, body interface{}) *http.Response {
	data, _ := json.Marshal(body)
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST %s failed: %v", url, err)
	}
	return resp
}

func get(t *testing.T, client *http.Client, url string) *http.Response {
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s failed: %v", url, err)
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, want int) {
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d: %s", want, resp.StatusCode, string(body))
	}
}
