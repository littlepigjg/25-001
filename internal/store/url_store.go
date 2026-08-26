package store

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	pkgerrors "logalert/pkg/errors"
	"logalert/pkg/logger"
	"sync"
	"time"
)

// PanicGuardFn 紧急保护函数类型
type PanicGuardFn func(code, rawURL string) bool

// URLStore URL存储
type URLStore struct {
	mu      sync.RWMutex
	cfg     *config.Config
	logger  *logger.Logger
	urls    map[string]model.ShortURL
	guardFn PanicGuardFn
}

// NewURLStore 创建URL存储
func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, pkgerrors.New(40000, "config cannot be nil")
	}
	return &URLStore{
		cfg:    cfg,
		logger: logger.New(),
		urls:   make(map[string]model.ShortURL),
	}, nil
}

// Load 加载URL数据
func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger.Info("URL store loaded successfully")
	return nil
}

// Close 关闭存储
func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls = make(map[string]model.ShortURL)
	s.logger.Info("URL store closed")
	return nil
}

// SetPanicGuard 设置紧急保护函数
func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.guardFn = fn
}

// Save 保存短链
func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return pkgerrors.New(40000, "short url cannot be nil")
	}
	if err := u.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return model.NewConflictError("short_url", "code already exists")
		}
	}
	s.urls[u.Code] = *u
	return nil
}

// Get 获取短链
func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	if code == "" {
		return nil, pkgerrors.New(40000, "code cannot be empty")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.urls[code]
	if !ok {
		return nil, model.NewNotFoundError("url", code)
	}
	return &u, nil
}

// RawSnapshot 获取原始快照（诊断用）
func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make(map[string]model.ShortURL, len(s.urls))
	for k, v := range s.urls {
		snapshot[k] = v
	}
	return snapshot
}

// SaveWithGuard 带安全保护的保存（故障演练钩子）
func (s *URLStore) SaveWithGuard(u *model.ShortURL, overwrite bool) error {
	if s.guardFn != nil && !s.guardFn(u.Code, u.RawURL) {
		return pkgerrors.New(50000, "panic guard rejected save operation")
	}
	return s.Save(u, overwrite)
}

// GetWithGuard 带安全保护的获取（故障演练钩子）
func (s *URLStore) GetWithGuard(code string) (*model.ShortURL, error) {
	if s.guardFn != nil && !s.guardFn(code, "") {
		return nil, pkgerrors.New(50000, "panic guard rejected get operation")
	}
	return s.Get(code)
}

// IncrementVisitsWithGuard 带安全保护的访问计数增加（故障演练钩子）
func (s *URLStore) IncrementVisitsWithGuard(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.urls[code]
	if !ok {
		return model.NewNotFoundError("url", code)
	}
	if s.guardFn != nil && !s.guardFn(code, u.RawURL) {
		return pkgerrors.New(50000, "panic guard rejected increment operation")
	}
	u.Visits++
	s.urls[code] = u
	return nil
}

// AccessLogStore 访问日志存储
type AccessLogStore struct {
	mu     sync.RWMutex
	cfg    *config.Config
	logger *logger.Logger
	logs   []AccessLogEntry
}

// AccessLogEntry 访问日志条目
type AccessLogEntry struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	Timestamp time.Time `json:"timestamp"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
}

// NewAccessLogStore 创建访问日志存储
func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, pkgerrors.New(40000, "config cannot be nil")
	}
	return &AccessLogStore{
		cfg:    cfg,
		logger: logger.New(),
		logs:   make([]AccessLogEntry, 0),
	}, nil
}

// Open 打开存储
func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger.Info("Access log store opened")
	return nil
}

// Close 关闭存储
func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = make([]AccessLogEntry, 0)
	s.logger.Info("Access log store closed")
	return nil
}

// Append 追加访问日志
func (s *AccessLogStore) Append(entry AccessLogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, entry)
}
