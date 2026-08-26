// Package handler 提供HTTP处理层
package handler

import (
	"net/http"
	"time"

	"logalert/pkg/logger"
)

// Middleware HTTP中间件
type Middleware struct {
	logger *logger.Logger
}

// NewMiddleware 创建中间件
func NewMiddleware(log *logger.Logger) *Middleware {
	return &Middleware{logger: log}
}

// RequestLogger 请求日志中间件
func (m *Middleware) RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start)
		m.logger.Infof("request: method=%s, path=%s, status=%d, duration=%v, remote=%s",
			r.Method, r.URL.Path, wrapped.statusCode, duration, r.RemoteAddr)
	})
}

// CORS CORS跨域中间件
func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-Trace-ID")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Recover 恢复中间件
func (m *Middleware) Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Errorf("panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestID 请求ID中间件
func (m *Middleware) RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

// ContentTypeChecker 检查Content-Type中间件
func (m *Middleware) ContentTypeChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" || r.Method == "PUT" {
			contentType := r.Header.Get("Content-Type")
			if contentType != "" && len(contentType) >= 16 {
				if contentType[:16] != "application/json" {
					http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
					return
				}
			} else if contentType != "" && contentType != "application/json" {
				http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// responseWriter 包装ResponseWriter
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// WriteHeader 重写WriteHeader
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.written = true
	rw.ResponseWriter.WriteHeader(code)
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	now := time.Now().UnixNano()
	for i := range b {
		b[i] = charset[now%int64(len(charset))]
		now = now / int64(len(charset))
	}
	return string(b)
}
