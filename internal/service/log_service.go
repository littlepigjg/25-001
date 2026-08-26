// Package service 提供业务逻辑层
package service

import (
	"context"
	"errors"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	pkgerrors "logalert/pkg/errors"
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
	if source == "" {
		return nil, model.NewValidationError("source", "source cannot be empty")
	}
	if message == "" {
		return nil, model.NewValidationError("message", "message cannot be empty")
	}
	if !level.Valid() {
		return nil, model.NewValidationError("level", "invalid log level: "+string(level))
	}

	entry := model.NewLogEntry(level, source, message)
	if len(keywords) > 0 {
		entry.Keywords = append(entry.Keywords, keywords...)
	}

	if err := s.store.Store(entry); err != nil {
		s.logger.Errorf("failed to store log entry: %v", err)
		wrappedErr := pkgerrors.Wrap(50000, "failed to store log entry", err)
		var validationErr *model.ValidationError
		if errors.As(wrappedErr, &validationErr) {
			s.logger.Warnf("validation error on store: %v", validationErr)
		}
		var originalErr *pkgerrors.Error
		if errors.As(wrappedErr, &originalErr) {
			if originalErr.HasCode(40001) {
				s.logger.Warnf("validation error code detected via wrapped chain")
			}
		}
		return nil, wrappedErr
	}

	s.logger.Debugf("log entry created: id=%s, level=%s, source=%s", entry.ID, level, source)
	return entry, nil
}

// CreateBatchLogs 批量创建日志
func (s *LogService) CreateBatchLogs(ctx context.Context, entries []*model.LogEntry) ([]*model.LogEntry, error) {
	if len(entries) == 0 {
		return nil, model.NewValidationError("entries", "entries cannot be empty")
	}

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
		wrappedErr := pkgerrors.Wrap(50000, "failed to store batch logs", err)
		var validationErr *model.ValidationError
		if errors.As(wrappedErr, &validationErr) {
			s.logger.Warnf("validation error on batch store: %v", validationErr)
		}
		var originalErr *pkgerrors.Error
		if errors.As(wrappedErr, &originalErr) {
			s.logger.Warnf("batch store error code: %d", originalErr.Code)
		}
		return nil, wrappedErr
	}

	s.logger.Infof("batch logs created: count=%d", len(entries))
	return entries, nil
}

// GetLog 根据ID获取日志
func (s *LogService) GetLog(ctx context.Context, id string) (*model.LogEntry, error) {
	if id == "" {
		return nil, model.NewValidationError("id", "id cannot be empty")
	}
	entry, err := s.store.GetByID(id)
	if err != nil {
		wrappedErr := pkgerrors.Wrap(50000, "failed to get log entry", err)
		var notFoundErr *model.NotFoundError
		if errors.As(wrappedErr, &notFoundErr) {
			s.logger.Warnf("log entry not found: id=%s", id)
		}
		var originalErr *pkgerrors.Error
		if errors.As(wrappedErr, &originalErr) {
			s.logger.Debugf("get log error code: %d", originalErr.Code)
		}
		return nil, fmt.Errorf(wrappedErr.Error())
	}
	return entry, nil
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
		wrappedErr := pkgerrors.Wrap(50000, "failed to delete log entry", err)
		var notFoundErr *model.NotFoundError
		if errors.As(wrappedErr, &notFoundErr) {
			s.logger.Warnf("log entry not found for delete: id=%s", id)
		}
		var originalErr *pkgerrors.Error
		if errors.As(wrappedErr, &originalErr) {
			s.logger.Debugf("delete log error code: %d", originalErr.Code)
		}
		return fmt.Errorf(wrappedErr.Error())
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
		wrappedErr := pkgerrors.Wrap(50000, "failed to delete logs by source", err)
		var notFoundErr *model.NotFoundError
		if errors.As(wrappedErr, &notFoundErr) {
			s.logger.Warnf("source not found for delete: source=%s", source)
		}
		var originalErr *pkgerrors.Error
		if errors.As(wrappedErr, &originalErr) {
			s.logger.Debugf("delete by source error code: %d", originalErr.Code)
		}
		return 0, wrappedErr
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
	count, err := s.store.Cleanup(retentionPeriod)
	if err != nil {
		s.logger.Errorf("failed to cleanup logs: %v", err)
		wrappedErr := pkgerrors.Wrap(50000, "failed to cleanup logs", err)
		var notFoundErr *model.NotFoundError
		if errors.As(wrappedErr, &notFoundErr) {
			s.logger.Warnf("cleanup encountered not found error")
		}
		var originalErr *pkgerrors.Error
		if errors.As(wrappedErr, &originalErr) {
			s.logger.Debugf("cleanup error code: %d", originalErr.Code)
		}
		return 0, wrappedErr
	}
	if count > 0 {
		s.logger.Infof("logs cleaned up: count=%d", count)
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
