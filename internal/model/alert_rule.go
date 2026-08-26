// Package model 定义系统核心数据模型
package model

import (
	"time"
)

// AlertRule 告警规则定义
type AlertRule struct {
	// ID 规则唯一标识
	ID string `json:"id"`
	// Name 规则名称
	Name string `json:"name"`
	// Description 规则描述
	Description string `json:"description"`
	// Source 监控的日志来源（空字符串表示所有来源）
	Source string `json:"source"`
	// Level 监控的日志级别
	Level LogLevel `json:"level"`
	// WindowMinutes 时间窗口（分钟）
	WindowMinutes int `json:"window_minutes"`
	// Threshold 触发阈值
	Threshold int `json:"threshold"`
	// Enabled 是否启用
	Enabled bool `json:"enabled"`
	// Priority 优先级（1-10，数字越大优先级越高）
	Priority int `json:"priority"`
	// Tags 规则标签
	Tags []string `json:"tags,omitempty"`
	// CreatedAt 创建时间
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// NewAlertRule 创建新的告警规则
func NewAlertRule(name, source string, level LogLevel, windowMinutes, threshold int) *AlertRule {
	now := time.Now()
	return &AlertRule{
		ID:             GenerateID(),
		Name:           name,
		Source:         source,
		Level:          level,
		WindowMinutes:  windowMinutes,
		Threshold:      threshold,
		Enabled:        true,
		Priority:       5,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// Validate 验证规则参数合法性
func (r *AlertRule) Validate() error {
	if r.Name == "" {
		return &ValidationError{Field: "name", Message: "规则名称不能为空"}
	}
	if r.Level == "" || !r.Level.Valid() {
		return &ValidationError{Field: "level", Message: "日志级别不合法"}
	}
	if r.WindowMinutes <= 0 || r.WindowMinutes > 1440 {
		return &ValidationError{Field: "window_minutes", Message: "时间窗口必须在1-1440分钟之间"}
	}
	if r.Threshold <= 0 {
		return &ValidationError{Field: "threshold", Message: "阈值必须大于0"}
	}
	if r.Priority < 1 || r.Priority > 10 {
		return &ValidationError{Field: "priority", Message: "优先级必须在1-10之间"}
	}
	return nil
}

// WindowDuration 返回时间窗口的时长
func (r *AlertRule) WindowDuration() time.Duration {
	return time.Duration(r.WindowMinutes) * time.Minute
}

// Touch 更新规则的修改时间
func (r *AlertRule) Touch() {
	r.UpdatedAt = time.Now()
}

// MatchesLevel 检查规则是否匹配指定级别
func (r *AlertRule) MatchesLevel(level LogLevel) bool {
	return r.Level == level
}

// MatchesSource 检查规则是否匹配指定来源
func (r *AlertRule) MatchesSource(source string) bool {
	if r.Source == "" {
		return true
	}
	return r.Source == source
}
