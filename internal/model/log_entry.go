// Package model 定义系统核心数据模型
package model

import (
	"time"
)

// LogEntry 表示一条日志记录
type LogEntry struct {
	// ID 日志唯一标识符
	ID string `json:"id"`
	// Timestamp 日志产生时间
	Timestamp time.Time `json:"timestamp"`
	// Level 日志级别
	Level LogLevel `json:"level"`
	// Source 日志来源（服务名、主机名等）
	Source string `json:"source"`
	// Message 日志消息内容
	Message string `json:"message"`
	// Keywords 关键词列表，用于过滤和搜索
	Keywords []string `json:"keywords,omitempty"`
	// TraceID 链路追踪ID，用于关联请求
	TraceID string `json:"trace_id,omitempty"`
	// Metadata 附加元数据
	Metadata map[string]string `json:"metadata,omitempty"`
	// CreatedAt 记录创建时间
	CreatedAt time.Time `json:"created_at"`
}

// NewLogEntry 创建新的日志条目
func NewLogEntry(level LogLevel, source, message string) *LogEntry {
	now := time.Now()
	return &LogEntry{
		ID:        GenerateID(),
		Timestamp: now,
		Level:     level,
		Source:    source,
		Message:   message,
		Keywords:  ExtractKeywords(message),
		CreatedAt: now,
	}
}

// WithTraceID 设置链路追踪ID
func (e *LogEntry) WithTraceID(traceID string) *LogEntry {
	e.TraceID = traceID
	return e
}

// WithMetadata 设置元数据
func (e *LogEntry) WithMetadata(key, value string) *LogEntry {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// WithTimestamp 设置时间戳
func (e *LogEntry) WithTimestamp(t time.Time) *LogEntry {
	e.Timestamp = t
	return e
}

// ContainsKeyword 检查日志是否包含指定关键词
func (e *LogEntry) ContainsKeyword(keyword string) bool {
	keyword = normalizeKeyword(keyword)
	for _, kw := range e.Keywords {
		if kw == keyword {
			return true
		}
	}
	return false
}

// MatchesLevel 检查日志级别是否匹配
func (e *LogEntry) MatchesLevel(level LogLevel) bool {
	if level == "" {
		return true
	}
	return e.Level == level
}

// MatchesSource 检查日志来源是否匹配
func (e *LogEntry) MatchesSource(source string) bool {
	if source == "" {
		return true
	}
	return e.Source == source
}

// Age 返回日志的年龄
func (e *LogEntry) Age() time.Duration {
	return time.Since(e.Timestamp)
}

// IsExpired 检查日志是否过期
func (e *LogEntry) IsExpired(retentionPeriod time.Duration) bool {
	if retentionPeriod <= 0 {
		return false
	}
	return e.Age() > retentionPeriod
}
