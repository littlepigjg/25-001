// Package timeutil 提供时间窗口工具
package timeutil

import (
	"time"
)

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NewTimeRange 创建时间范围
func NewTimeRange(start, end time.Time) *TimeRange {
	return &TimeRange{Start: start, End: end}
}

// NewLastMinutes 创建最近N分钟的时间范围
func NewLastMinutes(minutes int) *TimeRange {
	now := time.Now()
	return &TimeRange{
		Start: now.Add(-time.Duration(minutes) * time.Minute),
		End:   now,
	}
}

// NewLastHours 创建最近N小时的时间范围
func NewLastHours(hours int) *TimeRange {
	now := time.Now()
	return &TimeRange{
		Start: now.Add(-time.Duration(hours) * time.Hour),
		End:   now,
	}
}

// NewLastDays 创建最近N天的时间范围
func NewLastDays(days int) *TimeRange {
	now := time.Now()
	return &TimeRange{
		Start: now.Add(-time.Duration(days) * 24 * time.Hour),
		End:   now,
	}
}

// Contains 检查时间是否在范围内
func (tr *TimeRange) Contains(t time.Time) bool {
	return (t.After(tr.Start) || t.Equal(tr.Start)) && (t.Before(tr.End) || t.Equal(tr.End))
}

// Duration 返回时间范围时长
func (tr *TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

// DurationMinutes 返回时长（分钟）
func (tr *TimeRange) DurationMinutes() int {
	return int(tr.Duration().Minutes())
}

// DurationHours 返回时长（小时）
func (tr *TimeRange) DurationHours() int {
	return int(tr.Duration().Hours())
}

// DurationDays 返回时长（天）
func (tr *TimeRange) DurationDays() int {
	return int(tr.Duration().Hours() / 24)
}

// IsEmpty 检查范围是否为空
func (tr *TimeRange) IsEmpty() bool {
	return !tr.Start.Before(tr.End)
}

// HourKey 生成小时键
func HourKey(t time.Time) string {
	return t.Format("2006-01-02T15")
}

// DateKey 生成日期键
func DateKey(t time.Time) string {
	return t.Format("2006-01-02")
}

// DayStart 获取一天的开始时间
func DayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// DayEnd 获取一天的结束时间
func DayEnd(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), t.Location())
}

// HourStart 获取一小时的开始时间
func HourStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
}

// HourEnd 获取一小时的结束时间
func HourEnd(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 59, 59, int(time.Second-time.Nanosecond), t.Location())
}

// Now 返回当前时间
func Now() time.Time {
	return time.Now()
}
