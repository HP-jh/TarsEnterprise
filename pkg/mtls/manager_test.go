package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManagerFromPEM(t *testing.T) {
	pair, err := GenerateSelfSigned("test-server")
	if err != nil {
		t.Fatalf("generate self-signed: %v", err)
	}
	caPair, err := GenerateSelfSigned("test-ca")
	if err != nil {
		t.Fatalf("generate ca: %v", err)
	}

	cm, err := NewManagerFromPEM(pair.CertPEM, pair.KeyPEM, caPair.CertPEM)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	cfg := cm.TLSConfig()
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected TLS 1.3, got 0x%x", cfg.MinVersion)
	}
	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("expected RequireAndVerifyClientCert, got %v", cfg.ClientAuth)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(cfg.Certificates))
	}
}

func TestCertificateManager_ServerCertInfo(t *testing.T) {
	pair, _ := GenerateSelfSigned("info-test")
	caPair, _ := GenerateSelfSigned("info-ca")
	cm, err := NewManagerFromPEM(pair.CertPEM, pair.KeyPEM, caPair.CertPEM)
	if err != nil {
		t.Fatal(err)
	}

	info := cm.ServerCertInfo()
	if info.Subject == "" {
		t.Error("subject should not be empty")
	}
	if info.IsExpired() {
		t.Error("fresh cert should not be expired")
	}
	if info.DaysUntilExpiry() <= 0 {
		t.Error("fresh cert should have positive days until expiry")
	}
}

func TestCertificateManager_ValidateClientCert(t *testing.T) {
	caPair, _ := GenerateSelfSigned("ca")
	serverPair, _ := GenerateSelfSigned("server")
	clientPair, _ := GenerateSelfSigned("client")

	cm, err := NewManagerFromPEM(serverPair.CertPEM, serverPair.KeyPEM, caPair.CertPEM)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode(clientPair.CertPEM)
	if block == nil {
		t.Fatal("failed to decode client cert")
	}
	clientCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	// client cert not signed by our CA -> should fail
	err = cm.ValidateClientCert(clientCert)
	if err == nil {
		t.Error("expected validation to fail for untrusted client cert")
	}
}

func TestNewManager_FromFiles(t *testing.T) {
	tmp := t.TempDir()
	pair, _ := GenerateSelfSigned("file-test")
	caPair, _ := GenerateSelfSigned("file-ca")

	certPath := filepath.Join(tmp, "server.crt")
	keyPath := filepath.Join(tmp, "server.key")
	caPath := filepath.Join(tmp, "ca.crt")

	os.WriteFile(certPath, pair.CertPEM, 0644)
	os.WriteFile(keyPath, pair.KeyPEM, 0600)
	os.WriteFile(caPath, caPair.CertPEM, 0644)

	cm, err := NewManager(certPath, keyPath, caPath)
	if err != nil {
		t.Fatalf("new manager from files: %v", err)
	}

	info := cm.ServerCertInfo()
	if info.Subject == "" {
		t.Error("subject empty")
	}
}

func TestCertificateManager_Reload(t *testing.T) {
	tmp := t.TempDir()
	pair, _ := GenerateSelfSigned("reload-test")
	caPair, _ := GenerateSelfSigned("reload-ca")

	certPath := filepath.Join(tmp, "server.crt")
	keyPath := filepath.Join(tmp, "server.key")
	caPath := filepath.Join(tmp, "ca.crt")

	os.WriteFile(certPath, pair.CertPEM, 0644)
	os.WriteFile(keyPath, pair.KeyPEM, 0600)
	os.WriteFile(caPath, caPair.CertPEM, 0644)

	cm, err := NewManager(certPath, keyPath, caPath)
	if err != nil {
		t.Fatal(err)
	}

	pair2, _ := GenerateSelfSigned("reload-test-2")
	caPair2, _ := GenerateSelfSigned("reload-ca-2")
	os.WriteFile(certPath, pair2.CertPEM, 0644)
	os.WriteFile(keyPath, pair2.KeyPEM, 0600)
	os.WriteFile(caPath, caPair2.CertPEM, 0644)

	if err := cm.Reload(); err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	info := cm.ServerCertInfo()
	if info.Subject == "" {
		t.Error("subject empty after reload")
	}
}

func TestVerifyConnection_ServerCertExpired(t *testing.T) {
	pair, _ := GenerateSelfSigned("expired-server")
	caPair, _ := GenerateSelfSigned("expired-ca")
	cm, _ := NewManagerFromPEM(pair.CertPEM, pair.KeyPEM, caPair.CertPEM)

	cm.cert.Leaf.NotAfter = time.Now().Add(-24 * time.Hour)

	cfg := cm.TLSConfig()
	err := cfg.VerifyConnection(tls.ConnectionState{})
	if err == nil {
		t.Error("expected error for expired server cert")
	}
	if err != ErrServerCertExpired {
		t.Errorf("expected ErrServerCertExpired, got %v", err)
	}
}

func TestVerifyConnection_ClientCertRequired(t *testing.T) {
	pair, _ := GenerateSelfSigned("clientreq-server")
	caPair, _ := GenerateSelfSigned("clientreq-ca")
	cm, _ := NewManagerFromPEM(pair.CertPEM, pair.KeyPEM, caPair.CertPEM)

	cfg := cm.TLSConfig()
	err := cfg.VerifyConnection(tls.ConnectionState{PeerCertificates: []*x509.Certificate{}})
	if err == nil {
		t.Error("expected error when no client cert")
	}
	if err != ErrClientCertRequired {
		t.Errorf("expected ErrClientCertRequired, got %v", err)
	}
}

func TestVerifyConnection_ClientCertExpired(t *testing.T) {
	pair, _ := GenerateSelfSigned("cliexp-server")
	caPair, _ := GenerateSelfSigned("cliexp-ca")
	clientPair, _ := GenerateSelfSigned("cliexp-client")
	cm, _ := NewManagerFromPEM(pair.CertPEM, pair.KeyPEM, caPair.CertPEM)

	block, _ := pem.Decode(clientPair.CertPEM)
	clientCert, _ := x509.ParseCertificate(block.Bytes)
	clientCert.NotAfter = time.Now().Add(-24 * time.Hour)

	cfg := cm.TLSConfig()
	err := cfg.VerifyConnection(tls.ConnectionState{PeerCertificates: []*x509.Certificate{clientCert}})
	if err == nil {
		t.Error("expected error for expired client cert")
	}
	if err != ErrClientCertExpired {
		t.Errorf("expected ErrClientCertExpired, got %v", err)
	}
}

func TestFriendlyError(t *testing.T) {
	tests := []struct {
		inp  error
		want string
	}{
		{ErrServerCertExpired, "服务器安全证书已过期"},
		{ErrClientCertRequired, "连接需要客户端证书"},
		{ErrClientCertExpired, "您的身份证书已过期"},
		{ErrClientCertNotYetValid, "您的身份证书尚未生效"},
		{ErrClientCertInvalid, "您的身份证书无法通过验证"},
		{nil, ""},
	}
	for _, tc := range tests {
		got := FriendlyError(tc.inp)
		if tc.want != "" && (got == "" || !containsStr(got, tc.want)) {
			t.Errorf("FriendlyError(%v): got %q, want containing %q", tc.inp, got, tc.want)
		}
	}
}

func containsStr(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
