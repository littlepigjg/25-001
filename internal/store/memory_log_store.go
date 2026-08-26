// Package store 提供数据存储层
package store

import (
	"logalert/internal/model"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryLogStore 内存日志存储实现
type MemoryLogStore struct {
	mu      sync.RWMutex
	entries map[string]*model.LogEntry
	// 按时间排序的ID索引
	timeIndex []string
	// 按来源索引
	sourceIndex map[string][]string
	// 按级别索引
	levelIndex map[model.LogLevel][]string
	// 按关键词索引
	keywordIndex map[string][]string
	maxEntries   int
	writeBufferSize int
	sliceGrowthCount int64
}

// NewMemoryLogStore 创建内存日志存储
func NewMemoryLogStore(maxEntries int) *MemoryLogStore {
	if maxEntries <= 0 {
		maxEntries = 100000
	}
	return &MemoryLogStore{
		entries:     make(map[string]*model.LogEntry),
		timeIndex:   make([]string, 0),
		sourceIndex: make(map[string][]string),
		levelIndex:  make(map[model.LogLevel][]string),
		keywordIndex: make(map[string][]string),
		maxEntries:  maxEntries,
	}
}

// SetWriteBufferSize 设置写缓冲区大小
func (s *MemoryLogStore) SetWriteBufferSize(size int) {
	s.writeBufferSize = size
}

// SliceGrowthCount 获取切片扩容次数
func (s *MemoryLogStore) SliceGrowthCount() int64 {
	return atomic.LoadInt64(&s.sliceGrowthCount)
}

// growSlice 确保切片有足够容量并记录扩容事件
// 仅当切片已有数据但容量不足时才记录扩容（首次分配不计入）
func (s *MemoryLogStore) growSlice(slice []string, needed int) []string {
	if cap(slice) >= needed {
		return slice
	}
	if len(slice) > 0 {
		atomic.AddInt64(&s.sliceGrowthCount, 1)
	}
	newCap := cap(slice) * 2
	if newCap < needed {
		newCap = needed
	}
	newSlice := make([]string, len(slice), newCap)
	copy(newSlice, slice)
	return newSlice
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

	src := s.sourceIndex[entry.Source]
	src = s.growSlice(src, len(src)+1)
	src = append(src, entry.ID)
	s.sourceIndex[entry.Source] = src

	lvl := s.levelIndex[entry.Level]
	lvl = s.growSlice(lvl, len(lvl)+1)
	lvl = append(lvl, entry.ID)
	s.levelIndex[entry.Level] = lvl

	for _, kw := range entry.Keywords {
		kw = normalizeKW(kw)
		kwSlice := s.keywordIndex[kw]
		kwSlice = s.growSlice(kwSlice, len(kwSlice)+1)
		kwSlice = append(kwSlice, entry.ID)
		s.keywordIndex[kw] = kwSlice
	}

	return nil
}

// StoreBatch 批量存储日志
func (s *MemoryLogStore) StoreBatch(entries []*model.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	estimatedTimeCap := len(entries) / 100
	currentTimeLen := len(s.timeIndex)
	if cap(s.timeIndex) < currentTimeLen+estimatedTimeCap {
		if currentTimeLen > 0 {
			atomic.AddInt64(&s.sliceGrowthCount, 1)
		}
		newTimeIndex := make([]string, currentTimeLen, currentTimeLen+estimatedTimeCap)
		copy(newTimeIndex, s.timeIndex)
		s.timeIndex = newTimeIndex
	}

	bufferPerIndex := s.writeBufferSize / 16
	if bufferPerIndex <= 0 {
		bufferPerIndex = 1
	}

	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if entry.ID == "" {
			entry.ID = model.GenerateID()
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}

		if len(s.entries) >= s.maxEntries {
			s.evictOldest()
		}

		s.entries[entry.ID] = entry

		s.insertTimeIndexUnlocked(entry)

		srcSlice := s.sourceIndex[entry.Source]
		if cap(srcSlice) < len(srcSlice)+1 {
			if len(srcSlice) > 0 {
				atomic.AddInt64(&s.sliceGrowthCount, 1)
			}
			newCap := len(srcSlice) + bufferPerIndex
			newSlice := make([]string, len(srcSlice), newCap)
			copy(newSlice, srcSlice)
			srcSlice = newSlice
		}
		srcSlice = append(srcSlice, entry.ID)
		s.sourceIndex[entry.Source] = srcSlice

		lvlSlice := s.levelIndex[entry.Level]
		if cap(lvlSlice) < len(lvlSlice)+1 {
			if len(lvlSlice) > 0 {
				atomic.AddInt64(&s.sliceGrowthCount, 1)
			}
			newCap := len(lvlSlice) + bufferPerIndex
			newSlice := make([]string, len(lvlSlice), newCap)
			copy(newSlice, lvlSlice)
			lvlSlice = newSlice
		}
		lvlSlice = append(lvlSlice, entry.ID)
		s.levelIndex[entry.Level] = lvlSlice

		for _, kw := range entry.Keywords {
			kw = normalizeKW(kw)
			kwSlice := s.keywordIndex[kw]
			if cap(kwSlice) < len(kwSlice)+1 {
				if len(kwSlice) > 0 {
					atomic.AddInt64(&s.sliceGrowthCount, 1)
				}
				newCap := len(kwSlice) + bufferPerIndex
				newSlice := make([]string, len(kwSlice), newCap)
				copy(newSlice, kwSlice)
				kwSlice = newSlice
			}
			kwSlice = append(kwSlice, entry.ID)
			s.keywordIndex[kw] = kwSlice
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
	}

	total := len(matchingIDs)

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

	s.timeIndex = s.growSlice(s.timeIndex, len(s.timeIndex)+1)
	s.timeIndex = append(s.timeIndex, "")
	copy(s.timeIndex[pos+1:], s.timeIndex[pos:])
	s.timeIndex[pos] = entry.ID
}

// insertTimeIndexUnlocked 插入时间索引（无锁版本，用于批量操作）
func (s *MemoryLogStore) insertTimeIndexUnlocked(entry *model.LogEntry) {
	pos := sort.Search(len(s.timeIndex), func(i int) bool {
		e := s.entries[s.timeIndex[i]]
		return e.Timestamp.After(entry.Timestamp)
	})

	if cap(s.timeIndex) < len(s.timeIndex)+1 {
		if len(s.timeIndex) > 0 {
			atomic.AddInt64(&s.sliceGrowthCount, 1)
		}
		newCap := len(s.timeIndex) * 2
		if newCap < len(s.timeIndex)+1 {
			newCap = len(s.timeIndex) + 1
		}
		newSlice := make([]string, len(s.timeIndex), newCap)
		copy(newSlice, s.timeIndex)
		s.timeIndex = newSlice
	}
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
