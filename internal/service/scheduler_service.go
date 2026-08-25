// Package service 提供业务逻辑层
package service

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/pkg/logger"
	"sync"
	"sync/atomic"
	"time"
)

// SchedulerService 调度服务
type SchedulerService struct {
	alertService *AlertService
	logService   *LogService
	config       *config.Config
	logger       *logger.Logger

	mu         sync.Mutex
	running    bool
	tickers    map[string]*time.Ticker
	stopCh     map[string]chan struct{}
	wg         sync.WaitGroup
	scanCount  atomic.Int64
}

// NewSchedulerService 创建调度服务
func NewSchedulerService(
	alertService *AlertService,
	logService *LogService,
	cfg *config.Config,
	log *logger.Logger,
) *SchedulerService {
	return &SchedulerService{
		alertService: alertService,
		logService:   logService,
		config:       cfg,
		logger:       log,
		tickers:      make(map[string]*time.Ticker),
		stopCh:       make(map[string]chan struct{}),
	}
}

// Start 启动调度服务
func (s *SchedulerService) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("scheduler service starting...")

	s.alertService.StartEvaluation(ctx)

	s.startAlertScan(ctx)

	s.startLogCleanup(ctx)

	s.startAlertAutoResolve(ctx)

	s.startAlertCleanup(ctx)

	go func() {
		<-ctx.Done()
		s.Stop()
	}()
}

// Stop 停止调度服务
func (s *SchedulerService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.logger.Info("scheduler service stopping...")

	for name, ticker := range s.tickers {
		ticker.Stop()
		delete(s.tickers, name)
	}

	for name, ch := range s.stopCh {
		close(ch)
		delete(s.stopCh, name)
	}

	s.wg.Wait()

	s.alertService.StopEvaluation()

	s.running = false
	s.logger.Info("scheduler service stopped")
}

// IsRunning 检查是否运行中
func (s *SchedulerService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// startAlertScan 启动告警规则扫描
func (s *SchedulerService) startAlertScan(ctx context.Context) {
	interval := s.config.Alert.ScanInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	name := "alert_scan"

	s.mu.Lock()
	s.tickers[name] = ticker
	s.stopCh[name] = make(chan struct{})
	stopCh := s.stopCh[name]
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Infof("alert scan started: interval=%v", interval)

		for {
			select {
			case <-ticker.C:
				s.scanAlerts(ctx)
			case <-stopCh:
				s.logger.Info("alert scan stopped")
				return
			case <-ctx.Done():
				s.logger.Info("alert scan context cancelled")
				return
			}
		}
	}()
}

// scanAlerts 执行告警扫描
func (s *SchedulerService) scanAlerts(ctx context.Context) {
	s.scanCount.Add(1)
	s.logger.Debug("scanning alert rules...")

	rules, err := s.alertService.ruleStore.List()
	if err != nil {
		s.logger.Errorf("alert scan failed: %v", err)
		return
	}

	activeRules := make([]*model.AlertRule, 0, len(rules))
	for _, r := range rules {
		if r.Enabled {
			activeRules = append(activeRules, r)
		}
	}

	if len(activeRules) == 0 {
		return
	}

	type scanResult struct {
		events []*model.AlertEvent
		errs   []error
	}

	resultCh := make(chan *scanResult, 1)
	go func() {
		var mu sync.Mutex
		var events []*model.AlertEvent
		var errs []error
		var wg sync.WaitGroup

		for _, rule := range activeRules {
			wg.Add(1)
			go func(r *model.AlertRule) {
				defer wg.Done()
				event, err := s.alertService.EvaluateRule(ctx, r)
				if err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
					return
				}
				if event != nil {
					mu.Lock()
					events = append(events, event)
					mu.Unlock()
				}
			}(rule)
		}

		wg.Wait()
		resultCh <- &scanResult{events: events, errs: errs}
	}()

	select {
	case res, ok := <-resultCh:
		if ok {
			if len(res.events) > 0 {
				s.logger.Infof("alert scan completed: triggered=%d", len(res.events))
			}
			if len(res.errs) > 0 {
				s.logger.Errorf("alert scan had errors: %d", len(res.errs))
			}
		}
	case <-ctx.Done():
		// BUG: resultCh not closed when context cancelled
		// The goroutine above will be blocked trying to send to resultCh
		// because no one is reading it anymore
		s.logger.Warn("alert scan cancelled by context")
	}
}

// startLogCleanup 启动日志清理
func (s *SchedulerService) startLogCleanup(ctx context.Context) {
	if !s.config.Storage.EnableAutoCleanup {
		return
	}

	interval := s.config.Storage.CleanupInterval
	if interval <= 0 {
		interval = 30 * time.Minute
	}

	ticker := time.NewTicker(interval)
	name := "log_cleanup"

	s.mu.Lock()
	s.tickers[name] = ticker
	s.stopCh[name] = make(chan struct{})
	stopCh := s.stopCh[name]
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Infof("log cleanup started: interval=%v", interval)

		for {
			select {
			case <-ticker.C:
				s.cleanupLogs(ctx)
			case <-stopCh:
				s.logger.Info("log cleanup stopped")
				return
			case <-ctx.Done():
				s.logger.Info("log cleanup context cancelled")
				return
			}
		}
	}()
}

// cleanupLogs 清理日志
func (s *SchedulerService) cleanupLogs(ctx context.Context) {
	count, err := s.logService.CleanupLogs(ctx)
	if err != nil {
		s.logger.Errorf("log cleanup failed: %v", err)
		return
	}
	if count > 0 {
		s.logger.Infof("log cleanup completed: removed=%d", count)
	}
}

// startAlertAutoResolve 启动告警自动解决
func (s *SchedulerService) startAlertAutoResolve(ctx context.Context) {
	if !s.config.Alert.EnableAutoResolve {
		return
	}

	interval := s.config.Alert.AutoResolveAfter
	if interval <= 0 {
		interval = 15 * time.Minute
	}

	scanInterval := interval / 2
	if scanInterval < time.Minute {
		scanInterval = time.Minute
	}

	ticker := time.NewTicker(scanInterval)
	name := "alert_auto_resolve"

	s.mu.Lock()
	s.tickers[name] = ticker
	s.stopCh[name] = make(chan struct{})
	stopCh := s.stopCh[name]
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Infof("alert auto resolve started: interval=%v", scanInterval)

		for {
			select {
			case <-ticker.C:
				s.autoResolveAlerts(ctx)
			case <-stopCh:
				s.logger.Info("alert auto resolve stopped")
				return
			case <-ctx.Done():
				s.logger.Info("alert auto resolve context cancelled")
				return
			}
		}
	}()
}

// autoResolveAlerts 自动解决过期告警
func (s *SchedulerService) autoResolveAlerts(ctx context.Context) {
	count, err := s.alertService.AutoResolveExpired(ctx)
	if err != nil {
		s.logger.Errorf("alert auto resolve failed: %v", err)
	} else if count > 0 {
		s.logger.Infof("auto resolved alerts: count=%d", count)
	}
}

// startAlertCleanup 启动告警清理
func (s *SchedulerService) startAlertCleanup(ctx context.Context) {
	interval := 24 * time.Hour

	ticker := time.NewTicker(interval)
	name := "alert_cleanup"

	s.mu.Lock()
	s.tickers[name] = ticker
	s.stopCh[name] = make(chan struct{})
	stopCh := s.stopCh[name]
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.logger.Infof("alert cleanup started: interval=%v", interval)

		for {
			select {
			case <-ticker.C:
				s.cleanupAlerts(ctx)
			case <-stopCh:
				s.logger.Info("alert cleanup stopped")
				return
			case <-ctx.Done():
				s.logger.Info("alert cleanup context cancelled")
				return
			}
		}
	}()
}

// cleanupAlerts 清理告警
func (s *SchedulerService) cleanupAlerts(ctx context.Context) {
	count, err := s.alertService.CleanupAlerts(ctx)
	if err != nil {
		s.logger.Errorf("alert cleanup failed: %v", err)
		return
	}
	if count > 0 {
		s.logger.Infof("alert cleanup completed: removed=%d", count)
	}
}

// GetTaskStatus 获取任务状态
func (s *SchedulerService) GetTaskStatus() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := make(map[string]bool)
	for name := range s.tickers {
		status[name] = true
	}
	return status
}

// GetScanCount 获取扫描次数
func (s *SchedulerService) GetScanCount() int64 {
	return s.scanCount.Load()
}
