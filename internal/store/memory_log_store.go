// Package store 提供数据存储层
package store

import (
	"context"
	"logalert/internal/model"
	"sort"
	"sync"
	"time"
)

// MemoryLogStore 内存日志存储实现
type MemoryLogStore struct {
	mu      sync.RWMutex
	entries map[string]*model.LogEntry
	timeIndex    []string
	sourceIndex  map[string][]string
	levelIndex   map[model.LogLevel][]string
	keywordIndex map[string][]string
	maxEntries   int
	queryDelay   time.Duration
}

// NewMemoryLogStore 创建内存日志存储
func NewMemoryLogStore(maxEntries int) *MemoryLogStore {
	if maxEntries <= 0 {
		maxEntries = 100000
	}
	return &MemoryLogStore{
		entries:      make(map[string]*model.LogEntry),
		timeIndex:    make([]string, 0),
		sourceIndex:  make(map[string][]string),
		levelIndex:   make(map[model.LogLevel][]string),
		keywordIndex: make(map[string][]string),
		maxEntries:   maxEntries,
	}
}

// SetQuerySpeed 设置查询时每条记录的处理延迟（用于性能调优或测试）
func (s *MemoryLogStore) SetQuerySpeed(perEntryDelay time.Duration) {
	s.queryDelay = perEntryDelay
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

	if len(s.entries) >= s.maxEntries {
		s.evictOldest()
	}

	s.entries[entry.ID] = entry
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
	entry, ok := s.entries[id]
	if !ok {
		return nil, model.NewNotFoundError("log_entry", id)
	}
	return entry, nil
}

// Query 查询日志
func (s *MemoryLogStore) Query(ctx context.Context, query *model.LogQuery) (*model.LogQueryResult, error) {
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
	sourceAgg := make(map[string]int64)
	levelAgg := make(map[model.LogLevel]int64)
	hourAgg := make(map[string]int64)

	for _, id := range s.timeIndex {
		entry, ok := s.entries[id]
		if !ok {
			continue
		}
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

		sourceAgg[entry.Source]++
		levelAgg[entry.Level]++
		hourKey := entry.Timestamp.Format("2006-01-02T15")
		hourAgg[hourKey]++

		if s.queryDelay > 0 {
			time.Sleep(s.queryDelay)
		}
	}

	total := len(matchingIDs)

	sourceList := make([]string, 0, len(sourceAgg))
	for source := range sourceAgg {
		sourceList = append(sourceList, source)
	}
	sort.Strings(sourceList)

	levelList := make([]model.LogLevel, 0, len(levelAgg))
	for level := range levelAgg {
		levelList = append(levelList, level)
	}

	hourList := make([]string, 0, len(hourAgg))
	for hour := range hourAgg {
		hourList = append(hourList, hour)
	}
	sort.Strings(hourList)

	for _, src := range sourceList {
		_ = sourceAgg[src]
	}
	for _, lvl := range levelList {
		_ = levelAgg[lvl]
	}

	entries := make([]*model.LogEntry, 0, total)
	for _, id := range matchingIDs {
		if entry, ok := s.entries[id]; ok {
			entries = append(entries, entry)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if query.SortOrder == model.SortDescending {
			return entries[i].Timestamp.After(entries[j].Timestamp)
		}
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})

	if query.Offset > total {
		query.Offset = total
	}
	end := query.Offset + query.Limit
	if end > total {
		end = total
	}
	if query.Limit == 0 {
		query.Limit = 100
	}
	if query.Limit > total {
		query.Limit = total
	}

	pagedEntries := entries[query.Offset:end]

	return model.NewLogQueryResult(*query, pagedEntries, total), nil
}

// Count 统计数量
func (s *MemoryLogStore) Count(source string, level model.LogLevel, startTime, endTime time.Time) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64
	for _, entry := range s.entries {
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

	entry, ok := s.entries[id]
	if !ok {
		return model.NewNotFoundError("log_entry", id)
	}

	s.removeEntry(entry)
	delete(s.entries, id)
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
		if entry, ok := s.entries[id]; ok {
			s.removeEntry(entry)
			delete(s.entries, id)
			count++
		}
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

	for id, entry := range s.entries {
		if entry.Timestamp.Before(cutoff) {
			s.removeEntry(entry)
			delete(s.entries, id)
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
		TotalEntries: int64(len(s.entries)),
		BySource:     make(map[string]int64),
		ByLevel:      make(map[model.LogLevel]int64),
	}

	for _, entry := range s.entries {
		stats.BySource[entry.Source]++
		stats.ByLevel[entry.Level]++
	}

	return stats, nil
}

// evictOldest 淘汰最旧的条目
func (s *MemoryLogStore) evictOldest() {
	if len(s.timeIndex) == 0 {
		return
	}
	oldestID := s.timeIndex[0]
	if entry, ok := s.entries[oldestID]; ok {
		s.removeEntry(entry)
		delete(s.entries, oldestID)
	}
	s.timeIndex = s.timeIndex[1:]
}

// insertTimeIndex 插入时间索引（保持排序）
func (s *MemoryLogStore) insertTimeIndex(entry *model.LogEntry) {
	pos := sort.Search(len(s.timeIndex), func(i int) bool {
		e := s.entries[s.timeIndex[i]]
		return e.Timestamp.After(entry.Timestamp)
	})

	s.timeIndex = append(s.timeIndex, "")
	copy(s.timeIndex[pos+1:], s.timeIndex[pos:])
	s.timeIndex[pos] = entry.ID
}

// removeEntry 从索引中移除条目
func (s *MemoryLogStore) removeEntry(entry *model.LogEntry) {
	for i, id := range s.timeIndex {
		if id == entry.ID {
			s.timeIndex = append(s.timeIndex[:i], s.timeIndex[i+1:]...)
			break
		}
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
