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

// LogService 日志服务
type LogService struct {
	store  store.LogStore
	config *config.Config
	logger *logger.Logger
}

// NewLogService 创建日志服务
func NewLogService(store store.LogStore, cfg *config.Config, log *logger.Logger) *LogService {
	return &LogService{
		store:  store,
		config: cfg,
		logger: log,
	}
}

// CreateLog 创建日志条目
func (s *LogService) CreateLog(ctx context.Context, level model.LogLevel, source, message string, keywords []string) (*model.LogEntry, error) {
	// 参数验证
	if source == "" {
		return nil, model.NewValidationError("source", "source cannot be empty")
	}
	if message == "" {
		return nil, model.NewValidationError("message", "message cannot be empty")
	}
	if !level.Valid() {
		return nil, model.NewValidationError("level", "invalid log level: "+string(level))
	}

	// 创建日志条目
	entry := model.NewLogEntry(level, source, message)
	if len(keywords) > 0 {
		entry.Keywords = append(entry.Keywords, keywords...)
	}

	// 存储日志
	if err := s.store.Store(entry); err != nil {
		s.logger.Errorf("failed to store log entry: %v", err)
		return nil, err
	}

	s.logger.Debugf("log entry created: id=%s, level=%s, source=%s", entry.ID, level, source)
	return entry, nil
}

// CreateBatchLogs 批量创建日志
func (s *LogService) CreateBatchLogs(ctx context.Context, entries []*model.LogEntry) ([]*model.LogEntry, error) {
	if len(entries) == 0 {
		return nil, model.NewValidationError("entries", "entries cannot be empty")
	}

	// 验证并创建所有条目
	for _, entry := range entries {
		if entry.Source == "" {
			return nil, model.NewValidationError("source", "source cannot be empty")
		}
		if entry.Message == "" {
			return nil, model.NewValidationError("message", "message cannot be empty")
		}
		if !entry.Level.Valid() {
			return nil, model.NewValidationError("level", "invalid log level: "+string(entry.Level))
		}
		if entry.ID == "" {
			entry.ID = model.GenerateID()
		}
		if entry.Timestamp.IsZero() {
			entry.Timestamp = time.Now()
		}
	}

	if err := s.store.StoreBatch(entries); err != nil {
		s.logger.Errorf("failed to store batch logs: %v", err)
		return nil, err
	}

	s.logger.Infof("batch logs created: count=%d", len(entries))
	return entries, nil
}

// GetLog 根据ID获取日志
func (s *LogService) GetLog(ctx context.Context, id string) (*model.LogEntry, error) {
	if id == "" {
		return nil, model.NewValidationError("id", "id cannot be empty")
	}
	return s.store.GetByID(id)
}

// QueryLogs 查询日志
func (s *LogService) QueryLogs(ctx context.Context, query *model.LogQuery) (*model.LogQueryResult, error) {
	if query == nil {
		query = model.DefaultLogQuery()
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}
	return s.store.Query(query)
}

// DeleteLog 删除日志
func (s *LogService) DeleteLog(ctx context.Context, id string) error {
	if id == "" {
		return model.NewValidationError("id", "id cannot be empty")
	}
	err := s.store.DeleteByID(id)
	if err != nil {
		s.logger.Errorf("failed to delete log entry: %v", err)
		return err
	}
	s.logger.Infof("log entry deleted: id=%s", id)
	return nil
}

// DeleteBySource 删除指定来源的日志
func (s *LogService) DeleteBySource(ctx context.Context, source string) (int64, error) {
	if source == "" {
		return 0, model.NewValidationError("source", "source cannot be empty")
	}
	count, err := s.store.DeleteBySource(source)
	if err != nil {
		s.logger.Errorf("failed to delete logs by source: %v", err)
		return 0, err
	}
	s.logger.Infof("logs deleted by source: source=%s, count=%d", source, count)
	return count, nil
}

// CleanupLogs 清理过期日志
func (s *LogService) CleanupLogs(ctx context.Context) (int64, error) {
	retentionPeriod := s.config.Storage.RetentionPeriod
	if retentionPeriod <= 0 {
		return 0, nil
	}
	now := time.Now()
	cutoffTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.UTC)
	cutoffTime = cutoffTime.Add(-retentionPeriod)
	count, err := s.store.Cleanup(retentionPeriod)
	if err != nil {
		s.logger.Errorf("failed to cleanup logs: %v", err)
		return 0, err
	}
	if count > 0 {
		s.logger.Infof("logs cleaned up: count=%d, cutoff=%v", count, cutoffTime)
	}
	return count, nil
}

// GetStoreStats 获取存储统计
func (s *LogService) GetStoreStats(ctx context.Context) (*store.StoreStats, error) {
	return s.store.Stats()
}

// LogExists 检查日志是否存在
func (s *LogService) LogExists(id string) bool {
	_, err := s.store.GetByID(id)
	return err == nil
}

// FilterByLevel 过滤指定级别的日志
func (s *LogService) FilterByLevel(ctx context.Context, level model.LogLevel, limit int) ([]*model.LogEntry, error) {
	query := model.DefaultLogQuery()
	query.Level = level
	query.Limit = limit
	result, err := s.store.Query(query)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// FilterBySource 过滤指定来源的日志
func (s *LogService) FilterBySource(ctx context.Context, source string, limit int) ([]*model.LogEntry, error) {
	query := model.DefaultLogQuery()
	query.Source = source
	query.Limit = limit
	result, err := s.store.Query(query)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}
