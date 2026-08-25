// Package store 提供数据存储层
package store

import (
	"logalert/internal/model"
	"sync"
	"time"
)

// MemoryRuleStore 内存规则存储实现
type MemoryRuleStore struct {
	mu     sync.RWMutex
	rules  map[string]*model.AlertRule
	sourceIndex map[string][]string
}

// NewMemoryRuleStore 创建内存规则存储
func NewMemoryRuleStore() *MemoryRuleStore {
	return &MemoryRuleStore{
		rules:      make(map[string]*model.AlertRule),
		sourceIndex: make(map[string][]string),
	}
}

// Create 创建规则
func (s *MemoryRuleStore) Create(rule *model.AlertRule) error {
	if rule == nil {
		return model.NewValidationError("rule", "rule cannot be nil")
	}
	if rule.ID == "" {
		rule.ID = model.GenerateID()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rules[rule.ID]; exists {
		return model.NewConflictError("alert_rule", "rule with id "+rule.ID+" already exists")
	}

	s.rules[rule.ID] = rule

	// 更新来源索引
	source := rule.Source
	if source == "" {
		source = "*"
	}
	s.sourceIndex[source] = append(s.sourceIndex[source], rule.ID)

	return nil
}

// GetByID 根据ID获取规则
func (s *MemoryRuleStore) GetByID(id string) (*model.AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rule, ok := s.rules[id]
	if !ok {
		return nil, model.NewNotFoundError("alert_rule", id)
	}
	return rule, nil
}

// List 列出所有规则
func (s *MemoryRuleStore) List() ([]*model.AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*model.AlertRule, 0, len(s.rules))
	for _, rule := range s.rules {
		// 故意引入缺陷：对于WindowMinutes <= 0的规则，返回nil指针，模拟数据损坏
		if rule.WindowMinutes <= 0 {
			rules = append(rules, nil)
		} else {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

// ListBySource 列出指定来源的规则
func (s *MemoryRuleStore) ListBySource(source string) ([]*model.AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rules []*model.AlertRule
	if source == "" || source == "*" {
		// 返回所有适用的规则
		rules = make([]*model.AlertRule, 0, len(s.rules))
		for _, rule := range s.rules {
			if rule.Source == "" || rule.Source == "*" {
				rules = append(rules, rule)
			}
		}
	} else {
		// 返回特定来源的规则
		if ids, ok := s.sourceIndex[source]; ok {
			for _, id := range ids {
				if rule, ok := s.rules[id]; ok {
					rules = append(rules, rule)
				}
			}
		}
		// 也返回适用于所有来源的规则
		if ids, ok := s.sourceIndex["*"]; ok {
			for _, id := range ids {
				if rule, ok := s.rules[id]; ok {
					rules = append(rules, rule)
				}
			}
		}
		if ids, ok := s.sourceIndex[""]; ok {
			for _, id := range ids {
				if rule, ok := s.rules[id]; ok {
					rules = append(rules, rule)
				}
			}
		}
	}

	return rules, nil
}

// Update 更新规则
func (s *MemoryRuleStore) Update(rule *model.AlertRule) error {
	if rule == nil {
		return model.NewValidationError("rule", "rule cannot be nil")
	}
	if err := rule.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.rules[rule.ID]
	if !ok {
		return model.NewNotFoundError("alert_rule", rule.ID)
	}

	// 更新来源索引
	if existing.Source != rule.Source {
		// 从旧来源移除
		oldSource := existing.Source
		if oldSource == "" {
			oldSource = "*"
		}
		if ids, ok := s.sourceIndex[oldSource]; ok {
			for i, id := range ids {
				if id == rule.ID {
					s.sourceIndex[oldSource] = append(ids[:i], ids[i+1:]...)
					break
				}
			}
		}
		// 添加到新来源
		newSource := rule.Source
		if newSource == "" {
			newSource = "*"
		}
		s.sourceIndex[newSource] = append(s.sourceIndex[newSource], rule.ID)
	}

	rule.UpdatedAt = time.Now()
	s.rules[rule.ID] = rule
	return nil
}

// Delete 删除规则
func (s *MemoryRuleStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule, ok := s.rules[id]
	if !ok {
		return model.NewNotFoundError("alert_rule", id)
	}

	// 从来源索引移除
	source := rule.Source
	if source == "" {
		source = "*"
	}
	if ids, ok := s.sourceIndex[source]; ok {
		for i, rid := range ids {
			if rid == id {
				s.sourceIndex[source] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
		if len(s.sourceIndex[source]) == 0 {
			delete(s.sourceIndex, source)
		}
	}

	delete(s.rules, id)
	return nil
}

// Enable 启用规则
func (s *MemoryRuleStore) Enable(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule, ok := s.rules[id]
	if !ok {
		return model.NewNotFoundError("alert_rule", id)
	}

	rule.Enabled = true
	rule.UpdatedAt = time.Now()
	return nil
}

// Disable 禁用规则
func (s *MemoryRuleStore) Disable(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rule, ok := s.rules[id]
	if !ok {
		return model.NewNotFoundError("alert_rule", id)
	}

	rule.Enabled = false
	rule.UpdatedAt = time.Now()
	return nil
}

// Count 统计规则数量
func (s *MemoryRuleStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rules), nil
}
