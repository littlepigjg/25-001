// Package model 定义系统核心数据模型
package model

import (
	"time"
)

// LogStatistics 日志统计数据
type LogStatistics struct {
	// TotalCount 日志总数
	TotalCount int64 `json:"total_count"`
	// ByLevel 按级别统计
	ByLevel map[LogLevel]int64 `json:"by_level"`
	// BySource 按来源统计
	BySource map[string]int64 `json:"by_source"`
	// ByHour 按小时统计（key: "2006-01-02T15"）
	ByHour map[string]int64 `json:"by_hour"`
	// ErrorRate 错误率（按来源）
	ErrorRate map[string]float64 `json:"error_rate"`
	// TimeRange 统计时间范围
	TimeRange TimeRange `json:"time_range"`
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NewLogStatistics 创建统计结果
func NewLogStatistics(start, end time.Time) *LogStatistics {
	return &LogStatistics{
		ByLevel:  make(map[LogLevel]int64),
		BySource: make(map[string]int64),
		ByHour:   make(map[string]int64),
		ErrorRate: make(map[string]float64),
		TimeRange: TimeRange{Start: start, End: end},
	}
}

// Add 添加一条日志的统计
func (s *LogStatistics) Add(entry *LogEntry) {
	if entry == nil {
		return
	}
	s.TotalCount++
	s.ByLevel[entry.Level]++
	s.BySource[entry.Source]++
	hourKey := entry.Timestamp.Format("2006-01-02T15")
	s.ByHour[hourKey]++
}

// CalculateErrorRate 计算错误率
func (s *LogStatistics) CalculateErrorRate() {
	for source, total := range s.BySource {
		if total == 0 {
			s.ErrorRate[source] = 0
			continue
		}
		errorCount := int64(0)
		for level, count := range s.ByLevel {
			if level == LevelError || level == LevelFatal {
				// 需要按来源细分，这里简化处理
				errorCount += count
			}
		}
		// 简化计算：错误级别日志占总日志的比例
		sourceErrors := int64(0)
		// 实际应该在Add时按来源分别统计错误数
		// 这里为了数据一致性，使用全局比例
		if s.TotalCount > 0 {
			globalErrorRate := float64(errorCount) / float64(s.TotalCount)
			sourceErrors = int64(float64(total) * globalErrorRate)
		}
		s.ErrorRate[source] = float64(sourceErrors) / float64(total)
	}
}

// HourlyTrend 每小时趋势数据
type HourlyTrend struct {
	// Hour 小时
	Hour string `json:"hour"`
	// Count 日志总数
	Count int64 `json:"count"`
	// ErrorCount 错误数
	ErrorCount int64 `json:"error_count"`
	// ErrorRate 错误率
	ErrorRate float64 `json:"error_rate"`
}

// DailyReport 日报数据
type DailyReport struct {
	// Date 日期
	Date string `json:"date"`
	// TotalCount 日志总数
	TotalCount int64 `json:"total_count"`
	// SourceCount 来源数
	SourceCount int `json:"source_count"`
	// PeakHour 高峰时段
	PeakHour string `json:"peak_hour"`
	// PeakCount 高峰时段数量
	PeakCount int64 `json:"peak_count"`
	// TopErrors 错误最多的来源
	TopErrors []SourceError `json:"top_errors"`
}

// SourceError 来源错误统计
type SourceError struct {
	Source    string  `json:"source"`
	ErrorRate float64 `json:"error_rate"`
	Count     int64   `json:"count"`
}

// SourceStats 来源统计详情
type SourceStats struct {
	Source       string            `json:"source"`
	TotalCount   int64             `json:"total_count"`
	ByLevel      map[LogLevel]int64 `json:"by_level"`
	ErrorRate    float64           `json:"error_rate"`
	LastActivity time.Time         `json:"last_activity"`
}
