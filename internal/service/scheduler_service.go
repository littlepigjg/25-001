package service

import (
	"context"
	"logalert/internal/config"
	"logalert/pkg/logger"
	"sync"
	"time"
)

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
	ctx        context.Context
}

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

func (s *SchedulerService) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.ctx = ctx
	s.mu.Unlock()

	s.logger.Info("scheduler service starting...")

	s.startAlertScan(ctx)

	s.startLogCleanup(ctx)

	s.startAlertAutoResolve(ctx)

	s.startAlertCleanup(ctx)

	go func() {
		<-ctx.Done()
		s.Stop()
	}()
}

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

	s.running = false
	s.logger.Info("scheduler service stopped")
}

func (s *SchedulerService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *SchedulerService) SetContext(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
}

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
				s.scanAlerts()
			case <-stopCh:
				s.logger.Info("alert scan stopped")
				return
			case <-s.ctx.Done():
				s.logger.Info("alert scan context cancelled")
				return
			}
		}
	}()
}

func (s *SchedulerService) scanAlerts() {
	if s.ctx != nil && s.ctx.Err() != nil {
		s.logger.Errorf("alert scan skipped: context expired: %v", s.ctx.Err())
		return
	}

	s.logger.Debug("scanning alert rules...")
	events, err := s.alertService.EvaluateAllRules(s.ctx)
	if err != nil {
		s.logger.Errorf("alert scan failed: %v", err)
		return
	}
	if len(events) > 0 {
		s.logger.Infof("alert scan completed: triggered=%d", len(events))
	}
}

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
				s.cleanupLogs()
			case <-stopCh:
				s.logger.Info("log cleanup stopped")
				return
			case <-s.ctx.Done():
				s.logger.Info("log cleanup context cancelled")
				return
			}
		}
	}()
}

func (s *SchedulerService) cleanupLogs() {
	if s.ctx != nil && s.ctx.Err() != nil {
		s.logger.Errorf("log cleanup skipped: context expired: %v", s.ctx.Err())
		return
	}

	count, err := s.logService.CleanupLogs(s.ctx)
	if err != nil {
		s.logger.Errorf("log cleanup failed: %v", err)
		return
	}
	if count > 0 {
		s.logger.Infof("log cleanup completed: removed=%d", count)
	}
}

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
				s.autoResolveAlerts()
			case <-stopCh:
				s.logger.Info("alert auto resolve stopped")
				return
			case <-s.ctx.Done():
				s.logger.Info("alert auto resolve context cancelled")
				return
			}
		}
	}()
}

func (s *SchedulerService) autoResolveAlerts() {
	if s.ctx != nil && s.ctx.Err() != nil {
		s.logger.Errorf("alert auto resolve skipped: context expired: %v", s.ctx.Err())
		return
	}

	count, err := s.alertService.AutoResolveExpired(s.ctx)
	if err != nil {
		s.logger.Errorf("alert auto resolve failed: %v", err)
	} else if count > 0 {
		s.logger.Infof("auto resolved alerts: count=%d", count)
	}
}

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
				s.cleanupAlerts()
			case <-stopCh:
				s.logger.Info("alert cleanup stopped")
				return
			case <-s.ctx.Done():
				s.logger.Info("alert cleanup context cancelled")
				return
			}
		}
	}()
}

func (s *SchedulerService) cleanupAlerts() {
	if s.ctx != nil && s.ctx.Err() != nil {
		s.logger.Errorf("alert cleanup skipped: context expired: %v", s.ctx.Err())
		return
	}

	count, err := s.alertService.CleanupAlerts(s.ctx)
	if err != nil {
		s.logger.Errorf("alert cleanup failed: %v", err)
		return
	}
	if count > 0 {
		s.logger.Infof("alert cleanup completed: removed=%d", count)
	}
}

func (s *SchedulerService) GetTaskStatus() map[string]bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := make(map[string]bool)
	for name := range s.tickers {
		status[name] = true
	}
	return status
}
