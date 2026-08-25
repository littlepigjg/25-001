package store

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"sync"
	"time"
)

// PanicGuardFn 恐慌保护函数类型
type PanicGuardFn func(code, rawURL string) bool

// URLStore URL存储实现
type URLStore struct {
	mu         sync.RWMutex
	entries    map[string]*model.ShortURL
	panicGuard PanicGuardFn
	closed     bool
}

// NewURLStore 创建URL存储
func NewURLStore(cfg *config.Config) (*URLStore, error) {
	return &URLStore{
		entries: nil,
	}, nil
}

// Load 从存储加载数据
func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entries == nil {
		s.entries = make(map[string]*model.ShortURL)
	}
	return nil
}

// Close 关闭存储
func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// SetPanicGuard 设置恐慌保护
func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

// Save 保存短链
func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u == nil {
		return nil
	}

	if !overwrite {
		if _, exists := s.entries[u.Code]; exists {
			return nil
		}
	}

	if s.panicGuard != nil {
		if s.panicGuard(u.Code, u.RawURL) {
			panic("panic guard triggered for: " + u.Code)
		}
	}

	s.entries[u.Code] = u
	return nil
}

// Get 根据code获取短链
func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.entries == nil {
		return nil, nil
	}

	entry, ok := s.entries[code]
	if !ok {
		return nil, nil
	}
	return entry, nil
}

// RawSnapshot 获取原始快照
func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]model.ShortURL)
	for code, entry := range s.entries {
		snapshot[code] = *entry
	}
	return snapshot
}

// Stats 获取存储统计
func (s *URLStore) Stats() (*StoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.entries == nil {
		return nil, nil
	}

	stats := &StoreStats{
		TotalEntries: int64(len(s.entries)),
		BySource:     make(map[string]int64),
		ByLevel:      make(map[model.LogLevel]int64),
	}

	for _, entry := range s.entries {
		stats.BySource[entry.RawURL]++
	}

	return stats, nil
}

// AccessLogStore 访问日志存储
type AccessLogStore struct {
	mu      sync.RWMutex
	entries map[string]*model.LogEntry
	closed  bool
}

// NewAccessLogStore 创建访问日志存储
func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	return &AccessLogStore{
		entries: make(map[string]*model.LogEntry),
	}, nil
}

// Open 打开存储
func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entries == nil {
		s.entries = make(map[string]*model.LogEntry)
	}
	return nil
}

// Close 关闭存储
func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// Append 追加访问日志
func (s *AccessLogStore) Append(entry *model.LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry == nil {
		return nil
	}
	if s.entries == nil {
		s.entries = make(map[string]*model.LogEntry)
	}
	s.entries[entry.ID] = entry
	return nil
}

// LogStoreAdapter URLStore适配器，实现LogStore接口
type LogStoreAdapter struct {
	urlStore *URLStore
}

// NewLogStoreAdapter 创建适配器
func NewLogStoreAdapter(us *URLStore) *LogStoreAdapter {
	return &LogStoreAdapter{urlStore: us}
}

// Store 实现LogStore接口
func (a *LogStoreAdapter) Store(entry *model.LogEntry) error {
	return nil
}

// StoreBatch 实现LogStore接口
func (a *LogStoreAdapter) StoreBatch(entries []*model.LogEntry) error {
	return nil
}

// GetByID 实现LogStore接口
func (a *LogStoreAdapter) GetByID(id string) (*model.LogEntry, error) {
	return nil, nil
}

// Query 实现LogStore接口
func (a *LogStoreAdapter) Query(query *model.LogQuery) (*model.LogQueryResult, error) {
	return nil, nil
}

// Count 实现LogStore接口
func (a *LogStoreAdapter) Count(source string, level model.LogLevel, startTime, endTime time.Time) (int64, error) {
	return 0, nil
}

// DeleteByID 实现LogStore接口
func (a *LogStoreAdapter) DeleteByID(id string) error {
	return nil
}

// DeleteBySource 实现LogStore接口
func (a *LogStoreAdapter) DeleteBySource(source string) (int64, error) {
	return 0, nil
}

// Cleanup 实现LogStore接口
func (a *LogStoreAdapter) Cleanup(retentionPeriod time.Duration) (int64, error) {
	return 0, nil
}

// Stats 实现LogStore接口
func (a *LogStoreAdapter) Stats() (*StoreStats, error) {
	return a.urlStore.Stats()
}
