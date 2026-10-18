package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"
)

// CertificateManager 管理服务端证书、CA 证书池与客户端证书验证
// 为零信任网络接入提供证书全生命周期管理
type CertificateManager struct {
	mu sync.RWMutex

	cert     tls.Certificate
	caPool   *x509.CertPool
	clientAuth tls.ClientAuthType

	certFile string
	keyFile  string
	caFile   string

	loadedAt time.Time
}

// NewManager 从文件路径创建证书管理器
// 支持自签名 CA 和自定义 CA 配置
func NewManager(certFile, keyFile, caFile string) (*CertificateManager, error) {
	cm := &CertificateManager{
		certFile:   certFile,
		keyFile:    keyFile,
		caFile:     caFile,
		clientAuth: tls.RequireAndVerifyClientCert,
	}
	if err := cm.Reload(); err != nil {
		return nil, err
	}
	return cm, nil
}

// NewManagerFromPEM 从内存 PEM 创建证书管理器
func NewManagerFromPEM(certPEM, keyPEM, caPEM []byte) (*CertificateManager, error) {
	cm := &CertificateManager{
		clientAuth: tls.RequireAndVerifyClientCert,
	}
	if err := cm.loadFromPEM(certPEM, keyPEM, caPEM); err != nil {
		return nil, err
	}
	return cm, nil
}

// Reload 从磁盘重新加载证书
func (cm *CertificateManager) Reload() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	certPEM, err := os.ReadFile(cm.certFile)
	if err != nil {
		return fmt.Errorf("读取服务端证书失败：%w", err)
	}
	keyPEM, err := os.ReadFile(cm.keyFile)
	if err != nil {
		return fmt.Errorf("读取服务端密钥失败：%w", err)
	}
	caPEM, err := os.ReadFile(cm.caFile)
	if err != nil {
		return fmt.Errorf("读取 CA 证书失败：%w", err)
	}

	return cm.loadFromPEM(certPEM, keyPEM, caPEM)
}

func (cm *CertificateManager) loadFromPEM(certPEM, keyPEM, caPEM []byte) error {
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("解析证书密钥对失败：%w", err)
	}

	// 解析 leaf 证书以便后续检查过期
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("解析证书内容失败：%w", err)
	}
	cert.Leaf = leaf

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return fmt.Errorf("解析 CA 证书失败：无有效证书")
	}

	cm.cert = cert
	cm.caPool = pool
	cm.loadedAt = time.Now()
	return nil
}

// TLSConfig 生成用于 http.Server 的 tls.Config
func (cm *CertificateManager) TLSConfig() *tls.Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return &tls.Config{
		Certificates: []tls.Certificate{cm.cert},
		ClientCAs:    cm.caPool,
		ClientAuth:   cm.clientAuth,
		MinVersion:   tls.VersionTLS13,
		// 自定义 VerifyConnection 用于过期拒绝 + 友好错误
		VerifyConnection: cm.verifyConnection,
	}
}

// verifyConnection 在 TLS 握手完成后验证连接
// 证书过期自动拒绝 + 记录审计信息
func (cm *CertificateManager) verifyConnection(cs tls.ConnectionState) error {
	// 检查服务端证书是否过期
	if cm.cert.Leaf != nil && time.Now().After(cm.cert.Leaf.NotAfter) {
		return ErrServerCertExpired
	}

	// 检查客户端证书
	if len(cs.PeerCertificates) == 0 {
		return ErrClientCertRequired
	}

	clientCert := cs.PeerCertificates[0]
	now := time.Now()
	if now.Before(clientCert.NotBefore) {
		return ErrClientCertNotYetValid
	}
	if now.After(clientCert.NotAfter) {
		return ErrClientCertExpired
	}

	return nil
}

// ServerCertInfo 返回服务端证书信息（用于健康检查/状态展示）
func (cm *CertificateManager) ServerCertInfo() CertInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.cert.Leaf == nil {
		return CertInfo{}
	}
	return CertInfo{
		Subject:   cm.cert.Leaf.Subject.String(),
		Issuer:    cm.cert.Leaf.Issuer.String(),
		NotBefore: cm.cert.Leaf.NotBefore,
		NotAfter:  cm.cert.Leaf.NotAfter,
		LoadedAt:  cm.loadedAt,
	}
}

// ValidateClientCert 单独验证一份客户端证书（用于网关外的证书校验场景）
func (cm *CertificateManager) ValidateClientCert(cert *x509.Certificate) error {
	cm.mu.RLock()
	pool := cm.caPool
	cm.mu.RUnlock()

	if pool == nil {
		return fmt.Errorf("CA 证书池尚未初始化")
	}

	opts := x509.VerifyOptions{
		Roots:         pool,
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("客户端证书验证失败：%w", err)
	}
	return nil
}

// CertInfo 证书摘要信息
type CertInfo struct {
	Subject   string    `json:"subject"`
	Issuer    string    `json:"issuer"`
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
	LoadedAt  time.Time `json:"loaded_at"`
}

// DaysUntilExpiry 距离过期还有多少天
func (ci CertInfo) DaysUntilExpiry() int {
	return int(ci.NotAfter.Sub(time.Now()).Hours() / 24)
}

// IsExpired 证书是否已过期
func (ci CertInfo) IsExpired() bool {
	return time.Now().After(ci.NotAfter)
}
