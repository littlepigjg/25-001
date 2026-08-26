// Package store 提供数据存储层
package store

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"sync"
)

// PanicGuardFn panic保护函数类型
type PanicGuardFn func(code, rawURL string) bool

// URLStore URL存储
type URLStore struct {
	mu         sync.RWMutex
	urls       map[string]*model.ShortURL
	panicGuard PanicGuardFn
	loaded     bool
	closed     bool
}

// NewURLStore 创建URL存储
func NewURLStore(cfg *config.Config) (*URLStore, error) {
	return &URLStore{
		urls: make(map[string]*model.ShortURL),
	}, nil
}

// Load 从持久化存储加载
func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loaded = true
	return nil
}

// Close 关闭存储
func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// SetPanicGuard 设置panic保护
func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

// Save 保存短链
func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return model.NewValidationError("url", "url cannot be nil")
	}
	if u.Code == "" {
		return model.NewValidationError("code", "code cannot be empty")
	}
	if u.RawURL == "" {
		return model.NewValidationError("raw_url", "raw_url cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return model.NewConflictError("short_url", "code already exists")
		}
	}

	s.urls[u.Code] = u
	return nil
}

// Get 获取短链
func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	if code == "" {
		return nil, model.NewValidationError("code", "code cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.urls[code]
	if !ok {
		return nil, model.NewNotFoundError("short_url", code)
	}
	return url, nil
}

// RawSnapshot 获取原始快照
func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]model.ShortURL, len(s.urls))
	for k, v := range s.urls {
		snapshot[k] = *v
	}
	return snapshot
}
