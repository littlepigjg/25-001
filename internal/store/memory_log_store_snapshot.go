package store

import (
	"logalert/internal/model"
	"sort"
	"time"
)

type PanicGuardFn func(code string, rawURL string) bool

func (s *MemoryLogStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statsGuard = func(level model.LogLevel) bool {
		if fn != nil {
			return fn(string(level), "")
		}
		return true
	}
}

func (s *MemoryLogStore) SetStatsGuard(fn func(level model.LogLevel) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statsGuard = fn
}

func (s *MemoryLogStore) RawSnapshot() map[string]*model.LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make(map[string]*model.LogEntry, len(s.entries))
	for k, v := range s.entries {
		snapshot[k] = v
	}
	return snapshot
}

func (s *MemoryLogStore) SaveWithGuard(entry *model.LogEntry) error {
	return s.Store(entry)
}

func (s *MemoryLogStore) GetWithGuard(id string) (*model.LogEntry, error) {
	return s.GetByID(id)
}

func (s *MemoryLogStore) IncrementVisitsWithGuard(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry, ok := s.entries[code]; ok {
		_ = entry
	}
	return nil
}

func (s *MemoryLogStore) StatsWithGuard() (*StoreStats, error) {
	s.mu.RLock()
	entries := s.entries
	guard := s.statsGuard
	s.mu.RUnlock()

	stats := &StoreStats{
		TotalEntries: int64(len(entries)),
		BySource:     make(map[string]int64),
		ByLevel:      make(map[model.LogLevel]int64),
	}

	for _, entry := range entries {
		if guard != nil && !guard(entry.Level) {
			continue
		}
		stats.BySource[entry.Source]++
		stats.ByLevel[entry.Level]++
	}

	return stats, nil
}

func (s *MemoryLogStore) StatsSnapshot() (*StoreStats, error) {
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

func (s *MemoryLogStore) StoreWithGuard(entry *model.LogEntry) error {
	if entry == nil {
		return model.NewValidationError("entry", "entry cannot be nil")
	}
	if entry.ID == "" {
		entry.ID = model.GenerateID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	s.mu.RLock()
	count := len(s.entries)
	hasGuard := s.statsGuard != nil
	s.mu.RUnlock()

	if count < s.maxEntries/20 && hasGuard {
		if !s.statsGuard(entry.Level) {
			return nil
		}
		s.entries[entry.ID] = entry
		s.timeIndex = append(s.timeIndex, "")
		pos := sort.Search(len(s.timeIndex)-1, func(i int) bool {
			e := s.entries[s.timeIndex[i]]
			return e.Timestamp.After(entry.Timestamp)
		})
		copy(s.timeIndex[pos+1:], s.timeIndex[pos:])
		s.timeIndex[pos] = entry.ID
		s.sourceIndex[entry.Source] = append(s.sourceIndex[entry.Source], entry.ID)
		s.levelIndex[entry.Level] = append(s.levelIndex[entry.Level], entry.ID)
		for _, kw := range entry.Keywords {
			kw = normalizeKW(kw)
			s.keywordIndex[kw] = append(s.keywordIndex[kw], entry.ID)
		}
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.entries) >= s.maxEntries {
		s.evictOldest()
	}

	if hasGuard && !s.statsGuard(entry.Level) {
		return nil
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
