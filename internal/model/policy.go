// Package model 定义系统核心数据模型
package model

import "time"

// LogRetentionPolicy 日志保留策略
type LogRetentionPolicy struct {
	// MaxAge 最大保留时间
	MaxAge time.Duration `json:"max_age"`
	// MaxEntries 最大条目数
	MaxEntries int `json:"max_entries"`
	// MaxPerSource 每来源最大条目数
	MaxPerSource int `json:"max_per_source"`
	// AutoCleanup 是否自动清理
	AutoCleanup bool `json:"auto_cleanup"`
	// CleanupInterval 清理间隔
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// DefaultRetentionPolicy 创建默认保留策略
func DefaultRetentionPolicy() *LogRetentionPolicy {
	return &LogRetentionPolicy{
		MaxAge:          24 * time.Hour * 7,
		MaxEntries:      100000,
		MaxPerSource:    10000,
		AutoCleanup:     true,
		CleanupInterval: 30 * time.Minute,
	}
}

// AlertPolicy 告警策略
type AlertPolicy struct {
	// MaxRulesPerSource 每来源最大规则数
	MaxRulesPerSource int `json:"max_rules_per_source"`
	// MaxAlertsPerDay 每日最大告警数
	MaxAlertsPerDay int `json:"max_alerts_per_day"`
	// CooldownMinutes 规则冷却时间
	CooldownMinutes int `json:"cooldown_minutes"`
	// AutoResolveHours 自动解决时间
	AutoResolveHours int `json:"auto_resolve_hours"`
}

// DefaultAlertPolicy 创建默认告警策略
func DefaultAlertPolicy() *AlertPolicy {
	return &AlertPolicy{
		MaxRulesPerSource: 100,
		MaxAlertsPerDay:   10000,
		CooldownMinutes:   5,
		AutoResolveHours:  24,
	}
}

// QueryPolicy 查询策略
type QueryPolicy struct {
	// DefaultLimit 默认返回条数
	DefaultLimit int `json:"default_limit"`
	// MaxLimit 最大返回条数
	MaxLimit int `json:"max_limit"`
	// MaxTimeRangeHours 最大查询时间范围
	MaxTimeRangeHours int `json:"max_time_range_hours"`
	// EnableFullTextSearch 是否启用全文搜索
	EnableFullTextSearch bool `json:"enable_full_text_search"`
}

// DefaultQueryPolicy 创建默认查询策略
func DefaultQueryPolicy() *QueryPolicy {
	return &QueryPolicy{
		DefaultLimit:         100,
		MaxLimit:             10000,
		MaxTimeRangeHours:    720,
		EnableFullTextSearch: true,
	}
}
