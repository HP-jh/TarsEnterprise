package security

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ── Rate Limiter ──

type bucket struct {
	tokens    float64
	lastRefill time.Time
	mu        sync.Mutex
}

func (b *bucket) take() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = min(b.tokens+elapsed*2, 10) // 2 tokens/sec, max 10
	b.lastRefill = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

var rateBuckets = struct {
	mu sync.RWMutex
	m  map[string]*bucket
}{m: make(map[string]*bucket)}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			xff = strings.TrimSpace(xff[:i])
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host != "" {
		return host
	}
	return r.RemoteAddr
}

// RateLimit 令牌桶限速中间件（每个 IP 10 burst / 2 QPS）
func RateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		rateBuckets.mu.RLock()
		b, ok := rateBuckets.m[ip]
		rateBuckets.mu.RUnlock()
		if !ok {
			rateBuckets.mu.Lock()
			b = &bucket{tokens: 10, lastRefill: time.Now()}
			rateBuckets.m[ip] = b
			rateBuckets.mu.Unlock()
		}
		if !b.take() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "请求太快了，请稍后再试", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// ── CORS ──

// CORS 跨域中间件（默认允许 localhost 开发环境，生产环境应收紧）
func CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Token, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// ── Security Headers ──

// SecureHeaders 注入基础安全响应头
func SecureHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next(w, r)
	}
}

// ── Max Body Size ──

// MaxBody 限制请求体大小（默认 8 MiB）
func MaxBody(limit int64) func(http.HandlerFunc) http.HandlerFunc {
	if limit <= 0 {
		limit = 8 << 20 // 8 MiB
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next(w, r)
		}
	}
}

// ── Admin Auth ──

// AdminAuth 管理员 Token 校验中间件
func AdminAuth(token string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				http.Error(w, "管理员令牌尚未配置，请先设置 admin_token", http.StatusForbidden)
				return
			}
			if r.Header.Get("X-Admin-Token") != token {
				http.Error(w, "管理员令牌不正确，请检查后再试", http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
}

// ── Chain ──

// Chain 顺序应用多个中间件
func Chain(h http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// FriendlyMethodError 返回人话化的 HTTP 方法错误
func FriendlyMethodError(w http.ResponseWriter, method string) {
	msg := fmt.Sprintf("这个接口需要用 %s 方法访问，当前方法不对，请换个姿势重试", method)
	http.Error(w, msg, http.StatusMethodNotAllowed)
}
