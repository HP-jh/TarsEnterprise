package mtls

import (
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
)

// ZeroTrustConfig 零信任验证配置
type ZeroTrustConfig struct {
	AllowRoles []string // 允许的证书角色（匹配 CN 或 OU）
	Enforce    bool     // 是否强制拒绝无有效客户端证书的连接
}

// VerifyPeerCertificate 返回 tls.Config 可用的 VerifyPeerCertificate 回调
// 在标准 TLS 握手验证之后执行零信任身份校验
func (z *ZeroTrustConfig) VerifyPeerCertificate(allowRoles []string) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		if len(verifiedChains) == 0 || len(verifiedChains[0]) == 0 {
			if z.Enforce {
				return fmt.Errorf("零信任策略：未提供有效客户端证书，连接被拒绝")
			}
			return nil
		}
		cert := verifiedChains[0][0]
		// 提取身份信息
		identity := extractIdentity(cert)
		// 检查角色白名单
		if len(allowRoles) > 0 {
			allowed := false
			for _, role := range allowRoles {
				for _, id := range identity {
					if strings.EqualFold(role, id) {
						allowed = true
						break
					}
				}
				if allowed {
					break
				}
			}
			if !allowed {
				return fmt.Errorf("零信任策略：客户端身份 '%v' 不在允许列表中，连接被拒绝", identity)
			}
		}
		return nil
	}
}

// extractIdentity 从证书中提取身份标识（CN + OU）
func extractIdentity(cert *x509.Certificate) []string {
	ids := make([]string, 0, 4)
	if cert.Subject.CommonName != "" {
		ids = append(ids, cert.Subject.CommonName)
	}
	for _, ou := range cert.Subject.OrganizationalUnit {
		if ou != "" {
			ids = append(ids, ou)
		}
	}
	for _, o := range cert.Subject.Organization {
		if o != "" {
			ids = append(ids, o)
		}
	}
	return ids
}

// ClientIdentityFromRequest 从 HTTP 请求中提取 mTLS 客户端身份信息
// 返回空字符串表示未提供客户端证书
func ClientIdentityFromRequest(r *http.Request) string {
	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 {
		return ""
	}
	cert := r.TLS.VerifiedChains[0][0]
	if cert.Subject.CommonName != "" {
		return cert.Subject.CommonName
	}
	if len(cert.Subject.OrganizationalUnit) > 0 {
		return cert.Subject.OrganizationalUnit[0]
	}
	return ""
}

// RequireMTLSMiddleware HTTP 中间件：强制要求 mTLS 并提供人话化错误提示
func RequireMTLSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 {
			http.Error(w, "当前连接未启用双向认证（mTLS），请配置客户端证书后再试", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
