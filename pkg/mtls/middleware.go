package mtls

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
)

// contextKey 用于在请求上下文中存储客户端证书信息
type contextKey int

const (
	clientCertKey contextKey = iota
	clientIdentityKey
)

// ClientIdentity 从客户端证书中提取的身份信息
type ClientIdentity struct {
	Subject     string   `json:"subject"`
	Issuer      string   `json:"issuer"`
	Serial      string   `json:"serial"`
	DNSNames    []string `json:"dns_names"`
	Fingerprint string   `json:"fingerprint"`
}

// ClientCertMiddleware 从 TLS 连接中提取客户端证书并注入上下文
// 与 CertificateManager 配合使用，为零信任网络接入提供身份识别
func ClientCertMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			// 非 mTLS 连接或客户端未提供证书，直接放行（由 tls.Config.ClientAuth 控制是否拒绝）
			next(w, r)
			return
		}

		cert := r.TLS.PeerCertificates[0]
		identity := extractIdentity(cert)

		ctx := context.WithValue(r.Context(), clientCertKey, cert)
		ctx = context.WithValue(ctx, clientIdentityKey, identity)

		next(w, r.WithContext(ctx))
	}
}

// RequireClientCert 强制要求客户端证书中间件
// 如果请求未携带有效客户端证书，返回 495（SSL Certificate Required）
func RequireClientCert(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, FriendlyError(ErrClientCertRequired), 495)
			return
		}
		next(w, r)
	}
}

// ClientCertFromContext 从上下文中获取客户端证书
func ClientCertFromContext(ctx context.Context) (*x509.Certificate, bool) {
	cert, ok := ctx.Value(clientCertKey).(*x509.Certificate)
	return cert, ok
}

// ClientIdentityFromContext 从上下文中获取客户端身份信息
func ClientIdentityFromContext(ctx context.Context) (ClientIdentity, bool) {
	id, ok := ctx.Value(clientIdentityKey).(ClientIdentity)
	return id, ok
}

func extractIdentity(cert *x509.Certificate) ClientIdentity {
	fp := ""
	if len(cert.Raw) > 0 {
		// 简单的指纹前缀（完整指纹在需要时计算）
		sum := 0
		for _, b := range cert.Raw {
			sum += int(b)
		}
		fp = fmt.Sprintf("%x", sum)
	}
	return ClientIdentity{
		Subject:     cert.Subject.String(),
		Issuer:      cert.Issuer.String(),
		Serial:      cert.SerialNumber.String(),
		DNSNames:    cert.DNSNames,
		Fingerprint: fp,
	}
}
