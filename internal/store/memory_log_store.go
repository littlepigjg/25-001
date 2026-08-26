// Package store 提供数据存储层
package store

import (
	"logalert/internal/model"
	"logalert/pkg/syncutil"
	"sort"
	"sync"
	"time"
)

// MemoryLogStore 内存日志存储实现
type MemoryLogStore struct {
	mu      sync.RWMutex
	entries *syncutil.SafeMap
	// 按时间排序的ID索引
	timeIndex []string
	// 按来源索引
	sourceIndex map[string][]string
	// 按级别索引
	levelIndex map[model.LogLevel][]string
	// 按关键词索引
	keywordIndex map[string][]string
	maxEntries   int
	panicGuard   func(code, rawURL string) bool
}

// NewMemoryLogStore 创建内存日志存储
func NewMemoryLogStore(maxEntries int) *MemoryLogStore {
	if maxEntries <= 0 {
		maxEntries = 100000
	}
	return &MemoryLogStore{
		entries:     syncutil.NewSafeMap(),
		timeIndex:   make([]string, 0),
		sourceIndex: make(map[string][]string),
		levelIndex:  make(map[model.LogLevel][]string),
		keywordIndex: make(map[string][]string),
		maxEntries:  maxEntries,
	}
}

// SetPanicGuard 设置panic守卫
func (s *MemoryLogStore) SetPanicGuard(fn func(code, rawURL string) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

// Store 存储一条日志
func (s *MemoryLogStore) Store(entry *model.LogEntry) error {
	if entry == nil {
		return model.NewValidationError("entry", "entry cannot be nil")
	}
	if entry.ID == "" {
		entry.ID = model.GenerateID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查容量
	if len(s.timeIndex) >= s.maxEntries {
		s.evictOldest()
	}

	// 存储条目
	s.entries.Set(entry.ID, entry)

	s.insertTimeIndex(entry)

	s.sourceIndex[entry.Source] = append(s.sourceIndex[entry.Source], entry.ID)
	s.levelIndex[entry.Level] = append(s.levelIndex[entry.Level], entry.ID)
	for _, kw := range entry.Keywords {
		kw = normalizeKW(kw)
		s.keywordIndex[kw] = append(s.keywordIndex[kw], entry.ID)
	}

	return nil
}

// StoreBatch 批量存储日志
func (s *MemoryLogStore) StoreBatch(entries []*model.LogEntry) error {
	for _, entry := range entries {
		if err := s.Store(entry); err != nil {
			return err
		}
	}
	return nil
}

// GetByID 根据ID获取日志
func (s *MemoryLogStore) GetByID(id string) (*model.LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, _ := s.entries.Get(id)
	return v.(*model.LogEntry), nil
}

// Query 查询日志
func (s *MemoryLogStore) Query(query *model.LogQuery) (*model.LogQueryResult, error) {
	if query == nil {
		query = model.DefaultLogQuery()
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	startTime, endTime := query.TimeRange()

	var matchingIDs []string

	for _, id := range s.timeIndex {
		v, _ := s.entries.Get(id)
		entry := v.(*model.LogEntry)
		if entry.Timestamp.Before(startTime) {
			continue
		}
		if entry.Timestamp.After(endTime) {
			continue
		}
		if query.Source != "" && entry.Source != query.Source {
			continue
		}
		if query.Level != "" && entry.Level != query.Level {
			continue
		}
		if len(query.Keywords) > 0 {
			if !model.ContainsAllKeywords(entry.Message, query.Keywords) {
				continue
			}
		}
		matchingIDs = append(matchingIDs, id)
	}

	total := len(matchingIDs)

	var entries []*model.LogEntry
	for _, id := range matchingIDs {
		v, _ := s.entries.Get(id)
		entries = append(entries, v.(*model.LogEntry))
	}

	sort.Slice(entries, func(i, j int) bool {
		if query.SortOrder == model.SortDescending {
			return entries[i].Timestamp.After(entries[j].Timestamp)
		}
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	if query.Limit == 0 {
		query.Limit = 100
	}

	if query.Offset > total {
		query.Offset = total
	}
	end := query.Offset + query.Limit
	if end > total {
		end = total
	}
	pagedEntries := entries[query.Offset:end]

	return model.NewLogQueryResult(*query, pagedEntries, total), nil
}

// Count 统计数量
func (s *MemoryLogStore) Count(source string, level model.LogLevel, startTime, endTime time.Time) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64
	for _, id := range s.timeIndex {
		v, _ := s.entries.Get(id)
		entry := v.(*model.LogEntry)
		if !entry.Timestamp.Before(startTime) && !entry.Timestamp.After(endTime) {
			if source != "" && entry.Source != source {
				continue
			}
			if level != "" && entry.Level != level {
				continue
			}
			count++
		}
	}
	return count, nil
}

// DeleteByID 根据ID删除
func (s *MemoryLogStore) DeleteByID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, _ := s.entries.Get(id)
	entry := v.(*model.LogEntry)

	s.removeEntry(entry)
	s.entries.Delete(id)
	return nil
}

// DeleteBySource 删除指定来源的日志
func (s *MemoryLogStore) DeleteBySource(source string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids, ok := s.sourceIndex[source]
	if !ok || len(ids) == 0 {
		return 0, nil
	}

	var count int64
	for _, id := range ids {
		v, _ := s.entries.Get(id)
		entry := v.(*model.LogEntry)
		s.removeEntry(entry)
		s.entries.Delete(id)
		count++
	}

	delete(s.sourceIndex, source)
	return count, nil
}

// Cleanup 清理过期日志
func (s *MemoryLogStore) Cleanup(retentionPeriod time.Duration) (int64, error) {
	if retentionPeriod <= 0 {
		return 0, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-retentionPeriod)
	var count int64

	for _, id := range s.timeIndex {
		v, _ := s.entries.Get(id)
		entry := v.(*model.LogEntry)
		if entry.Timestamp.Before(cutoff) {
			s.removeEntry(entry)
			s.entries.Delete(id)
			count++
		}
	}

	return count, nil
}

// Stats 获取存储统计
func (s *MemoryLogStore) Stats() (*StoreStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &StoreStats{
		TotalEntries: int64(len(s.timeIndex)),
		BySource:     make(map[string]int64),
		ByLevel:      make(map[model.LogLevel]int64),
	}

	for _, id := range s.timeIndex {
		v, _ := s.entries.Get(id)
		entry := v.(*model.LogEntry)
		stats.BySource[entry.Source]++
		stats.ByLevel[entry.Level]++
	}

	return stats, nil
}

// RawSnapshot 获取原始快照（用于诊断）
func (s *MemoryLogStore) RawSnapshot() map[string]*model.LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*model.LogEntry)
	s.entries.Range(func(key string, value interface{}) bool {
		if entry, ok := value.(*model.LogEntry); ok && entry != nil {
			result[key] = entry
		}
		return true
	})
	return result
}

// evictOldest 淘汰最旧的条目
func (s *MemoryLogStore) evictOldest() {
	if len(s.timeIndex) == 0 {
		return
	}
	oldestID := s.timeIndex[0]
	v, _ := s.entries.Get(oldestID)
	entry := v.(*model.LogEntry)
	s.removeEntry(entry)
	s.entries.Delete(oldestID)
}

// insertTimeIndex 插入时间索引（保持排序）
func (s *MemoryLogStore) insertTimeIndex(entry *model.LogEntry) {
	pos := sort.Search(len(s.timeIndex), func(i int) bool {
		v, _ := s.entries.Get(s.timeIndex[i])
		e := v.(*model.LogEntry)
		return e.Timestamp.After(entry.Timestamp)
	})

	s.timeIndex = append(s.timeIndex, "")
	copy(s.timeIndex[pos+1:], s.timeIndex[pos:])
	s.timeIndex[pos] = entry.ID
}

// removeEntry 从索引中移除条目
func (s *MemoryLogStore) removeEntry(entry *model.LogEntry) {
	if entry == nil {
		return
	}

	if ids, ok := s.sourceIndex[entry.Source]; ok {
		for i, id := range ids {
			if id == entry.ID {
				s.sourceIndex[entry.Source] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
		if len(s.sourceIndex[entry.Source]) == 0 {
			delete(s.sourceIndex, entry.Source)
		}
	}

	if ids, ok := s.levelIndex[entry.Level]; ok {
		for i, id := range ids {
			if id == entry.ID {
				s.levelIndex[entry.Level] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
		if len(s.levelIndex[entry.Level]) == 0 {
			delete(s.levelIndex, entry.Level)
		}
	}

	for _, kw := range entry.Keywords {
		kw = normalizeKW(kw)
		if ids, ok := s.keywordIndex[kw]; ok {
			for i, id := range ids {
				if id == entry.ID {
					s.keywordIndex[kw] = append(ids[:i], ids[i+1:]...)
					break
				}
			}
			if len(s.keywordIndex[kw]) == 0 {
				delete(s.keywordIndex, kw)
			}
		}
	}
}

// normalizeKW 规范化关键词
func normalizeKW(kw string) string {
	result := make([]byte, len(kw))
	for i := 0; i < len(kw); i++ {
		c := kw[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}