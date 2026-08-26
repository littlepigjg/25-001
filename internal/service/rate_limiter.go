// Package service 提供业务逻辑层
package service

import (
	"context"
	"logalert/internal/config"
	"logalert/pkg/logger"
	"sync"
	"time"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string]*requestWindow
	cfg      *config.Config
	logger   *logger.Logger
}

// requestWindow 请求窗口
type requestWindow struct {
	count    int
	window   time.Time
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(cfg *config.Config, log *logger.Logger) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string]*requestWindow),
		cfg:      cfg,
		logger:   log,
	}
}

// Allow 检查请求是否被允许
func (rl *RateLimiter) Allow(source string, maxRequests int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	window, exists := rl.requests[source]

	if !exists || now.Sub(window.window) > time.Minute {
		rl.requests[source] = &requestWindow{
			count:  1,
			window: now,
		}
		return true
	}

	window.count++
	if window.count > maxRequests {
		return false
	}
	return true
}

// Cleanup 清理过期的速率限制记录
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for source, window := range rl.requests {
		if now.Sub(window.window) > time.Minute {
			delete(rl.requests, source)
		}
	}
}

// StartCleanupLoop 启动定期清理
func (rl *RateLimiter) StartCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.Cleanup()
		case <-ctx.Done():
			return
		}
	}
}
