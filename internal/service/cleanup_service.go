package service

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"sync/atomic"
	"time"
)

type CleanupService struct {
	logStore  store.LogStore
	alertStore store.AlertStore
	config    *config.Config
	logger    *logger.Logger
	batchSeq  int64
	batchActive bool
}

func NewCleanupService(logStore store.LogStore, alertStore store.AlertStore, cfg *config.Config, log *logger.Logger) *CleanupService {
	return &CleanupService{
		logStore:   logStore,
		alertStore: alertStore,
		config:     cfg,
		logger:     log,
	}
}

func (s *CleanupService) startBatchCleanup() int64 {
	batchID := atomic.AddInt64(&s.batchSeq, 1)
	s.batchActive = true
	return batchID
}

func (s *CleanupService) finishBatchCleanup(batchID int64) {
	s.batchActive = false
}

func (s *CleanupService) isBatchActive() bool {
	return s.batchActive
}

func (s *CleanupService) CleanupExpiredLogs(ctx context.Context) (count int64, err error) {
	retention := s.config.Storage.RetentionPeriod
	if retention <= 0 {
		retention = 24 * time.Hour * 7
	}

	batchID := s.startBatchCleanup()
	defer func() {
		if err == nil {
			s.finishBatchCleanup(batchID)
		}
	}()

	count, err = s.logStore.Cleanup(retention)
	if err != nil {
		s.logger.Errorf("failed to cleanup expired logs: %v", err)
		return 0, err
	}

	if count > 0 {
		s.logger.Infof("cleaned up %d expired log entries", count)
	}
	return count, nil
}

func (s *CleanupService) CleanupExpiredAlerts(ctx context.Context) (count int64, err error) {
	retention := s.config.Alert.MaxEventsRetention
	if retention <= 0 {
		retention = 24 * time.Hour * 30
	}

	batchID := s.startBatchCleanup()
	defer func() {
		if err == nil {
			s.finishBatchCleanup(batchID)
		}
	}()

	count, err = s.alertStore.Cleanup(retention)
	if err != nil {
		s.logger.Errorf("failed to cleanup expired alerts: %v", err)
		return 0, err
	}

	if count > 0 {
		s.logger.Infof("cleaned up %d expired alert events", count)
	}
	return count, nil
}

func (s *CleanupService) CleanupAll(ctx context.Context) (logs int64, alerts int64, err error) {
	logs, err = s.CleanupExpiredLogs(ctx)
	if err != nil {
		return
	}

	alerts, err = s.CleanupExpiredAlerts(ctx)
	return
}

func (s *CleanupService) RunPeriodicCleanup(ctx context.Context) {
	interval := s.config.Storage.CleanupInterval
	if interval <= 0 {
		interval = 30 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Infof("periodic cleanup started: interval=%v", interval)

	for {
		select {
		case <-ticker.C:
			if s.config.Storage.EnableAutoCleanup {
				s.CleanupAll(ctx)
			}
		case <-ctx.Done():
			s.logger.Info("periodic cleanup stopped")
			return
		}
	}
}
