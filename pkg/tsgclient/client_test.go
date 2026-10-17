package tsgclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_HealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			writeJSON(w, map[string]interface{}{"status": "ok"})
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := New(srv.URL)
	resp, err := c.HealthCheck(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resp["status"])
	}
}

func TestClient_SendCommand(t *testing.T) {
	var received map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/command" {
			_ = json.NewDecoder(r.Body).Decode(&received)
			writeJSON(w, map[string]interface{}{"status": "ok", "tool_id": received["tool_id"]})
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := New(srv.URL)
	resp, err := c.SendCommand(context.Background(), "test-tool", map[string]interface{}{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected ok, got %v", resp)
	}
}

func TestClient_StateSync(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/state/sync" {
			writeJSON(w, map[string]interface{}{"acked": true})
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := New(srv.URL)
	resp, err := c.StateSync(context.Background(), map[string]interface{}{"mode": "active"})
	if err != nil {
		t.Fatal(err)
	}
	if resp["acked"] != true {
		t.Errorf("expected acked true, got %v", resp)
	}
}

func TestClient_RoleTokenHeader(t *testing.T) {
	var gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Role-Token")
		writeJSON(w, map[string]interface{}{"token": gotToken})
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.SetRoleToken("my-role-token")
	_, err := c.HealthCheck(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotToken != "my-role-token" {
		t.Errorf("expected token my-role-token, got %s", gotToken)
	}
}

func TestClient_VerifyRole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/role/verify" {
			writeJSON(w, map[string]interface{}{"allowed": true})
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := New(srv.URL)
	allowed, err := c.VerifyRole(context.Background(), "u1", "res1", "read")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("expected allowed")
	}
}

func TestClient_BindRole(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/role/bind" {
			writeJSON(w, map[string]interface{}{"bound": true})
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := New(srv.URL)
	resp, err := c.BindRole(context.Background(), "dev", "engineer")
	if err != nil {
		t.Fatal(err)
	}
	if resp["bound"] != true {
		t.Errorf("expected bound true, got %v", resp)
	}
}

func TestClient_PullCommands(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/commands/outbound" {
			writeJSON(w, map[string]interface{}{
				"commands": []map[string]interface{}{
					{"id": "cmd1", "tool_id": "t1"},
				},
			})
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := New(srv.URL)
	cmds, err := c.PullCommands(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0]["id"] != "cmd1" {
		t.Errorf("expected cmd1, got %v", cmds[0]["id"])
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
