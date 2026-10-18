package httpapi

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"remoon.net/yu/backend/internal/auth"
	"remoon.net/yu/backend/internal/store"
)

const (
	sessionCookie = "yu_session"
	csrfCookie    = "yu_csrf"
	authViaCookie = "cookie"
)

type ctxKey int

const (
	userKey ctxKey = iota
	authMethodKey
)

func userFromContext(ctx context.Context) *store.User {
	u, _ := ctx.Value(userKey).(*store.User)
	return u
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", r.RemoteAddr,
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && s.originAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(origin string) bool {
	publicOrigin, err := url.Parse(s.cfg.PublicBaseURL)
	if err == nil {
		if origin == publicOrigin.Scheme+"://"+publicOrigin.Host {
			return true
		}
		if parsed, err := url.Parse(origin); err == nil &&
			parsed.Scheme == publicOrigin.Scheme && parsed.Host == publicOrigin.Host {
			return true
		}
	}

	for _, allowed := range s.cfg.AllowedOrigins {
		if allowed == origin {
			return true
		}
	}

	return false
}

// originGuard 拒绝来自非白名单来源的写请求，缓解 CSRF。
func (s *Server) originGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		// 采集接口面向任意站点，来源校验交给站点 token 与限流。
		if r.URL.Path == "/api/collect" {
			next.ServeHTTP(w, r)
			return
		}

		if origin := r.Header.Get("Origin"); origin != "" && !s.originAllowed(origin) {
			writeError(w, http.StatusForbidden, "请求来源不被允许")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// authRequired 从会话 Cookie 中还原当前用户。
func (s *Server) authRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
			if user, err := s.store.GetUserBySession(ctx, auth.HashToken(c.Value)); err == nil {
				ctx = context.WithValue(ctx, userKey, user)
				ctx = context.WithValue(ctx, authMethodKey, authViaCookie)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
			token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
			if token != "" {
				if user, err := s.store.GetUserByAPIKey(ctx, auth.HashToken(token)); err == nil {
					ctx = context.WithValue(ctx, userKey, user)
					ctx = context.WithValue(ctx, authMethodKey, "api_key")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		writeError(w, http.StatusUnauthorized, "未登录或会话已过期")
	})
}

// csrfProtect 对基于 Cookie 的写请求校验双提交 token。
func (s *Server) csrfProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) {
			next.ServeHTTP(w, r)
			return
		}

		if method, _ := r.Context().Value(authMethodKey).(string); method != authViaCookie {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(csrfCookie)
		header := r.Header.Get("X-CSRF-Token")

		if err != nil || cookie.Value == "" ||
			subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) != 1 {
			writeError(w, http.StatusForbidden, "CSRF 校验失败")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) setCSRFCookie(w http.ResponseWriter) {
	token, err := auth.NewToken()
	if err != nil {
		slog.Error("生成 CSRF token 失败", "err", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int((30 * 24 * time.Hour).Seconds()),
		HttpOnly: false,
		Secure:   s.cfg.SecureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) clearCookies(w http.ResponseWriter) {
	for _, name := range []string{sessionCookie, csrfCookie} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: name == sessionCookie,
			Secure:   s.cfg.SecureCookies,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func trimSpace(s string) string {
	return strings.TrimSpace(s)
}
