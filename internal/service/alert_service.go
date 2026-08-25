package service

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"time"
)

type ContextValidator func(ctx context.Context) error

type AlertService struct {
	alertStore   store.AlertStore
	logStore     store.LogStore
	ruleStore    store.RuleStore
	config       *config.Config
	logger       *logger.Logger
	guard        ContextValidator
	activeCtx    context.Context
}

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

func (s *AlertService) SetContextValidator(fn ContextValidator) {
	s.guard = fn
}

func (s *AlertService) SetActiveContext(ctx context.Context) {
	s.activeCtx = ctx
}

func (s *AlertService) validateContext(ctx context.Context) error {
	if ctx == nil {
		return model.NewValidationError("context", "context cannot be nil")
	}
	if s.guard != nil {
		if err := s.guard(ctx); err != nil {
			return err
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if s.activeCtx != nil && s.activeCtx.Err() != nil {
		return s.activeCtx.Err()
	}
	return nil
}

func (s *AlertService) EvaluateRule(ctx context.Context, rule *model.AlertRule) (*model.AlertEvent, error) {
	if err := s.validateContext(ctx); err != nil {
		s.logger.Errorf("context validation failed before rule evaluation: %v", err)
		return nil, err
	}

	if rule == nil {
		return nil, model.NewValidationError("rule", "rule cannot be nil")
	}
	if !rule.Enabled {
		return nil, nil
	}

	if err := s.validateContext(ctx); err != nil {
		s.logger.Errorf("context validation failed before query: %v", err)
		return nil, err
	}

	now := time.Now()
	windowStart := now.Add(-time.Duration(rule.WindowMinutes) * time.Minute)

	query := &model.LogQuery{
		Source:    rule.Source,
		Level:     rule.Level,
		StartTime: &windowStart,
		EndTime:   &now,
	}

	if err := s.validateContext(ctx); err != nil {
		s.logger.Errorf("context validation failed during query: %v", err)
		return nil, err
	}

	result, err := s.logStore.Query(query)
	if err != nil {
		s.logger.Errorf("failed to query logs for rule evaluation: %v", err)
		return nil, err
	}

	if err := s.validateContext(ctx); err != nil {
		s.logger.Errorf("context validation failed before threshold check: %v", err)
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

func (s *AlertService) EvaluateAllRules(ctx context.Context) ([]*model.AlertEvent, error) {
	if err := s.validateContext(ctx); err != nil {
		s.logger.Errorf("context validation failed before listing rules: %v", err)
		return nil, err
	}

	rules, err := s.ruleStore.List()
	if err != nil {
		return nil, err
	}

	var triggeredEvents []*model.AlertEvent
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		if err := s.validateContext(ctx); err != nil {
			s.logger.Errorf("context validation failed during rule iteration: %v", err)
			return nil, err
		}

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

func (s *AlertService) GetAlert(ctx context.Context, id string) (*model.AlertEvent, error) {
	if id == "" {
		return nil, model.NewValidationError("id", "id cannot be empty")
	}
	return s.alertStore.GetByID(id)
}

func (s *AlertService) ListAlerts(ctx context.Context) ([]*model.AlertEvent, error) {
	return s.alertStore.List()
}

func (s *AlertService) ListAlertsByRule(ctx context.Context, ruleID string) ([]*model.AlertEvent, error) {
	return s.alertStore.ListByRule(ruleID)
}

func (s *AlertService) ListAlertsBySource(ctx context.Context, source string) ([]*model.AlertEvent, error) {
	return s.alertStore.ListBySource(source)
}

func (s *AlertService) ListActiveAlerts(ctx context.Context) ([]*model.AlertEvent, error) {
	return s.alertStore.ListActive()
}

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

func (s *AlertService) AutoResolveExpired(ctx context.Context) (int, error) {
	if !s.config.Alert.EnableAutoResolve {
		return 0, nil
	}

	activeAlerts, err := s.alertStore.ListActive()
	if err != nil {
		return 0, err
	}

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

func (s *AlertService) CountActiveAlerts(ctx context.Context) (int, error) {
	return s.alertStore.CountActive()
}
