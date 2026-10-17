package mcp

import (
	"context"
	"testing"
)

func TestHub_RegisterAndList(t *testing.T) {
	h := NewHub()
	h.Register(&Server{ID: "s1", Name: "Server 1", Tools: []Tool{{Name: "t1"}}})
	h.Register(&Server{ID: "s2", Name: "Server 2", Tools: []Tool{{Name: "t2"}}})

	list := h.ListServers()
	if len(list) != 2 {
		t.Errorf("expected 2 servers, got %d", len(list))
	}
}

func TestHub_Call(t *testing.T) {
	h := NewHub()
	h.Register(&Server{ID: "s1", Tools: []Tool{{Name: "t1"}}})

	result, err := h.Call(context.Background(), "s1", "t1", map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != "simulated" {
		t.Error("expected simulated status")
	}
}

func TestHub_Call_NotFound(t *testing.T) {
	h := NewHub()
	_, err := h.Call(context.Background(), "x", "y", nil)
	if err == nil {
		t.Error("expected error")
	}
}

func TestOpenClawAdapter_Register(t *testing.T) {
	h := NewHub()
	oca := NewOpenClawAdapter(h)
	oca.RegisterAsMCP("oc1", "http://localhost:8080")

	list := h.ListServers()
	if len(list) != 1 {
		t.Fatalf("expected 1 server, got %d", len(list))
	}
	if list[0].Meta["provider"] != "openclaw" {
		t.Error("provider mismatch")
	}
}

func TestOpenClawAdapter_Mitigate(t *testing.T) {
	h := NewHub()
	oca := NewOpenClawAdapter(h)
	m, ok := oca.Mitigate("CVE-2025-XXXX")
	if !ok {
		t.Error("expected mitigation")
	}
	if m == "" {
		t.Error("empty mitigation")
	}
}
