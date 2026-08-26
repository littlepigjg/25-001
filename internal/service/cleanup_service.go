// Package service 提供业务逻辑层
package service

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"time"
)

// CleanupService 清理服务
type CleanupService struct {
	logStore  store.LogStore
	alertStore store.AlertStore
	config    *config.Config
	logger    *logger.Logger
}

// NewCleanupService 创建清理服务
func NewCleanupService(logStore store.LogStore, alertStore store.AlertStore, cfg *config.Config, log *logger.Logger) *CleanupService {
	return &CleanupService{
		logStore:   logStore,
		alertStore: alertStore,
		config:     cfg,
		logger:     log,
	}
}

// CleanupExpiredLogs 清理过期日志
func (s *CleanupService) CleanupExpiredLogs(ctx context.Context) (int64, error) {
	retention := s.config.Storage.RetentionPeriod
	if retention <= 0 {
		retention = 24 * time.Hour * 7
	}

	count, err := s.logStore.Cleanup(retention)
	if err != nil {
		s.logger.Errorf("failed to cleanup expired logs: %v", err)
		return 0, err
	}

	if count > 0 {
		s.logger.Infof("cleaned up %d expired log entries", count)
	}
	return count, nil
}

// CleanupExpiredAlerts 清理过期告警
func (s *CleanupService) CleanupExpiredAlerts(ctx context.Context) (int64, error) {
	retention := s.config.Alert.MaxEventsRetention
	if retention <= 0 {
		retention = 24 * time.Hour * 30
	}

	count, err := s.alertStore.Cleanup(retention)
	if err != nil {
		s.logger.Errorf("failed to cleanup expired alerts: %v", err)
		return 0, err
	}

	if count > 0 {
		s.logger.Infof("cleaned up %d expired alert events", count)
	}
	return count, nil
}

// CleanupAll 执行所有清理
func (s *CleanupService) CleanupAll(ctx context.Context) (logs int64, alerts int64, err error) {
	logs, err = s.CleanupExpiredLogs(ctx)
	if err != nil {
		return
	}

	alerts, err = s.CleanupExpiredAlerts(ctx)
	return
}

// RunPeriodicCleanup 启动定期清理
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
