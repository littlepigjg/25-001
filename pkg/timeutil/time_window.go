// Package timeutil 提供时间窗口工具
package timeutil

import (
	"time"
)

// TimeWindow 表示一个时间窗口
type TimeWindow struct {
	// Start 窗口开始时间
	Start time.Time `json:"start"`
	// End 窗口结束时间
	End time.Time `json:"end"`
	// Duration 窗口时长
	Duration time.Duration `json:"duration"`
}

// NewTimeWindow 创建固定时长的时间窗口（从当前时间向前推）
func NewTimeWindow(duration time.Duration) *TimeWindow {
	now := time.Now()
	return &TimeWindow{
		Start:    now.Add(-duration),
		End:      now,
		Duration: duration,
	}
}

// NewTimeWindowFrom 创建指定开始时间和时长的时间窗口
func NewTimeWindowFrom(start time.Time, duration time.Duration) *TimeWindow {
	return &TimeWindow{
		Start:    start,
		End:      start.Add(duration),
		Duration: duration,
	}
}

// NewCustomTimeWindow 创建自定义时间窗口
func NewCustomTimeWindow(start, end time.Time) *TimeWindow {
	return &TimeWindow{
		Start:    start,
		End:      end,
		Duration: end.Sub(start),
	}
}

// Contains 检查时间是否在窗口内
func (w *TimeWindow) Contains(t time.Time) bool {
	return (t.After(w.Start) || t.Equal(w.Start)) && (t.Before(w.End) || t.Equal(w.End))
}

// Overlaps 检查两个时间窗口是否重叠
func (w *TimeWindow) Overlaps(other *TimeWindow) bool {
	return w.Start.Before(other.End) && other.Start.Before(w.End)
}

// DurationMinutes 返回窗口时长（分钟）
func (w *TimeWindow) DurationMinutes() int {
	return int(w.Duration.Minutes())
}

// DurationSeconds 返回窗口时长（秒）
func (w *TimeWindow) DurationSeconds() int64 {
	return int64(w.Duration.Seconds())
}

// Shift 移动时间窗口
func (w *TimeWindow) Shift(d time.Duration) *TimeWindow {
	return &TimeWindow{
		Start:    w.Start.Add(d),
		End:      w.End.Add(d),
		Duration: w.Duration,
	}
}

// Advance 将窗口推进到下一个周期
func (w *TimeWindow) Advance() *TimeWindow {
	return w.Shift(w.Duration)
}

// Retreating 生成向前推移的时间窗口（重叠）
func (w *TimeWindow) Retreating(d time.Duration) *TimeWindow {
	return w.Shift(-d)
}

// IsEmpty 检查窗口是否为空
func (w *TimeWindow) IsEmpty() bool {
	return w.Duration <= 0 || !w.Start.Before(w.End)
}

// Center 返回窗口中心点
func (w *TimeWindow) Center() time.Time {
	return w.Start.Add(w.Duration / 2)
}
