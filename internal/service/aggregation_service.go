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

// AggregationService 聚合服务
type AggregationService struct {
	logStore  store.LogStore
	config    *config.Config
	logger    *logger.Logger
}

// NewAggregationService 创建聚合服务
func NewAggregationService(logStore store.LogStore, cfg *config.Config, log *logger.Logger) *AggregationService {
	return &AggregationService{
		logStore: logStore,
		config:   cfg,
		logger:   log,
	}
}

// AggregateByHour 按小时聚合
func (as *AggregationService) AggregateByHour(ctx context.Context, hours int) (map[string]int64, error) {
	if hours <= 0 {
		hours = 24
	}

	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(hours) * time.Hour)

	query := &model.LogQuery{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     100000,
	}

	result, err := as.logStore.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	hourlyData := make(map[string]int64)
	for _, entry := range result.Items {
		hourKey := entry.Timestamp.Format("2006-01-02T15")
		hourlyData[hourKey]++
	}

	return hourlyData, nil
}

// AggregateBySource 按来源聚合
func (as *AggregationService) AggregateBySource(ctx context.Context, startTime, endTime time.Time) (map[string]int64, error) {
	query := &model.LogQuery{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     100000,
	}

	result, err := as.logStore.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	sourceData := make(map[string]int64)
	for _, entry := range result.Items {
		sourceData[entry.Source]++
	}

	return sourceData, nil
}

// AggregateByLevel 按级别聚合
func (as *AggregationService) AggregateByLevel(ctx context.Context, startTime, endTime time.Time) (map[model.LogLevel]int64, error) {
	query := &model.LogQuery{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     100000,
	}

	result, err := as.logStore.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	levelData := make(map[model.LogLevel]int64)
	for _, entry := range result.Items {
		levelData[entry.Level]++
	}

	return levelData, nil
}

// GetTopSources 获取日志量最多的来源
func (as *AggregationService) GetTopSources(ctx context.Context, limit int) ([]model.SourceError, error) {
	if limit <= 0 {
		limit = 10
	}

	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	query := &model.LogQuery{
		StartTime: &startTime,
		EndTime:   &endTime,
		Limit:     100000,
	}

	result, err := as.logStore.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	sourceCount := make(map[string]int64)
	sourceErrors := make(map[string]int64)

	for _, entry := range result.Items {
		sourceCount[entry.Source]++
		if entry.Level == model.LevelError || entry.Level == model.LevelFatal {
			sourceErrors[entry.Source]++
		}
	}

	// 转换为排序切片
	sources := make([]model.SourceError, 0, len(sourceCount))
	for source, count := range sourceCount {
		errorRate := float64(0)
		if count > 0 {
			errorRate = float64(sourceErrors[source]) / float64(count)
		}
		sources = append(sources, model.SourceError{
			Source:    source,
			Count:     count,
			ErrorRate: errorRate,
		})
	}

	// 排序
	for i := 0; i < len(sources)-1; i++ {
		for j := i + 1; j < len(sources); j++ {
			if sources[i].Count < sources[j].Count {
				sources[i], sources[j] = sources[j], sources[i]
			}
		}
	}

	if len(sources) > limit {
		sources = sources[:limit]
	}

	return sources, nil
}
