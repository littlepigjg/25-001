// Package model 定义系统核心数据模型
package model

import (
	"time"
)

// AlertEvent 告警事件
type AlertEvent struct {
	// ID 事件唯一标识
	ID string `json:"id"`
	// RuleID 触发的规则ID
	RuleID string `json:"rule_id"`
	// RuleName 触发的规则名称
	RuleName string `json:"rule_name"`
	// Source 日志来源
	Source string `json:"source"`
	// Level 日志级别
	Level LogLevel `json:"level"`
	// Count 触发时的日志数量
	Count int `json:"count"`
	// Threshold 触发阈值
	Threshold int `json:"threshold"`
	// WindowMinutes 时间窗口
	WindowMinutes int `json:"window_minutes"`
	// TriggeredAt 触发时间
	TriggeredAt time.Time `json:"triggered_at"`
	// ResolvedAt 解决时间（为空表示未解决）
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	// Status 事件状态
	Status AlertStatus `json:"status"`
	// Description 事件描述
	Description string `json:"description"`
}

// AlertStatus 告警状态
type AlertStatus string

const (
	// AlertFired 已触发
	AlertFired AlertStatus = "FIRED"
	// AlertResolved 已解决
	AlertResolved AlertStatus = "RESOLVED"
	// AlertAcknowledged 已确认
	AlertAcknowledged AlertStatus = "ACKNOWLEDGED"
)

// NewAlertEvent 创建新的告警事件
func NewAlertEvent(rule *AlertRule, count int) *AlertEvent {
	now := time.Now()
	return &AlertEvent{
		ID:             GenerateID(),
		RuleID:         rule.ID,
		RuleName:       rule.Name,
		Source:         rule.Source,
		Level:          rule.Level,
		Count:          count,
		Threshold:      rule.Threshold,
		WindowMinutes:  rule.WindowMinutes,
		TriggeredAt:    now,
		Status:         AlertFired,
		Description:    buildAlertDescription(rule, count),
	}
}

// buildAlertDescription 构建告警描述
func buildAlertDescription(rule *AlertRule, count int) string {
	desc := "规则[" + rule.Name + "]触发: "
	desc += rule.Source + " 来源在 "
	desc += itoa(rule.WindowMinutes) + " 分钟内有 "
	desc += itoa(count) + " 条 "
	desc += string(rule.Level) + " 日志，超过阈值 "
	desc += itoa(rule.Threshold)
	return desc
}

// itoa 简单整数转字符串
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// Acknowledge 确认告警
func (e *AlertEvent) Acknowledge() {
	e.Status = AlertAcknowledged
}

// Resolve 解决告警
func (e *AlertEvent) Resolve() {
	now := time.Now()
	e.ResolvedAt = &now
	e.Status = AlertResolved
}

// IsActive 检查告警是否活跃
func (e *AlertEvent) IsActive() bool {
	return e.Status == AlertFired || e.Status == AlertAcknowledged
}

// Duration 计算告警持续时间
func (e *AlertEvent) Duration() time.Duration {
	end := time.Now()
	if e.ResolvedAt != nil {
		end = *e.ResolvedAt
	}
	return end.Sub(e.TriggeredAt)
}
