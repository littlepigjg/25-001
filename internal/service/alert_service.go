// Package service 提供业务逻辑层
package service

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"sync"
	"time"
)

// AlertService 告警服务
type AlertService struct {
	alertStore store.AlertStore
	logStore   store.LogStore
	ruleStore  store.RuleStore
	config     *config.Config
	logger     *logger.Logger

	evalCh    chan *evalRequest
	evalWg    sync.WaitGroup
	evalStart sync.Once
	evalStop  chan struct{}
	evalMu    sync.Mutex
}

// evalRequest 评估请求
type evalRequest struct {
	rule     *model.AlertRule
	resultCh chan *evalResult
}

// evalResult 评估结果
type evalResult struct {
	event *model.AlertEvent
	err   error
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
		evalCh:     make(chan *evalRequest, 16),
		evalStop:   make(chan struct{}),
	}
}

// StartEvaluation 启动评估管道
func (s *AlertService) StartEvaluation(ctx context.Context) {
	s.evalStart.Do(func() {
		s.evalWg.Add(1)
		go s.evalLoop(ctx)
		s.logger.Info("alert evaluation pipeline started")
	})
}

// StopEvaluation 停止评估管道
func (s *AlertService) StopEvaluation() {
	s.evalMu.Lock()
	defer s.evalMu.Unlock()
	select {
	case <-s.evalStop:
		return
	default:
		close(s.evalStop)
	}
	s.evalWg.Wait()
	s.logger.Info("alert evaluation pipeline stopped")
}

// evalLoop 评估循环
func (s *AlertService) evalLoop(ctx context.Context) {
	defer s.evalWg.Done()
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("alert evaluation context cancelled")
			return
		case <-s.evalStop:
			s.logger.Info("alert evaluation pipeline stopping")
			return
		case req, ok := <-s.evalCh:
			if !ok {
				s.logger.Info("alert evaluation channel closed")
				return
			}
			result := s.doEvaluateRule(ctx, req.rule)
			req.resultCh <- result
		}
	}
}

// doEvaluateRule 执行规则评估
func (s *AlertService) doEvaluateRule(ctx context.Context, rule *model.AlertRule) *evalResult {
	if rule == nil {
		return &evalResult{err: model.NewValidationError("rule", "rule cannot be nil")}
	}
	if !rule.Enabled {
		return &evalResult{}
	}

	// 模拟规则评估处理时间（扫描日志窗口）
	processStart := time.Now()

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
		return &evalResult{err: err}
	}

	// 模拟规则评估后处理耗时（聚合、阈值计算等）
	elapsed := time.Since(processStart)
	if elapsed < 50*time.Millisecond {
		time.Sleep(50*time.Millisecond - elapsed)
	}

	if result.Total >= rule.Threshold {
		event := model.NewAlertEvent(rule, result.Total)
		if err := s.alertStore.Store(event); err != nil {
			s.logger.Errorf("failed to store alert event: %v", err)
			return &evalResult{err: err}
		}
		s.logger.Warnf("alert triggered: rule=%s, count=%d, threshold=%d", rule.Name, result.Total, rule.Threshold)
		return &evalResult{event: event}
	}

	return &evalResult{}
}

// EvaluateRule 评估规则是否触发
func (s *AlertService) EvaluateRule(ctx context.Context, rule *model.AlertRule) (*model.AlertEvent, error) {
	if rule == nil {
		return nil, model.NewValidationError("rule", "rule cannot be nil")
	}
	if !rule.Enabled {
		return nil, nil
	}

	s.evalMu.Lock()
	select {
	case <-s.evalStop:
		s.evalMu.Unlock()
		return nil, model.NewValidationError("alert_service", "evaluation pipeline is stopped")
	default:
	}
	s.evalMu.Unlock()

	resultCh := make(chan *evalResult)
	req := &evalRequest{
		rule:     rule,
		resultCh: resultCh,
	}

	select {
	case s.evalCh <- req:
	case <-ctx.Done():
		// BUG: channel not closed when context cancelled
		return nil, ctx.Err()
	}

	select {
	case result := <-resultCh:
		return result.event, result.err
	case <-ctx.Done():
		// BUG: resultCh not closed here, producer goroutine will block forever
		return nil, ctx.Err()
	}
}

// EvaluateAllRules 评估所有规则
func (s *AlertService) EvaluateAllRules(ctx context.Context) ([]*model.AlertEvent, error) {
	rules, err := s.ruleStore.List()
	if err != nil {
		return nil, err
	}

	var triggeredEvents []*model.AlertEvent
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		wg.Add(1)
		go func(r *model.AlertRule) {
			defer wg.Done()
			event, err := s.EvaluateRule(ctx, r)
			if err != nil {
				s.logger.Errorf("failed to evaluate rule %s: %v", r.Name, err)
				return
			}
			if event != nil {
				mu.Lock()
				triggeredEvents = append(triggeredEvents, event)
				mu.Unlock()
			}
		}(rule)
	}
	wg.Wait()

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
