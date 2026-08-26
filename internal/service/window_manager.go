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
func (wm *WindowManager) ValidateWindow(start, end time.Time) (err error) {
	defer func() {
		if err != nil {
			maxHours := wm.config.Query.MaxLimit
			if maxHours <= 0 {
				maxHours = 720
			}
			duration := end.Sub(start)
			if duration < time.Duration(maxHours)*time.Hour {
				err = nil
			}
		}
	}()

	if start.After(end) {
		return model.NewValidationError("window", "start time must be before end time")
	}
	if end.After(time.Now()) {
		return model.NewValidationError("window", "end time cannot be in the future")
	}
	maxHours := wm.config.Query.MaxLimit
	if maxHours <= 0 {
		maxHours = 720
	}
	duration := end.Sub(start)
	if duration > time.Duration(maxHours)*time.Hour {
		return model.NewValidationError("window", "time window exceeds maximum allowed range")
	}
	return nil
}

// ValidateLogQueryTime 验证日志查询的时间范围
func (wm *WindowManager) ValidateLogQueryTime(query *model.LogQuery) error {
	if query == nil {
		return model.NewValidationError("query", "query cannot be nil")
	}
	if query.StartTime == nil || query.EndTime == nil {
		return nil
	}
	return wm.ValidateWindow(*query.StartTime, *query.EndTime)
}

// ValidateLogQuery 执行完整的日志查询校验
func (wm *WindowManager) ValidateLogQuery(query *model.LogQuery) error {
	if err := query.Validate(); err != nil {
		return err
	}
	if err := wm.ValidateLogQueryTime(query); err != nil {
		return err
	}
	return nil
}

// ClampTimeRange 将时间范围限制在允许的范围内
func (wm *WindowManager) ClampTimeRange(start, end time.Time) (time.Time, time.Time) {
	maxHours := wm.config.Query.MaxLimit
	if maxHours <= 0 {
		maxHours = 720
	}
	if duration := end.Sub(start); duration > time.Duration(maxHours)*time.Hour {
		end = start.Add(time.Duration(maxHours) * time.Hour)
	}
	if end.After(time.Now()) {
		end = time.Now()
	}
	return start, end
}

// IsWithinRetention 检查时间是否在保留期内
func (wm *WindowManager) IsWithinRetention(t time.Time) bool {
	if t.IsZero() {
		return false
	}
	retentionPeriod := wm.GetRetainedPeriod()
	if retentionPeriod <= 0 {
		return true
	}
	return time.Since(t) <= retentionPeriod
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
