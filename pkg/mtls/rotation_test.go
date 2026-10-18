package mtls

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotationManager_Initialize_AutoGen(t *testing.T) {
	dir := t.TempDir()
	rm := NewRotationManager(
		filepath.Join(dir, "cert.pem"),
		filepath.Join(dir, "key.pem"),
		filepath.Join(dir, "ca.pem"),
		2*time.Hour,
		30*time.Minute,
	)
	if err := rm.Initialize(); err != nil {
		t.Fatalf("initialize failed: %v", err)
	}
	cfg := rm.GetTLSConfig()
	if cfg == nil {
		t.Fatal("expected tls.Config, got nil")
	}
	if cfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Error("expected RequireAndVerifyClientCert")
	}
	if cfg.MinVersion != tls.VersionTLS13 {
		t.Error("expected TLS1.3")
	}
}

func TestRotationManager_RotateNow(t *testing.T) {
	dir := t.TempDir()
	rm := NewRotationManager(
		filepath.Join(dir, "cert.pem"),
		filepath.Join(dir, "key.pem"),
		filepath.Join(dir, "ca.pem"),
		2*time.Hour,
		30*time.Minute,
	)
	if err := rm.RotateNow(); err != nil {
		t.Fatalf("rotate failed: %v", err)
	}
	// 检查文件已写入
	for _, f := range []string{"cert.pem", "key.pem", "ca.pem"} {
		p := filepath.Join(dir, f)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected file %s to exist: %v", f, err)
		}
	}
	// 检查 expiry 已设置
	if rm.Expiry().Before(time.Now().Add(time.Hour)) {
		t.Error("expected expiry to be in the future")
	}
}

func TestRotationManager_LoadExisting(t *testing.T) {
	dir := t.TempDir()
	// 先创建一套证书
	rm1 := NewRotationManager(
		filepath.Join(dir, "cert.pem"),
		filepath.Join(dir, "key.pem"),
		filepath.Join(dir, "ca.pem"),
		2*time.Hour,
		30*time.Minute,
	)
	if err := rm1.RotateNow(); err != nil {
		t.Fatalf("first rotate failed: %v", err)
	}
	// 再用新 manager 加载
	rm2 := NewRotationManager(
		filepath.Join(dir, "cert.pem"),
		filepath.Join(dir, "key.pem"),
		filepath.Join(dir, "ca.pem"),
		2*time.Hour,
		30*time.Minute,
	)
	if err := rm2.Initialize(); err != nil {
		t.Fatalf("load existing failed: %v", err)
	}
	if rm2.GetTLSConfig() == nil {
		t.Fatal("expected tls.Config after loading existing certs")
	}
}

func TestRotationManager_IsExpiringSoon(t *testing.T) {
	rm := NewRotationManager("", "", "", 2*time.Hour, 30*time.Minute)
	_ = rm.RotateNow()
	if rm.IsExpiringSoon() {
		t.Error("fresh cert should not be expiring soon")
	}
	// 手动把 nextRotate 推到未来很近
	rm.mu.Lock()
	rm.nextRotate = time.Now().Add(10 * time.Minute)
	rm.mu.Unlock()
	if !rm.IsExpiringSoon() {
		t.Error("cert within window should be expiring soon")
	}
}

func TestRotationManager_StartStop(t *testing.T) {
	dir := t.TempDir()
	rm := NewRotationManager(
		filepath.Join(dir, "cert.pem"),
		filepath.Join(dir, "key.pem"),
		filepath.Join(dir, "ca.pem"),
		2*time.Hour,
		30*time.Minute,
	)
	_ = rm.Initialize()
	rm.Start(100 * time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	rm.Stop()
	// 只要能正常停止即通过
}

func TestRotationManager_ParseCertificate(t *testing.T) {
	rm := NewRotationManager("", "", "", 2*time.Hour, 30*time.Minute)
	_ = rm.RotateNow()
	cert, err := rm.ParseCertificate()
	if err != nil {
		t.Fatalf("parse cert failed: %v", err)
	}
	if cert.Subject.CommonName != "tarsenterprise.auto" {
		t.Errorf("expected CN tarsenterprise.auto, got %s", cert.Subject.CommonName)
	}
}
