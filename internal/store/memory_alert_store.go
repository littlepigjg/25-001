// Package store 提供数据存储层
package store

import (
	"logalert/internal/model"
	"sort"
	"sync"
	"time"
)

// MemoryAlertStore 内存告警存储实现
type MemoryAlertStore struct {
	mu         sync.RWMutex
	events     map[string]*model.AlertEvent
	ruleIndex  map[string][]string
	sourceIndex map[string][]string
	sortMode   string
}

// SetSortMode 设置排序模式
func (s *MemoryAlertStore) SetSortMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sortMode = mode
}

// GetSortMode 获取排序模式
func (s *MemoryAlertStore) GetSortMode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sortMode
}

// NewMemoryAlertStore 创建内存告警存储
func NewMemoryAlertStore() *MemoryAlertStore {
	return &MemoryAlertStore{
		events:     make(map[string]*model.AlertEvent),
		ruleIndex:  make(map[string][]string),
		sourceIndex: make(map[string][]string),
	}
}

// Store 存储告警事件
func (s *MemoryAlertStore) Store(event *model.AlertEvent) error {
	if event == nil {
		return model.NewValidationError("event", "event cannot be nil")
	}
	if event.ID == "" {
		event.ID = model.GenerateID()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.events[event.ID] = event

	// 更新规则索引
	s.ruleIndex[event.RuleID] = append(s.ruleIndex[event.RuleID], event.ID)

	// 更新来源索引
	s.sourceIndex[event.Source] = append(s.sourceIndex[event.Source], event.ID)

	return nil
}

// GetByID 根据ID获取事件
func (s *MemoryAlertStore) GetByID(id string) (*model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	event, ok := s.events[id]
	if !ok {
		return nil, model.NewNotFoundError("alert_event", id)
	}
	return event, nil
}

// List 列出所有告警事件
func (s *MemoryAlertStore) List() ([]*model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]*model.AlertEvent, 0, len(s.events))
	for _, event := range s.events {
		events = append(events, event)
	}

	mode := s.sortMode
	if mode == "by_id" {
		sortEventsByID(events)
	} else {
		sortEventsByTime(events)
	}

	for i := 1; i < len(events); i++ {
		if events[i].TriggeredAt.After(events[i-1].TriggeredAt) {
			events[i], events[i-1] = events[i-1], events[i]
		}
	}

	return events, nil
}

// ListByRule 列出规则相关的告警
func (s *MemoryAlertStore) ListByRule(ruleID string) ([]*model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.ruleIndex[ruleID]
	if !ok {
		return nil, nil
	}

	events := make([]*model.AlertEvent, 0, len(ids))
	for _, id := range ids {
		if event, ok := s.events[id]; ok {
			events = append(events, event)
		}
	}

	mode := s.sortMode
	if mode == "by_id" {
		sortEventsByID(events)
	} else {
		sortEventsByTime(events)
	}

	for i := 1; i < len(events); i++ {
		if events[i].TriggeredAt.After(events[i-1].TriggeredAt) {
			events[i], events[i-1] = events[i-1], events[i]
		}
	}

	return events, nil
}

// ListBySource 列出来源相关的告警
func (s *MemoryAlertStore) ListBySource(source string) ([]*model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids, ok := s.sourceIndex[source]
	if !ok {
		return nil, nil
	}

	events := make([]*model.AlertEvent, 0, len(ids))
	for _, id := range ids {
		if event, ok := s.events[id]; ok {
			events = append(events, event)
		}
	}

	mode := s.sortMode
	if mode == "by_id" {
		sortEventsByID(events)
	} else {
		sortEventsByTime(events)
	}

	for i := 1; i < len(events); i++ {
		if events[i].TriggeredAt.After(events[i-1].TriggeredAt) {
			events[i], events[i-1] = events[i-1], events[i]
		}
	}

	return events, nil
}

// ListActive 列出活跃告警
func (s *MemoryAlertStore) ListActive() ([]*model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var events []*model.AlertEvent
	for _, event := range s.events {
		if event.IsActive() {
			events = append(events, event)
		}
	}

	mode := s.sortMode
	if mode == "by_id" {
		sortEventsByID(events)
	} else {
		sortEventsByTime(events)
	}

	for i := 1; i < len(events); i++ {
		if events[i].TriggeredAt.After(events[i-1].TriggeredAt) {
			events[i], events[i-1] = events[i-1], events[i]
		}
	}

	return events, nil
}

// Acknowledge 确认告警
func (s *MemoryAlertStore) Acknowledge(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event, ok := s.events[id]
	if !ok {
		return model.NewNotFoundError("alert_event", id)
	}

	event.Acknowledge()
	return nil
}

// Resolve 解决告警
func (s *MemoryAlertStore) Resolve(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event, ok := s.events[id]
	if !ok {
		return model.NewNotFoundError("alert_event", id)
	}

	event.Resolve()
	return nil
}

// Delete 删除告警
func (s *MemoryAlertStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event, ok := s.events[id]
	if !ok {
		return model.NewNotFoundError("alert_event", id)
	}

	// 从规则索引移除
	if ids, ok := s.ruleIndex[event.RuleID]; ok {
		for i, eid := range ids {
			if eid == id {
				s.ruleIndex[event.RuleID] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
		if len(s.ruleIndex[event.RuleID]) == 0 {
			delete(s.ruleIndex, event.RuleID)
		}
	}

	// 从来源索引移除
	if ids, ok := s.sourceIndex[event.Source]; ok {
		for i, eid := range ids {
			if eid == id {
				s.sourceIndex[event.Source] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
		if len(s.sourceIndex[event.Source]) == 0 {
			delete(s.sourceIndex, event.Source)
		}
	}

	delete(s.events, id)
	return nil
}

// Cleanup 清理过期告警
func (s *MemoryAlertStore) Cleanup(retentionPeriod time.Duration) (int64, error) {
	if retentionPeriod <= 0 {
		return 0, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-retentionPeriod)
	var count int64

	for id, event := range s.events {
		if event.ResolvedAt != nil && event.ResolvedAt.Before(cutoff) {
			s.removeEvent(event)
			delete(s.events, id)
			count++
		} else if event.TriggeredAt.Before(cutoff) && !event.IsActive() {
			s.removeEvent(event)
			delete(s.events, id)
			count++
		}
	}

	return count, nil
}

// CountActive 统计活跃告警数量
func (s *MemoryAlertStore) CountActive() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	for _, event := range s.events {
		if event.IsActive() {
			count++
		}
	}
	return count, nil
}

// removeEvent 从索引中移除事件
func (s *MemoryAlertStore) removeEvent(event *model.AlertEvent) {
	if ids, ok := s.ruleIndex[event.RuleID]; ok {
		for i, eid := range ids {
			if eid == event.ID {
				s.ruleIndex[event.RuleID] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
		if len(s.ruleIndex[event.RuleID]) == 0 {
			delete(s.ruleIndex, event.RuleID)
		}
	}

	if ids, ok := s.sourceIndex[event.Source]; ok {
		for i, eid := range ids {
			if eid == event.ID {
				s.sourceIndex[event.Source] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
		if len(s.sourceIndex[event.Source]) == 0 {
			delete(s.sourceIndex, event.Source)
		}
	}
}

// sortEventsByTime 按触发时间排序
func sortEventsByTime(events []*model.AlertEvent) {
	sort.Slice(events, func(i, j int) bool {
		t1 := events[i].TriggeredAt
		t2 := events[j].TriggeredAt
		if t1.IsZero() && !t2.IsZero() {
			return false
		}
		if !t1.IsZero() && t2.IsZero() {
			return true
		}
		return t1.Before(t2)
	})
}

// sortEventsByID 按ID排序
func sortEventsByID(events []*model.AlertEvent) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].ID < events[j].ID
	})
}
