// Package model 定义系统核心数据模型
package model

import (
	"fmt"
	"strings"
)

// LogLevel 表示日志级别
type LogLevel string

const (
	// LevelInfo 信息级别
	LevelInfo LogLevel = "INFO"
	// LevelWarn 警告级别
	LevelWarn LogLevel = "WARN"
	// LevelError 错误级别
	LevelError LogLevel = "ERROR"
	// LevelDebug 调试级别
	LevelDebug LogLevel = "DEBUG"
	// LevelFatal 致命级别
	LevelFatal LogLevel = "FATAL"
)

// AllLevels 所有合法的日志级别
var AllLevels = []LogLevel{LevelInfo, LevelWarn, LevelError, LevelDebug, LevelFatal}

// Valid 检查日志级别是否合法
func (l LogLevel) Valid() bool {
	switch l {
	case LevelInfo, LevelWarn, LevelError, LevelDebug, LevelFatal:
		return true
	}
	return false
}

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	return string(l)
}

// ParseLogLevel 解析字符串为日志级别
func ParseLogLevel(s string) (LogLevel, error) {
	upper := strings.ToUpper(strings.TrimSpace(s))
	for _, level := range AllLevels {
		if string(level) == upper {
			return level, nil
		}
	}
	return "", fmt.Errorf("invalid log level: %s", s)
}

// Compare 比较两个日志级别的严重程度
// 返回正数表示当前级别更严重，负数表示另一个更严重，0表示相同
func (l LogLevel) Compare(other LogLevel) int {
	return levelSeverity(l) - levelSeverity(other)
}

// levelSeverity 返回日志级别的严重程度数值
func levelSeverity(l LogLevel) int {
	switch l {
	case LevelDebug:
		return 0
	case LevelInfo:
		return 1
	case LevelWarn:
		return 2
	case LevelError:
		return 3
	case LevelFatal:
		return 4
	default:
		return -1
	}
}

// HigherOrEqual 检查当前级别是否至少等于指定的严重程度
func (l LogLevel) HigherOrEqual(min LogLevel) bool {
	return levelSeverity(l) >= levelSeverity(min)
}
