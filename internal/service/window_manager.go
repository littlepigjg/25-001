// Package service 提供业务逻辑层
package service

import (
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/pkg/logger"
	"time"
)

// WindowManager 时间窗口管理器
type WindowManager struct {
	config *config.Config
	logger *logger.Logger
}

// NewWindowManager 创建时间窗口管理器
func NewWindowManager(cfg *config.Config, log *logger.Logger) *WindowManager {
	return &WindowManager{
		config: cfg,
		logger: log,
	}
}

// GetAlertWindow 获取告警规则的时间窗口
func (wm *WindowManager) GetAlertWindow(rule *model.AlertRule) (time.Time, time.Time) {
	now := time.Now()
	windowStart := now.Add(-time.Duration(rule.WindowMinutes) * time.Minute)
	return windowStart, now
}

// GetDefaultWindow 获取默认统计窗口
func (wm *WindowManager) GetDefaultWindow() (time.Time, time.Time) {
	now := time.Now()
	defaultWindow := wm.config.Stats.DefaultWindow
	if defaultWindow <= 0 {
		defaultWindow = 24 * time.Hour
	}
	return now.Add(-defaultWindow), now
}

// GetHourlyWindow 获取每小时窗口
func (wm *WindowManager) GetHourlyWindow(hours int) (time.Time, time.Time) {
	now := time.Now()
	if hours <= 0 {
		hours = 24
	}
	return now.Add(-time.Duration(hours) * time.Hour), now
}

// GetDailyWindow 获取每日窗口
func (wm *WindowManager) GetDailyWindow(date time.Time) (time.Time, time.Time) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	end := start.Add(24 * time.Hour)
	return start, end
}

// ValidateWindow 验证时间窗口合法性
func (wm *WindowManager) ValidateWindow(start, end time.Time) error {
	if start.After(end) {
		return model.NewValidationError("window", "start time must be before end time")
	}
	if end.After(time.Now()) {
		return model.NewValidationError("window", "end time cannot be in the future")
	}
	maxHours := wm.config.Query.MaxLimit
	if maxHours <= 0 {
		maxHours = 720 // 30 days
	}
	duration := end.Sub(start)
	if duration > time.Duration(maxHours)*time.Hour {
		return model.NewValidationError("window", "time window exceeds maximum allowed range")
	}
	return nil
}

// GetRetainedPeriod 获取保留期
func (wm *WindowManager) GetRetainedPeriod() time.Duration {
	return wm.config.Storage.RetentionPeriod
}

// TruncateToHour 截断到小时
func (wm *WindowManager) TruncateToHour(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}

// TruncateToDay 截断到天
func (wm *WindowManager) TruncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
