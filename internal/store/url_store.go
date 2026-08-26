package store

import (
	"context"
	"logalert/internal/model"
	"logalert/pkg/errors"
	"sync"
	"time"
)

// PanicGuardFn 故障演练守卫函数
type PanicGuardFn func(code, rawURL string) bool

// URLStore 短链存储接口
type URLStore struct {
	mu      sync.RWMutex
	urls    map[string]*model.ShortURL
	panicFn PanicGuardFn
}

// NewURLStore 创建URL存储
func NewURLStore(cfg interface{}) (*URLStore, error) {
	return &URLStore{
		urls: make(map[string]*model.ShortURL),
	}, nil
}

// Load 从持久化存储加载数据
func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return nil
}

// Close 关闭存储
func (s *URLStore) Close() error {
	return nil
}

// SetPanicGuard 设置故障演练守卫
func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicFn = fn
}

// Save 保存短链
func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return errors.New(40000, "short URL cannot be nil")
	}
	if u.Code == "" {
		return errors.New(40000, "code cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.panicFn != nil && s.panicFn(u.Code, u.RawURL) {
		panic("panic guard triggered for code: " + u.Code)
	}

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return errors.New(40900, "code already exists: "+u.Code)
		}
	}

	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}

	s.urls[u.Code] = u
	return nil
}

// Get 根据code获取短链
func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.urls[code]
	if !ok {
		return nil, errors.New(40400, "short URL not found: "+code)
	}
	return u, nil
}

// RawSnapshot 获取原始快照
func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]model.ShortURL, len(s.urls))
	for code, u := range s.urls {
		snapshot[code] = *u
	}
	return snapshot
}

// AccessLogStore 访问日志存储
type AccessLogStore struct {
	mu     sync.RWMutex
	logs   []AccessLogEntry
}

// AccessLogEntry 访问日志条目
type AccessLogEntry struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	Timestamp time.Time `json:"timestamp"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Status    int       `json:"status"`
}

// NewAccessLogStore 创建访问日志存储
func NewAccessLogStore(cfg interface{}) (*AccessLogStore, error) {
	return &AccessLogStore{
		logs: make([]AccessLogEntry, 0),
	}, nil
}

// Open 打开访问日志存储
func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return nil
}

// Close 关闭访问日志存储
func (s *AccessLogStore) Close() error {
	return nil
}

// Append 添加访问日志
func (s *AccessLogStore) Append(entry AccessLogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, entry)
}

// GetLogs 获取所有访问日志
func (s *AccessLogStore) GetLogs() []AccessLogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]AccessLogEntry, len(s.logs))
	copy(result, s.logs)
	return result
}
