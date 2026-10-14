// Package middleware HTTP 中间件
//
// 包含：请求ID、结构化日志、Panic恢复、限流、JWT认证、RBAC
package middleware

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/time/rate"
)

// ─── 请求ID ────────────────────────────────────────────

type requestIDKey struct{}

// RequestID 注入唯一请求ID
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := fmt.Sprintf("req-%d", time.Now().UnixNano())
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID 获取请求ID
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// ─── 结构化日志 ────────────────────────────────────────

// Logging 结构化请求日志
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// 包装 ResponseWriter 捕获状态码
			wrapped := &responseWriter{ResponseWriter: w, status: 200}
			
			next.ServeHTTP(wrapped, r)
			
			logger.InfoContext(r.Context(), "http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"request_id", GetRequestID(r.Context()),
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// ─── Panic 恢复 ────────────────────────────────────────

// Recover Panic 恢复中间件
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.ErrorContext(r.Context(), "panic_recovered",
						"error", err,
						"request_id", GetRequestID(r.Context()),
						"path", r.URL.Path,
					)
					http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// ─── 限流 ──────────────────────────────────────────────

// RateLimiter 令牌桶限流器
type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mu       sync.Mutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter 创建限流器
// r: 每秒令牌数, burst: 突发容量
func NewRateLimiter(r float64, burst int) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		rate:     rate.Limit(r),
		burst:    burst,
	}
}

// Limit 限流中间件
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		limiter := rl.getLimiter(ip)
		
		if !limiter.Allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	if limiter, exists := rl.visitors[ip]; exists {
		return limiter
	}
	
	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.visitors[ip] = limiter
	return limiter
}

// CleanupStale 定期清理空闲访客（需 goroutine 调用）
func (rl *RateLimiter) CleanupStale(idle time.Duration) {
	ticker := time.NewTicker(idle)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		// 生产版：记录 lastSeen 并清理
		rl.mu.Unlock()
	}
}

func extractIP(r *http.Request) string {
	// 优先从 X-Forwarded-For 获取（反代场景）
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	
	// X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// RemoteAddr
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		return host[:idx]
	}
	return host
}

// ─── JWT 认证 ──────────────────────────────────────────

// Claims JWT 声明
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// JWTAuth JWT 认证中间件
type JWTAuth struct {
	publicKey  *rsa.PublicKey
	signingKey []byte // HMAC
	method     jwt.SigningMethod
}

// NewJWTAuthHMAC 创建 HMAC JWT 认证器
func NewJWTAuthHMAC(secret []byte) *JWTAuth {
	return &JWTAuth{
		signingKey: secret,
		method:     jwt.SigningMethodHS256,
	}
}

// AuthRequired JWT 认证中间件
func (j *JWTAuth) AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, `{"error":"missing authorization token"}`, http.StatusUnauthorized)
			return
		}
		
		claims := &Claims{}
		parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return j.signingKey, nil
		})
		
		if err != nil || !parsed.Valid {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		
		// 检查过期
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			http.Error(w, `{"error":"token expired"}`, http.StatusUnauthorized)
			return
		}
		
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GenerateToken 生成 JWT Token
func (j *JWTAuth) GenerateToken(userID, username string, roles []string, expiry time.Duration) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "lexscale",
		},
	}
	
	token := jwt.NewWithClaims(j.method, claims)
	return token.SignedString(j.signingKey)
}

// GetClaims 从 Context 获取 Claims
func GetClaims(ctx context.Context) *Claims {
	if claims, ok := ctx.Value("claims").(*Claims); ok {
		return claims
	}
	return nil
}

func extractToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	
	// Bearer token
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	
	return ""
}

// ─── RBAC 授权 ─────────────────────────────────────────

// Role 角色定义
type Role string

const (
	RoleAdmin     Role = "admin"     // 管理员：全部权限
	RolePartner   Role = "partner"   // 合伙人：审批 + 查看全部案件
	RoleAssociate Role = "associate" // 律师：创建案件 + 执行分析
	RoleViewer    Role = "viewer"    // 只读：查看案件和报告
)

// RequireRole 要求指定角色
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			
			hasRole := false
			for _, required := range roles {
				for _, userRole := range claims.Roles {
					if Role(userRole) == required || Role(userRole) == RoleAdmin {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}
			
			if !hasRole {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}
