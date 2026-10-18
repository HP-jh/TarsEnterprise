package mtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientCertMiddleware_NoTLS(t *testing.T) {
	called := false
	handler := ClientCertMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, ok := ClientCertFromContext(r.Context())
		if ok {
			t.Error("should not have client cert in plaintext request")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if !called {
		t.Error("handler should have been called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestClientCertMiddleware_WithCert(t *testing.T) {
	pair, _ := GenerateSelfSigned("middleware-client")
	block, _ := pem.Decode(pair.CertPEM)
	cert, _ := x509.ParseCertificate(block.Bytes)

	called := false
	handler := ClientCertMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		id, ok := ClientIdentityFromContext(r.Context())
		if !ok {
			t.Error("expected identity in context")
			return
		}
		if id.Subject == "" {
			t.Error("subject should not be empty")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{cert},
	}
	rr := httptest.NewRecorder()
	handler(rr, req)

	if !called {
		t.Error("handler should have been called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestRequireClientCert_Missing(t *testing.T) {
	handler := RequireClientCert(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != 495 {
		t.Errorf("expected 495, got %d", rr.Code)
	}
	body := rr.Body.String()
	if body == "" || !containsSub(body, "客户端证书") {
		t.Errorf("expected friendly error about client cert, got %q", body)
	}
}

func TestRequireClientCert_Present(t *testing.T) {
	pair, _ := GenerateSelfSigned("require-client")
	block, _ := pem.Decode(pair.CertPEM)
	cert, _ := x509.ParseCertificate(block.Bytes)

	called := false
	handler := RequireClientCert(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{cert},
	}
	rr := httptest.NewRecorder()
	handler(rr, req)

	if !called {
		t.Error("handler should have been called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestClientCertFromContext_NotFound(t *testing.T) {
	cert, ok := ClientCertFromContext(context.Background())
	if ok {
		t.Error("should not find cert in empty context")
	}
	if cert != nil {
		t.Error("cert should be nil")
	}
}

func TestClientIdentityFromContext_NotFound(t *testing.T) {
	id, ok := ClientIdentityFromContext(context.Background())
	if ok {
		t.Error("should not find identity in empty context")
	}
	if id.Subject != "" {
		t.Error("identity should be empty")
	}
}

func TestExtractIdentity(t *testing.T) {
	pair, _ := GenerateSelfSigned("extract-test")
	block, _ := pem.Decode(pair.CertPEM)
	cert, _ := x509.ParseCertificate(block.Bytes)

	id := extractIdentity(cert)
	if id.Subject == "" {
		t.Error("subject empty")
	}
	if id.Fingerprint == "" {
		t.Error("fingerprint empty")
	}
}

func containsSub(s, sub string) bool {
	if len(s) < len(sub) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
