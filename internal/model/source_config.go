// Package model 定义系统核心数据模型
package model

import "time"

// SourceConfig 来源配置
type SourceConfig struct {
	// Source 来源名称
	Source string `json:"source"`
	// Enabled 是否启用日志接收
	Enabled bool `json:"enabled"`
	// MaxLogSize 最大日志大小（字节）
	MaxLogSize int `json:"max_log_size"`
	// RateLimit 每秒最大日志数
	RateLimit int `json:"rate_limit"`
	// Retention 保留时间
	Retention time.Duration `json:"retention"`
	// Tags 来源标签
	Tags []string `json:"tags,omitempty"`
}

// DefaultSourceConfig 创建默认来源配置
func DefaultSourceConfig(source string) *SourceConfig {
	return &SourceConfig{
		Source:     source,
		Enabled:    true,
		MaxLogSize: 4096,
		RateLimit:  100,
		Retention:  24 * time.Hour * 7,
	}
}

// Validate 验证来源配置
func (sc *SourceConfig) Validate() error {
	if sc.Source == "" {
		return NewValidationError("source", "source cannot be empty")
	}
	if sc.MaxLogSize <= 0 || sc.MaxLogSize > 1048576 {
		return NewValidationError("max_log_size", "max_log_size must be between 1 and 1MB")
	}
	if sc.RateLimit < 0 {
		return NewValidationError("rate_limit", "rate_limit cannot be negative")
	}
	return nil
}

// SourceHealth 来源健康状态
type SourceHealth struct {
	Source        string    `json:"source"`
	LastActivity  time.Time `json:"last_activity"`
	TotalLogs     int64     `json:"total_logs"`
	ErrorRate     float64   `json:"error_rate"`
	Status        string    `json:"status"`
	Uptime        float64   `json:"uptime"`
}

// SourceFilter 来源过滤器
type SourceFilter struct {
	Sources  []string `json:"sources"`
	Exclude  []string `json:"exclude"`
	MinCount int      `json:"min_count"`
}
