package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RotationManager 证书自动轮换管理器
type RotationManager struct {
	mu         sync.RWMutex
	certPath   string
	keyPath    string
	caPath     string
	tlsConfig  *tls.Config
	pair       *Pair
	caPEM      []byte
	nextRotate time.Time
	stopCh     chan struct{}
	interval   time.Duration
	window     time.Duration
	commonName string
}

// NewRotationManager 创建轮换管理器（interval=证书有效期, window=过期前多久轮换）
func NewRotationManager(certPath, keyPath, caPath string, interval, window time.Duration) *RotationManager {
	if interval <= 0 {
		interval = 720 * time.Hour // 30天
	}
	if window <= 0 {
		window = 168 * time.Hour // 7天
	}
	return &RotationManager{
		certPath:   certPath,
		keyPath:    keyPath,
		caPath:     caPath,
		interval:   interval,
		window:     window,
		commonName: "tarsenterprise.auto",
		stopCh:     make(chan struct{}),
	}
}

// Initialize 加载现有证书或自动生成
func (rm *RotationManager) Initialize() error {
	// 尝试从文件加载
	if rm.certPath != "" && rm.keyPath != "" {
		if _, err := os.Stat(rm.certPath); err == nil {
			if _, err := os.Stat(rm.keyPath); err == nil {
				certPEM, err := os.ReadFile(rm.certPath)
				if err != nil {
					return fmt.Errorf("read cert: %w", err)
				}
				keyPEM, err := os.ReadFile(rm.keyPath)
				if err != nil {
					return fmt.Errorf("read key: %w", err)
				}
				rm.pair = &Pair{CertPEM: certPEM, KeyPEM: keyPEM}
				if rm.caPath != "" {
					caPEM, err := os.ReadFile(rm.caPath)
					if err == nil {
						rm.caPEM = caPEM
					}
				}
				return rm.buildTLSConfig()
			}
		}
	}
	// 文件不存在则自动生成
	return rm.RotateNow()
}

// RotateNow 立即生成新证书并持久化
func (rm *RotationManager) RotateNow() error {
	pair, err := GenerateSelfSigned(rm.commonName)
	if err != nil {
		return fmt.Errorf("generate cert: %w", err)
	}
	rm.mu.Lock()
	rm.pair = pair
	rm.caPEM = pair.CertPEM // 自签名场景下 CA 即自身证书
	rm.nextRotate = time.Now().Add(rm.interval)
	rm.mu.Unlock()

	// 持久化到文件（如果配置了路径）
	if rm.certPath != "" {
		_ = os.MkdirAll(filepath.Dir(rm.certPath), 0750)
		if err := os.WriteFile(rm.certPath, pair.CertPEM, 0644); err != nil {
			return fmt.Errorf("write cert: %w", err)
		}
	}
	if rm.keyPath != "" {
		_ = os.MkdirAll(filepath.Dir(rm.keyPath), 0750)
		if err := os.WriteFile(rm.keyPath, pair.KeyPEM, 0600); err != nil {
			return fmt.Errorf("write key: %w", err)
		}
	}
	if rm.caPath != "" {
		_ = os.MkdirAll(filepath.Dir(rm.caPath), 0750)
		if err := os.WriteFile(rm.caPath, pair.CertPEM, 0644); err != nil {
			return fmt.Errorf("write ca: %w", err)
		}
	}
	return rm.buildTLSConfig()
}

// buildTLSConfig 根据当前 pair 重建 tls.Config
func (rm *RotationManager) buildTLSConfig() error {
	rm.mu.RLock()
	pair := rm.pair
	caPEM := rm.caPEM
	rm.mu.RUnlock()

	if pair == nil {
		return fmt.Errorf("no cert pair available")
	}
	if len(caPEM) == 0 {
		caPEM = pair.CertPEM
	}
	cfg, err := LoadTLSConfig(pair.CertPEM, pair.KeyPEM, caPEM)
	if err != nil {
		return fmt.Errorf("load tls config: %w", err)
	}
	rm.mu.Lock()
	rm.tlsConfig = cfg
	rm.mu.Unlock()
	return nil
}

// GetTLSConfig 返回当前 tls.Config（线程安全）
func (rm *RotationManager) GetTLSConfig() *tls.Config {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	if rm.tlsConfig == nil {
		return nil
	}
	// 返回拷贝防止外部修改
	cfg := rm.tlsConfig.Clone()
	return cfg
}

// Start 启动后台轮换监控（建议 1h 检查一次）
func (rm *RotationManager) Start(checkInterval time.Duration) {
	if checkInterval <= 0 {
		checkInterval = 1 * time.Hour
	}
	ticker := time.NewTicker(checkInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rm.mu.RLock()
				next := rm.nextRotate
				rm.mu.RUnlock()
				if time.Until(next) < rm.window {
					if err := rm.RotateNow(); err != nil {
						// 轮换失败记录到 stderr，不中断服务
						fmt.Fprintf(os.Stderr, "[mtls-rotation] cert rotation failed: %v\n", err)
					} else {
						fmt.Fprintf(os.Stderr, "[mtls-rotation] cert rotated successfully, next at %s\n", rm.nextRotate.Format(time.RFC3339))
					}
				}
			case <-rm.stopCh:
				return
			}
		}
	}()
}

// Stop 停止后台轮换监控
func (rm *RotationManager) Stop() {
	close(rm.stopCh)
}

// Expiry 返回当前证书预计过期时间
func (rm *RotationManager) Expiry() time.Time {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.nextRotate
}

// IsExpiringSoon 检查证书是否即将过期（进入轮换窗口）
func (rm *RotationManager) IsExpiringSoon() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return time.Until(rm.nextRotate) < rm.window
}

// ParseCertificate 解析当前证书获取 x509 信息（用于调试/监控）
func (rm *RotationManager) ParseCertificate() (*x509.Certificate, error) {
	rm.mu.RLock()
	pair := rm.pair
	rm.mu.RUnlock()
	if pair == nil || len(pair.CertPEM) == 0 {
		return nil, fmt.Errorf("no cert available")
	}
	block, _ := pem.Decode(pair.CertPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode cert PEM")
	}
	return x509.ParseCertificate(block.Bytes)
}
