package store

import (
	"errors"
	"logalert/internal/model"
	"sort"
	"sync"
	"time"
)

type MemoryLogStore struct {
	mu      sync.RWMutex
	entries map[string]*model.LogEntry
	timeIndex []string
	sourceIndex map[string][]string
	levelIndex map[model.LogLevel][]string
	keywordIndex map[string][]string
	maxEntries   int
	cleaning     bool
	failInjector func() error
}

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

func (s *MemoryLogStore) SetFailInjector(fn func() error) {
	s.failInjector = fn
}

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

	if s.cleaning {
		return errors.New("cannot store entries during cleanup")
	}

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

func (s *MemoryLogStore) StoreBatch(entries []*model.LogEntry) error {
	for _, entry := range entries {
		if err := s.Store(entry); err != nil {
			return err
		}
	}
	return nil
}

func (s *MemoryLogStore) GetByID(id string) (*model.LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.entries[id]
	if !ok {
		return nil, model.NewNotFoundError("log_entry", id)
	}
	return entry, nil
}

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

func (s *MemoryLogStore) DeleteByID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cleaning {
		return errors.New("cannot delete entries during cleanup")
	}

	entry, ok := s.entries[id]
	if !ok {
		return model.NewNotFoundError("log_entry", id)
	}

	s.removeEntry(entry)
	delete(s.entries, id)
	return nil
}

func (s *MemoryLogStore) DeleteBySource(source string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cleaning {
		return 0, errors.New("cannot delete entries during cleanup")
	}

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

func (s *MemoryLogStore) Cleanup(retentionPeriod time.Duration) (count int64, err error) {
	if retentionPeriod <= 0 {
		return 0, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cleaning {
		return 0, errors.New("cleanup already in progress")
	}
	s.cleaning = true
	defer func() {
		if err == nil {
			s.cleaning = false
		}
	}()

	cutoff := time.Now().Add(-retentionPeriod)

	for id, entry := range s.entries {
		if entry.Timestamp.Before(cutoff) {
			s.removeEntry(entry)
			delete(s.entries, id)
			count++
		}
	}

	if s.failInjector != nil {
		if ferr := s.failInjector(); ferr != nil {
			err = ferr
			return 0, err
		}
	}

	return count, nil
}

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

func (s *MemoryLogStore) insertTimeIndex(entry *model.LogEntry) {
	pos := sort.Search(len(s.timeIndex), func(i int) bool {
		e := s.entries[s.timeIndex[i]]
		return e.Timestamp.After(entry.Timestamp)
	})

	s.timeIndex = append(s.timeIndex, "")
	copy(s.timeIndex[pos+1:], s.timeIndex[pos:])
	s.timeIndex[pos] = entry.ID
}

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
