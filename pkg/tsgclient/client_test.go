package tsgclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_HealthCheck(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "v4.11.5"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewInsecureClient(server.Listener.Addr().String())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	hr, err := client.HealthCheck(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if hr.Status != "ok" {
		t.Errorf("expected status ok, got %s", hr.Status)
	}
	if hr.Version != "v4.11.5" {
		t.Errorf("expected version v4.11.5, got %s", hr.Version)
	}
}

func TestClient_SendCommand(t *testing.T) {
	var received map[string]interface{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/interop/receive-command", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "received"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewInsecureClient(server.Listener.Addr().String())
	client.SetRoleToken("test-token-123")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.SendCommand(ctx, "test_tool", map[string]interface{}{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if received["role_token"] != "test-token-123" {
		t.Errorf("expected role_token in request, got %v", received["role_token"])
	}
}

func TestClient_StateSync(t *testing.T) {
	var received map[string]interface{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/interop/receive-state", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "synced"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewInsecureClient(server.Listener.Addr().String())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.StateSync(ctx, map[string]interface{}{"mode": "active"})
	if err != nil {
		t.Fatal(err)
	}
	state := received["state"].(map[string]interface{})
	if state["mode"] != "active" {
		t.Errorf("expected mode active, got %v", state["mode"])
	}
}

func TestClient_FetchCommands(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/interop/commands", func(w http.ResponseWriter, r *http.Request) {
		cmds := []CommandMessage{
			{ID: "cmd-1", From: "tsg", ToolID: "tool_a", Params: map[string]interface{}{"x": 1}},
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"commands": cmds})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewInsecureClient(server.Listener.Addr().String())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmds, err := client.FetchCommands(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].ToolID != "tool_a" {
		t.Errorf("expected tool_a, got %s", cmds[0].ToolID)
	}
}

func TestClient_ValidateRoleToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/role/validate", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(RoleValidation{Valid: true, RoleID: "admin"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewInsecureClient(server.Listener.Addr().String())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rv, err := client.ValidateRoleToken(ctx, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if !rv.Valid {
		t.Error("expected valid")
	}
	if rv.RoleID != "admin" {
		t.Errorf("expected admin, got %s", rv.RoleID)
	}
}

func TestGenerateSelfSigned(t *testing.T) {
	pair, err := GenerateSelfSigned("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(pair.CertPEM) == 0 || len(pair.KeyPEM) == 0 {
		t.Error("expected cert and key")
	}
}
