package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/HP-jh/TarsEnterprise/pkg/config"
)

func TestHandleUpdateCheck(t *testing.T) {
	cfg := &config.Config{AdminToken: "secret"}
	gw := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/update/check", nil)
	rec := httptest.NewRecorder()
	gw.handleUpdateCheck(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp["current"]; !ok {
		t.Fatal("expected current in response")
	}
	if _, ok := resp["latest"]; !ok {
		t.Fatal("expected latest in response")
	}
	// has_update may be true or false depending on GitHub reachability
}

func TestHandleUpdateCheckMethodNotAllowed(t *testing.T) {
	cfg := &config.Config{AdminToken: "secret"}
	gw := New(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/update/check", nil)
	rec := httptest.NewRecorder()
	gw.handleUpdateCheck(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleUpdateApplyUnauthorized(t *testing.T) {
	cfg := &config.Config{AdminToken: "secret"}
	gw := New(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/update/apply", nil)
	rec := httptest.NewRecorder()
	gw.handleUpdateApply(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleUpdateApplyForbiddenWhenNoToken(t *testing.T) {
	cfg := &config.Config{AdminToken: ""}
	gw := New(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/update/apply", nil)
	rec := httptest.NewRecorder()
	gw.handleUpdateApply(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestHandleUpdateApplyBadMethod(t *testing.T) {
	cfg := &config.Config{AdminToken: "secret"}
	gw := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/update/apply", nil)
	rec := httptest.NewRecorder()
	gw.handleUpdateApply(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestHandleUpdateApplyWithToken(t *testing.T) {
	cfg := &config.Config{AdminToken: "secret"}
	gw := New(cfg)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/update/apply", strings.NewReader("{}"))
	req.Header.Set("X-Admin-Token", "secret")
	rec := httptest.NewRecorder()
	gw.handleUpdateApply(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Should either report no update or an error from GitHub/network
	if _, ok := resp["ok"]; !ok {
		t.Fatalf("expected ok in response: %s", rec.Body.String())
	}
}
