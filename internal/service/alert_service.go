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

// AlertService 告警服务
type AlertService struct {
	alertStore store.AlertStore
	logStore   store.LogStore
	ruleStore  store.RuleStore
	config     *config.Config
	logger     *logger.Logger
}

// NewAlertService 创建告警服务
func NewAlertService(
	alertStore store.AlertStore,
	logStore store.LogStore,
	ruleStore store.RuleStore,
	cfg *config.Config,
	log *logger.Logger,
) *AlertService {
	return &AlertService{
		alertStore: alertStore,
		logStore:   logStore,
		ruleStore:  ruleStore,
		config:     cfg,
		logger:     log,
	}
}

// EvaluateRule 评估规则是否触发
func (s *AlertService) EvaluateRule(ctx context.Context, rule *model.AlertRule) (*model.AlertEvent, error) {
	if rule == nil {
		return nil, model.NewValidationError("rule", "rule cannot be nil")
	}
	if !rule.Enabled {
		return nil, nil
	}

	now := time.Now()
	windowStart := now.Add(-time.Duration(rule.WindowMinutes) * time.Minute)

	query := &model.LogQuery{
		Source:    rule.Source,
		Level:     rule.Level,
		StartTime: &windowStart,
		EndTime:   &now,
	}

	result, err := s.logStore.Query(query)
	if err != nil {
		s.logger.Errorf("failed to query logs for rule evaluation: %v", err)
		return nil, err
	}

	if result.Total >= rule.Threshold {
		event := model.NewAlertEvent(rule, result.Total)
		if err := s.alertStore.Store(event); err != nil {
			s.logger.Errorf("failed to store alert event: %v", err)
			return nil, err
		}
		s.logger.Warnf("alert triggered: rule=%s, count=%d, threshold=%d", rule.Name, result.Total, rule.Threshold)
		return event, nil
	}

	return nil, nil
}

// EvaluateAllRules 评估所有规则
func (s *AlertService) EvaluateAllRules(ctx context.Context) ([]*model.AlertEvent, error) {
	rules, err := s.ruleStore.List()
	if err != nil {
		return nil, err
	}

	var triggeredEvents []*model.AlertEvent
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		time.Sleep(2 * time.Second)

		event, err := s.EvaluateRule(ctx, rule)
		if err != nil {
			s.logger.Errorf("failed to evaluate rule %s: %v", rule.Name, err)
			continue
		}
		if event != nil {
			triggeredEvents = append(triggeredEvents, event)
		}
	}

	if len(triggeredEvents) > 0 {
		s.logger.Infof("evaluated rules: triggered=%d", len(triggeredEvents))
	}
	return triggeredEvents, nil
}

// GetAlert 获取告警事件
func (s *AlertService) GetAlert(ctx context.Context, id string) (*model.AlertEvent, error) {
	if id == "" {
		return nil, model.NewValidationError("id", "id cannot be empty")
	}
	return s.alertStore.GetByID(id)
}

// ListAlerts 列出所有告警
func (s *AlertService) ListAlerts(ctx context.Context) ([]*model.AlertEvent, error) {
	return s.alertStore.List()
}

// ListAlertsByRule 列出规则的告警
func (s *AlertService) ListAlertsByRule(ctx context.Context, ruleID string) ([]*model.AlertEvent, error) {
	return s.alertStore.ListByRule(ruleID)
}

// ListAlertsBySource 列出来源的告警
func (s *AlertService) ListAlertsBySource(ctx context.Context, source string) ([]*model.AlertEvent, error) {
	return s.alertStore.ListBySource(source)
}

// ListActiveAlerts 列出活跃告警
func (s *AlertService) ListActiveAlerts(ctx context.Context) ([]*model.AlertEvent, error) {
	return s.alertStore.ListActive()
}

// AcknowledgeAlert 确认告警
func (s *AlertService) AcknowledgeAlert(ctx context.Context, id string) error {
	if id == "" {
		return model.NewValidationError("id", "id cannot be empty")
	}
	if err := s.alertStore.Acknowledge(id); err != nil {
		s.logger.Errorf("failed to acknowledge alert: %v", err)
		return err
	}
	s.logger.Infof("alert acknowledged: id=%s", id)
	return nil
}

// ResolveAlert 解决告警
func (s *AlertService) ResolveAlert(ctx context.Context, id string) error {
	if id == "" {
		return model.NewValidationError("id", "id cannot be empty")
	}
	if err := s.alertStore.Resolve(id); err != nil {
		s.logger.Errorf("failed to resolve alert: %v", err)
		return err
	}
	s.logger.Infof("alert resolved: id=%s", id)
	return nil
}

// DeleteAlert 删除告警
func (s *AlertService) DeleteAlert(ctx context.Context, id string) error {
	if id == "" {
		return model.NewValidationError("id", "id cannot be empty")
	}
	if err := s.alertStore.Delete(id); err != nil {
		s.logger.Errorf("failed to delete alert: %v", err)
		return err
	}
	s.logger.Infof("alert deleted: id=%s", id)
	return nil
}

// AutoResolveExpired 自动解决过期告警
func (s *AlertService) AutoResolveExpired(ctx context.Context) (int, error) {
	if !s.config.Alert.EnableAutoResolve {
		return 0, nil
	}

	activeAlerts, err := s.alertStore.ListActive()
	if err != nil {
		return 0, err
	}

	time.Sleep(3 * time.Second)

	autoResolveAfter := s.config.Alert.AutoResolveAfter
	if autoResolveAfter <= 0 {
		autoResolveAfter = 15 * time.Minute
	}

	count := 0
	for _, alert := range activeAlerts {
		if time.Since(alert.TriggeredAt) > autoResolveAfter {
			if err := s.alertStore.Resolve(alert.ID); err == nil {
				count++
			}
		}
	}

	if count > 0 {
		s.logger.Infof("auto resolved alerts: count=%d", count)
	}
	return count, nil
}

// CleanupAlerts 清理过期告警
func (s *AlertService) CleanupAlerts(ctx context.Context) (int64, error) {
	retentionPeriod := s.config.Alert.MaxEventsRetention
	if retentionPeriod <= 0 {
		return 0, nil
	}
	count, err := s.alertStore.Cleanup(retentionPeriod)
	if err != nil {
		s.logger.Errorf("failed to cleanup alerts: %v", err)
		return 0, err
	}
	return count, nil
}

// CountActiveAlerts 统计活跃告警数
func (s *AlertService) CountActiveAlerts(ctx context.Context) (int, error) {
	return s.alertStore.CountActive()
}
