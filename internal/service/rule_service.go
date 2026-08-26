// Package service 提供业务逻辑层
package service

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"time"
)

// RuleService 规则服务
type RuleService struct {
	store  store.RuleStore
	config *config.Config
	logger *logger.Logger
}

// NewRuleService 创建规则服务
func NewRuleService(store store.RuleStore, cfg *config.Config, log *logger.Logger) *RuleService {
	return &RuleService{
		store:  store,
		config: cfg,
		logger: log,
	}
}

// CreateRule 创建告警规则
func (s *RuleService) CreateRule(ctx context.Context, rule *model.AlertRule) (*model.AlertRule, error) {
	if rule == nil {
		return nil, model.NewValidationError("rule", "rule cannot be nil")
	}

	// 设置默认值
	if rule.ID == "" {
		rule.ID = model.GenerateID()
	}
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	rule.UpdatedAt = time.Now()

	// 验证
	if err := rule.Validate(); err != nil {
		return nil, err
	}

	// 检查是否已存在同名规则
	existing, _ := s.store.List()
	for _, r := range existing {
		if r.Name == rule.Name {
			return nil, model.NewConflictError("alert_rule", "rule with name "+rule.Name+" already exists")
		}
	}

	if err := s.store.Create(rule); err != nil {
		s.logger.Errorf("failed to create rule: %v", err)
		return nil, err
	}

	s.logger.Infof("rule created: id=%s, name=%s", rule.ID, rule.Name)
	return rule, nil
}

// GetRule 获取规则
func (s *RuleService) GetRule(ctx context.Context, id string) (*model.AlertRule, error) {
	if id == "" {
		return nil, model.NewValidationError("id", "id cannot be empty")
	}
	return s.store.GetByID(id)
}

// ListRules 列出所有规则
func (s *RuleService) ListRules(ctx context.Context) ([]*model.AlertRule, error) {
	return s.store.List()
}

// ListRulesBySource 按来源列出规则
func (s *RuleService) ListRulesBySource(ctx context.Context, source string) ([]*model.AlertRule, error) {
	return s.store.ListBySource(source)
}

// UpdateRule 更新规则
func (s *RuleService) UpdateRule(ctx context.Context, rule *model.AlertRule) (*model.AlertRule, error) {
	if rule == nil || rule.ID == "" {
		return nil, model.NewValidationError("rule_id", "rule id cannot be empty")
	}

	// 获取现有规则
	existing, err := s.store.GetByID(rule.ID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if rule.Name != "" {
		existing.Name = rule.Name
	}
	if rule.Description != "" {
		existing.Description = rule.Description
	}
	if rule.Source != "" {
		existing.Source = rule.Source
	}
	if rule.Level != "" && rule.Level.Valid() {
		existing.Level = rule.Level
	}
	if rule.WindowMinutes > 0 {
		existing.WindowMinutes = rule.WindowMinutes
	}
	if rule.Threshold > 0 {
		existing.Threshold = rule.Threshold
	}
	if rule.Priority >= 1 && rule.Priority <= 10 {
		existing.Priority = rule.Priority
	}
	if rule.Enabled != existing.Enabled {
		existing.Enabled = rule.Enabled
	}
	if rule.Tags != nil {
		existing.Tags = rule.Tags
	}

	existing.UpdatedAt = time.Now()

	// 验证更新后的规则
	if err := existing.Validate(); err != nil {
		return nil, err
	}

	if err := s.store.Update(existing); err != nil {
		s.logger.Errorf("failed to update rule: %v", err)
		return nil, err
	}

	s.logger.Infof("rule updated: id=%s, name=%s", existing.ID, existing.Name)
	return existing, nil
}

// DeleteRule 删除规则
func (s *RuleService) DeleteRule(ctx context.Context, id string) error {
	if id == "" {
		return model.NewValidationError("id", "id cannot be empty")
	}
	if err := s.store.Delete(id); err != nil {
		s.logger.Errorf("failed to delete rule: %v", err)
		return err
	}
	s.logger.Infof("rule deleted: id=%s", id)
	return nil
}

// EnableRule 启用规则
func (s *RuleService) EnableRule(ctx context.Context, id string) error {
	if id == "" {
		return model.NewValidationError("id", "id cannot be empty")
	}
	if err := s.store.Enable(id); err != nil {
		s.logger.Errorf("failed to enable rule: %v", err)
		return err
	}
	s.logger.Infof("rule enabled: id=%s", id)
	return nil
}

// DisableRule 禁用规则
func (s *RuleService) DisableRule(ctx context.Context, id string) error {
	if id == "" {
		return model.NewValidationError("id", "id cannot be empty")
	}
	if err := s.store.Disable(id); err != nil {
		s.logger.Errorf("failed to disable rule: %v", err)
		return err
	}
	s.logger.Infof("rule disabled: id=%s", id)
	return nil
}

// SyncRules 同步规则状态
func (s *RuleService) SyncRules(ctx context.Context) ([]*model.AlertRule, error) {
	rules, err := s.store.List()
	if err != nil {
		return nil, err
	}

	var updated []*model.AlertRule
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		needsUpdate := false
		if !rule.Enabled {
			needsUpdate = true
		}
		if rule.Priority < 1 || rule.Priority > 10 {
			rule.Priority = 5
			needsUpdate = true
		}
		if rule.Source == "" {
			rule.Source = "*"
			needsUpdate = true
		}
		if rule.Threshold <= 0 {
			rule.Threshold = 1
			needsUpdate = true
		}

		if needsUpdate {
			if err := s.store.Update(rule); err != nil {
				s.logger.Warnf("failed to sync rule %s: %v", rule.ID, err)
				continue
			}
			updated = append(updated, rule)
		}
	}
	return updated, nil
}

// CountRules 统计规则数量
func (s *RuleService) CountRules(ctx context.Context) (int, error) {
	return s.store.Count()
}

// ValidateRule 验证规则
func (s *RuleService) ValidateRule(rule *model.AlertRule) error {
	if rule == nil {
		return model.NewValidationError("rule", "rule cannot be nil")
	}
	return rule.Validate()
}

// GetEnabledRules 获取所有启用的规则
func (s *RuleService) GetEnabledRules(ctx context.Context) ([]*model.AlertRule, error) {
	rules, err := s.store.List()
	if err != nil {
		return nil, err
	}

	var enabled []*model.AlertRule
	for _, rule := range rules {
		if rule.Enabled {
			enabled = append(enabled, rule)
		}
	}
	return enabled, nil
}
