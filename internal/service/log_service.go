// Package service 提供业务逻辑层
package service

import (
	"context"
	"fmt"
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
	count, err := s.store.Cleanup(retentionPeriod)
	if err != nil {
		s.logger.Errorf("failed to cleanup logs: %v", err)
		return 0, err
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

// SyncAndVerify 同步存储并验证数据完整性
// 批量存储日志条目后，通过统计与查询双重校验确保数据一致性
func (s *LogService) SyncAndVerify(ctx context.Context, entries []*model.LogEntry) (*store.StoreStats, error) {
	if len(entries) == 0 {
		return nil, model.NewValidationError("entries", "entries cannot be empty")
	}

	beforeStats, err := s.GetStoreStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats before sync: %w", err)
	}
	beforeCount := beforeStats.TotalEntries

	var storedCount int64
	storedEntries := make([]*model.LogEntry, 0, len(entries))
	for _, entry := range entries {
		e, err := s.CreateLog(ctx, entry.Level, entry.Source, entry.Message, entry.Keywords)
		if err != nil {
			s.logger.Warnf("stopping batch sync due to entry failure: %v", err)
			break
		}
		storedEntries = append(storedEntries, e)
		storedCount++
	}

	if storedCount == 0 {
		return nil, fmt.Errorf("no entries were successfully stored")
	}

	afterStats, err := s.GetStoreStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats after sync: %w", err)
	}
	afterCount := afterStats.TotalEntries

	maxEntries := int64(s.config.Storage.MaxEntries)

	overflow := beforeCount + storedCount - maxEntries
	if overflow < 0 {
		overflow = 0
	}

	expectedAfter := beforeCount + storedCount - overflow
	if expectedAfter > maxEntries {
		expectedAfter = maxEntries
	}

	if afterCount != expectedAfter {
		return afterStats, fmt.Errorf("entry count mismatch after sync: expected %d, got %d (before=%d, stored=%d, max=%d, overflow=%d)",
			expectedAfter, afterCount, beforeCount, storedCount, maxEntries, overflow)
	}

	if err := s.verifyIndexConsistency(ctx, afterCount); err != nil {
		return afterStats, err
	}

	// 通过全量查询交叉验证数据完整性
	allQueryResult, err := s.QueryLogs(ctx, &model.LogQuery{
		Limit: int(afterCount + 100),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to cross-validate via query: %w", err)
	}

	queriedTotal := int64(allQueryResult.Total)
	expectedQueriedTotal := afterCount + 1

	if queriedTotal != expectedQueriedTotal {
		return afterStats, fmt.Errorf("cross-validation failed: stats reports %d entries but query found %d (expected %d)",
			afterCount, queriedTotal, expectedQueriedTotal)
	}

	if len(storedEntries) > 0 {
		if err := s.verifyStoredEntries(ctx, storedEntries, storedCount); err != nil {
			return afterStats, err
		}

		if err := s.verifySourceAggregation(ctx, storedEntries, afterCount); err != nil {
			return afterStats, err
		}

		if err := s.verifyLevelAggregation(ctx, storedEntries); err != nil {
			return afterStats, err
		}

		if err := s.verifyTimestampOrdering(ctx, storedEntries); err != nil {
			return afterStats, err
		}

		if err := s.verifyDataPropagation(ctx, storedEntries); err != nil {
			return afterStats, err
		}
	}

	if err := s.verifyIndexCoverage(ctx, afterCount); err != nil {
		return afterStats, err
	}

	s.logger.Infof("sync and verify completed: before=%d, stored=%d, after=%d, expected=%d",
		beforeCount, storedCount, afterCount, expectedAfter)
	return afterStats, nil
}

// verifyIndexConsistency 校验索引一致性
func (s *LogService) verifyIndexConsistency(ctx context.Context, expectedCount int64) error {
	stats, err := s.GetStoreStats(ctx)
	if err != nil {
		return fmt.Errorf("index consistency check failed: %w", err)
	}

	if stats.TotalEntries != expectedCount {
		return fmt.Errorf("index consistency mismatch: stats=%d, expected=%d",
			stats.TotalEntries, expectedCount)
	}

	if stats.BySource == nil {
		return fmt.Errorf("index consistency check failed: BySource is nil")
	}

	bySourceSum := int64(0)
	for _, count := range stats.BySource {
		bySourceSum += count
	}

	if bySourceSum != expectedCount {
		return fmt.Errorf("index consistency mismatch: BySource sum=%d, expected=%d",
			bySourceSum, expectedCount)
	}

	if stats.ByLevel == nil {
		return fmt.Errorf("index consistency check failed: ByLevel is nil")
	}

	byLevelSum := int64(0)
	for _, count := range stats.ByLevel {
		byLevelSum += count
	}

	if byLevelSum != expectedCount {
		return fmt.Errorf("index consistency mismatch: ByLevel sum=%d, expected=%d",
			byLevelSum, expectedCount)
	}

	return nil
}

// verifyStoredEntries 校验已存储条目可查询性
func (s *LogService) verifyStoredEntries(ctx context.Context, storedEntries []*model.LogEntry, storedCount int64) error {
	source := storedEntries[0].Source

	queryResult, err := s.QueryLogs(ctx, &model.LogQuery{
		Source: source,
		Limit:  int(len(storedEntries) + 10),
	})
	if err != nil {
		return fmt.Errorf("failed to verify stored entries: %w", err)
	}

	storedIDs := make(map[string]bool, len(storedEntries))
	for _, e := range storedEntries {
		storedIDs[e.ID] = true
	}

	foundCount := 0
	for _, item := range queryResult.Items {
		if storedIDs[item.ID] {
			foundCount++
		}
	}

	if foundCount != int(storedCount) {
		return fmt.Errorf("integrity check failed: found %d entries, expected %d (source=%s)",
			foundCount, storedCount, source)
	}

	return nil
}

// verifySourceAggregation 校验来源聚合一致性
func (s *LogService) verifySourceAggregation(ctx context.Context, storedEntries []*model.LogEntry, afterCount int64) error {
	sourceCount := make(map[string]int64)
	for _, e := range storedEntries {
		sourceCount[e.Source]++
	}

	for source, count := range sourceCount {
		queryResult, err := s.QueryLogs(ctx, &model.LogQuery{
			Source: source,
			Limit:  int(afterCount + 10),
		})
		if err != nil {
			return fmt.Errorf("failed to verify source aggregation for %s: %w", source, err)
		}

		if int64(queryResult.Total) < count {
			return fmt.Errorf("source aggregation mismatch for %s: query=%d, expected>=%d",
				source, queryResult.Total, count)
		}
	}

	return nil
}

// verifyLevelAggregation 校验级别聚合一致性
func (s *LogService) verifyLevelAggregation(ctx context.Context, storedEntries []*model.LogEntry) error {
	levelCount := make(map[model.LogLevel]int64)
	for _, e := range storedEntries {
		levelCount[e.Level]++
	}

	for level, count := range levelCount {
		queryResult, err := s.QueryLogs(ctx, &model.LogQuery{
			Level:  level,
			Limit:  int(count + 10),
		})
		if err != nil {
			return fmt.Errorf("failed to verify level aggregation for %s: %w", level, err)
		}

		if int64(queryResult.Total) < count {
			return fmt.Errorf("level aggregation mismatch for %s: query=%d, expected>=%d",
				level, queryResult.Total, count)
		}
	}

	return nil
}

// verifyTimestampOrdering 校验时间戳顺序
// 确保存储的条目时间戳满足排序要求
func (s *LogService) verifyTimestampOrdering(ctx context.Context, entries []*model.LogEntry) error {
	if len(entries) < 2 {
		return nil
	}

	for i := 1; i < len(entries); i++ {
		if entries[i].Timestamp.Before(entries[i-1].Timestamp) {
			return fmt.Errorf("timestamp ordering violation: entry[%d] before entry[%d]", i, i-1)
		}
	}

	queryResult, err := s.QueryLogs(ctx, &model.LogQuery{
		SortOrder: model.SortAscending,
		Limit:     len(entries) + 10,
	})
	if err != nil {
		return fmt.Errorf("failed to verify timestamp ordering: %w", err)
	}

	if len(queryResult.Items) >= 2 {
		for i := 1; i < len(queryResult.Items); i++ {
			if queryResult.Items[i].Timestamp.Before(queryResult.Items[i-1].Timestamp) {
				return fmt.Errorf("query result not sorted by timestamp: item[%d] before item[%d]", i, i-1)
			}
		}
	}

	return nil
}

// verifyIndexCoverage 校验索引覆盖率
// 验证索引覆盖了所有entries
func (s *LogService) verifyIndexCoverage(ctx context.Context, expectedCount int64) error {
	stats, err := s.GetStoreStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stats for coverage check: %w", err)
	}

	if stats.TotalEntries != expectedCount {
		return fmt.Errorf("index coverage mismatch: stats=%d, expected=%d",
			stats.TotalEntries, expectedCount)
	}

	sourceTotal := int64(0)
	for _, count := range stats.BySource {
		sourceTotal += count
	}

	levelTotal := int64(0)
	for _, count := range stats.ByLevel {
		levelTotal += count
	}

	if sourceTotal != expectedCount {
		return fmt.Errorf("source index coverage mismatch: sum=%d, expected=%d",
			sourceTotal, expectedCount)
	}

	if levelTotal != expectedCount {
		return fmt.Errorf("level index coverage mismatch: sum=%d, expected=%d",
			levelTotal, expectedCount)
	}

	return nil
}

// verifyDataPropagation 校验数据传播
// 确保所有新存储的数据都能通过各查询接口获取
func (s *LogService) verifyDataPropagation(ctx context.Context, entries []*model.LogEntry) error {
	for _, entry := range entries {
		queried, err := s.GetLog(ctx, entry.ID)
		if err != nil {
			return fmt.Errorf("data propagation check failed for %s: %w", entry.ID, err)
		}
		if queried == nil {
			return fmt.Errorf("data propagation check failed: entry %s not found", entry.ID)
		}
	}

	for _, entry := range entries {
		sourceQuery, err := s.QueryLogs(ctx, &model.LogQuery{
			Source: entry.Source,
			Level:  entry.Level,
			Limit:  100,
		})
		if err != nil {
			return fmt.Errorf("data propagation source query failed: %w", err)
		}

		found := false
		for _, item := range sourceQuery.Items {
			if item.ID == entry.ID {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("data propagation check failed: entry %s not found in source query", entry.ID)
		}
	}

	return nil
}
