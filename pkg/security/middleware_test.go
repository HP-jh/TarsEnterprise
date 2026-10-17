package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRateLimit_AllowsFirstRequests(t *testing.T) {
	handler := RateLimit(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCORS_AddsHeaders(t *testing.T) {
	handler := CORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected CORS origin header")
	}
}

func TestCORS_Preflight(t *testing.T) {
	handler := CORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", rec.Code)
	}
}

func TestSecureHeaders_AddsHeaders(t *testing.T) {
	handler := SecureHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("expected X-Content-Type-Options: nosniff")
	}
}

func TestMaxBody_RejectsOversized(t *testing.T) {
	handler := MaxBody(100)(func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Body.Read(make([]byte, 200))
		if err != nil {
			http.Error(w, "body too large", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 200)))
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized body, got %d", rec.Code)
	}
}

func TestAdminAuth_AllowsValidToken(t *testing.T) {
	handler := AdminAuth("secret123")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Admin-Token", "secret123")
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAdminAuth_RejectsMissingToken(t *testing.T) {
	handler := AdminAuth("secret123")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAdminAuth_RejectsEmptyConfiguredToken(t *testing.T) {
	handler := AdminAuth("")(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when admin token not configured, got %d", rec.Code)
	}
}

func TestChain_Order(t *testing.T) {
	var order []string
	h := Chain(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	},
		func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "m1")
				next(w, r)
			}
		},
		func(next http.HandlerFunc) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				order = append(order, "m2")
				next(w, r)
			}
		},
	)
	h(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if len(order) != 3 || order[0] != "m1" || order[1] != "m2" || order[2] != "handler" {
		t.Fatalf("unexpected chain order: %v", order)
	}
}
