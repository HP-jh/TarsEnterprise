package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HP-jh/TarsEnterprise/pkg/config"
)

func TestGateway_Health(t *testing.T) {
	gw := New(&config.Config{})
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status %d", rec.Code)
	}
	var body map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Error("status not ok")
	}
}

func TestGateway_CompanySchema(t *testing.T) {
	gw := New(&config.Config{CompanyType: "tech"})
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/company/schema?type=education", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status %d", rec.Code)
	}
}

func TestGateway_Tools(t *testing.T) {
	gw := New(&config.Config{Tools: []config.ToolConfig{{ID: "t1", Name: "Tool1"}}})
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tools", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status %d", rec.Code)
	}
}

func TestGateway_ConfigMerge(t *testing.T) {
	gw := New(&config.Config{})
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]interface{}{
		"modules": []map[string]interface{}{{"a": 1}, {"b": 2}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/merge", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status %d", rec.Code)
	}
}

func TestGateway_LangDetect(t *testing.T) {
	gw := New(&config.Config{})
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/lang/detect", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status %d", rec.Code)
	}
}
