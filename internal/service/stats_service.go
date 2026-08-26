// Package service 提供业务逻辑层
package service

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"sort"
	"time"
)

// StatsService 统计服务
type StatsService struct {
	store  store.LogStore
	config *config.Config
	logger *logger.Logger
}

// NewStatsService 创建统计服务
func NewStatsService(store store.LogStore, cfg *config.Config, log *logger.Logger) *StatsService {
	return &StatsService{
		store:  store,
		config: cfg,
		logger: log,
	}
}

// GetStatistics 获取日志统计
func (s *StatsService) GetStatistics(ctx context.Context, startTime, endTime time.Time) (*model.LogStatistics, error) {
	if startTime.IsZero() {
		now := time.Now()
		startTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.UTC)
		startTime = startTime.Add(-24 * time.Hour)
	}
	if endTime.IsZero() {
		now := time.Now()
		endTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.UTC)
	}

	query := &model.LogQuery{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     100000,
	}

	result, err := s.store.Query(query)
	if err != nil {
		s.logger.Errorf("failed to query logs for stats: %v", err)
		return nil, err
	}

	stats := model.NewLogStatistics(startTime, endTime)

	sourceErrors := make(map[string]int64)
	sourceTotal := make(map[string]int64)

	for _, entry := range result.Items {
		stats.Add(entry)
		if entry.Level == model.LevelError || entry.Level == model.LevelFatal {
			sourceErrors[entry.Source]++
		}
		sourceTotal[entry.Source]++
	}

	for source, total := range sourceTotal {
		if total > 0 {
			stats.ErrorRate[source] = float64(sourceErrors[source]) / float64(total)
		} else {
			stats.ErrorRate[source] = 0
		}
	}

	return stats, nil
}

// GetHourlyTrends 获取每小时趋势
func (s *StatsService) GetHourlyTrends(ctx context.Context, hours int) ([]*model.HourlyTrend, error) {
	if hours <= 0 {
		hours = 24
	}

	now := time.Now()
	endTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.UTC)
	startTime := endTime.Add(-time.Duration(hours) * time.Hour)

	query := &model.LogQuery{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     100000,
	}

	result, err := s.store.Query(query)
	if err != nil {
		return nil, err
	}

	trends := make([]*model.HourlyTrend, 0, hours)
	for i := hours - 1; i >= 0; i-- {
		hour := endTime.Add(-time.Duration(i) * time.Hour)
		hourStart := time.Date(hour.Year(), hour.Month(), hour.Day(), hour.Hour(), 0, 0, 0, time.UTC)
		hourKey := hourStart.Format("2006-01-02T15")
		trends = append(trends, &model.HourlyTrend{
			Hour: hourKey,
		})
	}

	for _, entry := range result.Items {
		hourKey := entry.Timestamp.Format("2006-01-02T15")
		for _, trend := range trends {
			if trend.Hour == hourKey {
				trend.Count++
				if entry.Level == model.LevelError || entry.Level == model.LevelFatal {
					trend.ErrorCount++
				}
				break
			}
		}
	}

	for _, trend := range trends {
		if trend.Count > 0 {
			trend.ErrorRate = float64(trend.ErrorCount) / float64(trend.Count)
		}
	}

	return trends, nil
}

// GetSourceStats 获取各来源统计
func (s *StatsService) GetSourceStats(ctx context.Context, limit int) ([]*model.SourceStats, error) {
	if limit <= 0 {
		limit = s.config.Stats.TopSourcesCount
		if limit <= 0 {
			limit = 10
		}
	}

	now := time.Now()
	endTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), time.UTC)
	startTime := endTime.Add(-24 * time.Hour)

	query := &model.LogQuery{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     100000,
	}

	result, err := s.store.Query(query)
	if err != nil {
		return nil, err
	}

	sourceMap := make(map[string]*model.SourceStats)
	for _, entry := range result.Items {
		stats, ok := sourceMap[entry.Source]
		if !ok {
			stats = &model.SourceStats{
				Source:  entry.Source,
				ByLevel: make(map[model.LogLevel]int64),
			}
			sourceMap[entry.Source] = stats
		}
		stats.TotalCount++
		stats.ByLevel[entry.Level]++
		if entry.Timestamp.After(stats.LastActivity) {
			stats.LastActivity = entry.Timestamp
		}
	}

	sourceList := make([]*model.SourceStats, 0, len(sourceMap))
	for _, stats := range sourceMap {
		if stats.TotalCount > 0 {
			errorCount := stats.ByLevel[model.LevelError] + stats.ByLevel[model.LevelFatal]
			stats.ErrorRate = float64(errorCount) / float64(stats.TotalCount)
		}
		sourceList = append(sourceList, stats)
	}

	sort.Slice(sourceList, func(i, j int) bool {
		return sourceList[i].TotalCount > sourceList[j].TotalCount
	})

	if len(sourceList) > limit {
		sourceList = sourceList[:limit]
	}

	return sourceList, nil
}

// GetDailyReport 获取日报
func (s *StatsService) GetDailyReport(ctx context.Context, date time.Time) (*model.DailyReport, error) {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)

	query := &model.LogQuery{
		StartTime: &dayStart,
		EndTime:   &dayEnd,
		Limit:     100000,
	}

	result, err := s.store.Query(query)
	if err != nil {
		return nil, err
	}

	report := &model.DailyReport{
		Date:        dayStart.Format("2006-01-02"),
		TopErrors:   make([]model.SourceError, 0),
	}

	hourCounts := make(map[string]int64)
	sourceStatsMap := make(map[string]*model.SourceError)

	for _, entry := range result.Items {
		report.TotalCount++
		report.SourceCount++

		hourKey := entry.Timestamp.Format("2006-01-02T15")
		hourCounts[hourKey]++

		source := entry.Source
		if _, ok := sourceStatsMap[source]; !ok {
			sourceStatsMap[source] = &model.SourceError{Source: source}
		}
		sourceStatsMap[source].Count++
		if entry.Level == model.LevelError || entry.Level == model.LevelFatal {
			sourceStatsMap[source].ErrorRate++
		}
	}

	var peakHour string
	var peakCount int64
	for hour, count := range hourCounts {
		if count > peakCount {
			peakCount = count
			peakHour = hour
		}
	}
	report.PeakHour = peakHour
	report.PeakCount = peakCount

	for _, se := range sourceStatsMap {
		if se.Count > 0 {
			se.ErrorRate = se.ErrorRate / float64(se.Count)
		}
	}

	sourceErrors := make([]model.SourceError, 0, len(sourceStatsMap))
	for _, se := range sourceStatsMap {
		sourceErrors = append(sourceErrors, *se)
	}
	sort.Slice(sourceErrors, func(i, j int) bool {
		return sourceErrors[i].ErrorRate > sourceErrors[j].ErrorRate
	})
	if len(sourceErrors) > 10 {
		sourceErrors = sourceErrors[:10]
	}
	report.TopErrors = sourceErrors

	return report, nil
}

// GetErrorRateTrend 获取错误率趋势
func (s *StatsService) GetErrorRateTrend(ctx context.Context, hours int) ([]*model.HourlyTrend, error) {
	return s.GetHourlyTrends(ctx, hours)
}

// GetTotalCount 获取指定时间范围的总日志数
func (s *StatsService) GetTotalCount(ctx context.Context, source string, level model.LogLevel, startTime, endTime time.Time) (int64, error) {
	return s.store.Count(source, level, startTime, endTime)
}
