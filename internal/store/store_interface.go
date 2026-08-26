// Package store 提供数据存储层
package store

import (
	"logalert/internal/model"
	"time"
)

// LogStore 日志存储接口
type LogStore interface {
	// Store 存储一条日志
	Store(entry *model.LogEntry) error
	// StoreBatch 批量存储日志
	StoreBatch(entries []*model.LogEntry) error
	// GetByID 根据ID获取日志
	GetByID(id string) (*model.LogEntry, error)
	// Query 查询日志
	Query(query *model.LogQuery) (*model.LogQueryResult, error)
	// Count 统计数量
	Count(source string, level model.LogLevel, startTime, endTime time.Time) (int64, error)
	// DeleteByID 根据ID删除
	DeleteByID(id string) error
	// DeleteBySource 删除指定来源的日志
	DeleteBySource(source string) (int64, error)
	// Cleanup 清理过期日志
	Cleanup(retentionPeriod time.Duration) (int64, error)
	// Stats 获取存储统计
	Stats() (*StoreStats, error)
}

// RuleStore 规则存储接口
type RuleStore interface {
	// Create 创建规则
	Create(rule *model.AlertRule) error
	// GetByID 根据ID获取规则
	GetByID(id string) (*model.AlertRule, error)
	// List 列出所有规则
	List() ([]*model.AlertRule, error)
	// ListBySource 列出指定来源的规则
	ListBySource(source string) ([]*model.AlertRule, error)
	// Update 更新规则
	Update(rule *model.AlertRule) error
	// Delete 删除规则
	Delete(id string) error
	// Enable 启用规则
	Enable(id string) error
	// Disable 禁用规则
	Disable(id string) error
	// Count 统计规则数量
	Count() (int, error)
}

// AlertStore 告警事件存储接口
type AlertStore interface {
	// Store 存储告警事件
	Store(event *model.AlertEvent) error
	// GetByID 根据ID获取事件
	GetByID(id string) (*model.AlertEvent, error)
	// List 列出所有告警事件
	List() ([]*model.AlertEvent, error)
	// ListByRule 列出规则相关的告警
	ListByRule(ruleID string) ([]*model.AlertEvent, error)
	// ListBySource 列出来源相关的告警
	ListBySource(source string) ([]*model.AlertEvent, error)
	// ListActive 列出活跃告警
	ListActive() ([]*model.AlertEvent, error)
	// Acknowledge 确认告警
	Acknowledge(id string) error
	// Resolve 解决告警
	Resolve(id string) error
	// Delete 删除告警
	Delete(id string) error
	// Cleanup 清理过期告警
	Cleanup(retentionPeriod time.Duration) (int64, error)
	// CountActive 统计活跃告警数量
	CountActive() (int, error)
}

// StoreStats 存储统计信息
type StoreStats struct {
	// TotalEntries 总日志条目数
	TotalEntries int64 `json:"total_entries"`
	// TotalRules 总规则数
	TotalRules int `json:"total_rules"`
	// TotalAlerts 总告警数
	TotalAlerts int `json:"total_alerts"`
	// ActiveAlerts 活跃告警数
	ActiveAlerts int `json:"active_alerts"`
	// BySource 按来源统计
	BySource map[string]int64 `json:"by_source"`
	// ByLevel 按级别统计
	ByLevel map[model.LogLevel]int64 `json:"by_level"`
}
