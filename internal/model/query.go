// Package model 定义系统核心数据模型
package model

import (
	"time"
)

// LogQuery 日志查询条件
type LogQuery struct {
	// Source 日志来源过滤（空字符串表示不滤）
	Source string `json:"source,omitempty"`
	// Level 日志级别过滤
	Level LogLevel `json:"level,omitempty"`
	// Keywords 关键词过滤（AND逻辑）
	Keywords []string `json:"keywords,omitempty"`
	// StartTime 查询起始时间
	StartTime *time.Time `json:"start_time,omitempty"`
	// EndTime 查询结束时间
	EndTime *time.Time `json:"end_time,omitempty"`
	// Limit 返回最大条数
	Limit int `json:"limit,omitempty"`
	// Offset 偏移量
	Offset int `json:"offset,omitempty"`
	// SortOrder 排序方式
	SortOrder SortOrder `json:"sort_order,omitempty"`
}

// SortOrder 排序方式
type SortOrder string

const (
	// SortAscending 升序
	SortAscending SortOrder = "asc"
	// SortDescending 降序
	SortDescending SortOrder = "desc"
)

// DefaultLogQuery 创建默认查询条件
func DefaultLogQuery() *LogQuery {
	return &LogQuery{
		Limit:     100,
		SortOrder: SortDescending,
	}
}

// Validate 验证查询条件
func (q *LogQuery) Validate() error {
	if q.Limit < 0 {
		return &ValidationError{Field: "limit", Message: "limit不能为负数"}
	}
	if q.Limit > 10000 {
		q.Limit = 10000
	}
	if q.Offset < 0 {
		return &ValidationError{Field: "offset", Message: "offset不能为负数"}
	}
	if q.StartTime != nil && q.EndTime != nil {
		if q.StartTime.After(*q.EndTime) {
			return &ValidationError{Field: "time_range", Message: "起始时间不能晚于结束时间"}
		}
	}
	return nil
}

// TimeRange 构建时间范围
func (q *LogQuery) TimeRange() (time.Time, time.Time) {
	start := time.Unix(0, 0)
	end := time.Now()
	if q.StartTime != nil {
		start = *q.StartTime
	}
	if q.EndTime != nil {
		end = *q.EndTime
	}
	return start, end
}

// Matches 检查日志是否匹配查询条件
func (q *LogQuery) Matches(entry *LogEntry) bool {
	if entry == nil {
		return false
	}
	// 时间范围
	start, end := q.TimeRange()
	if entry.Timestamp.Before(start) || entry.Timestamp.After(end) {
		return false
	}
	// 来源
	if q.Source != "" && entry.Source != q.Source {
		return false
	}
	// 级别
	if q.Level != "" && entry.Level != q.Level {
		return false
	}
	// 关键词
	if len(q.Keywords) > 0 {
		if !ContainsAllKeywords(entry.Message, q.Keywords) {
			return false
		}
	}
	return true
}

// LogQueryResult 查询结果
type LogQueryResult struct {
	// Total 总匹配数
	Total int `json:"total"`
	// Items 当前页数据
	Items []*LogEntry `json:"items"`
	// Query 查询条件
	Query LogQuery `json:"query"`
}

// NewLogQueryResult 创建查询结果
func NewLogQueryResult(query LogQuery, items []*LogEntry, total int) *LogQueryResult {
	return &LogQueryResult{
		Total: total,
		Items: items,
		Query: query,
	}
}
