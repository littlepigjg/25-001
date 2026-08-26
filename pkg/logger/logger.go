// Package logger 提供结构化日志功能
package logger

import (
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	// LevelDebug 调试
	LevelDebug LogLevel = iota
	// LevelInfo 信息
	LevelInfo
	// LevelWarn 警告
	LevelWarn
	// LevelError 错误
	LevelError
	// LevelFatal 致命
	LevelFatal
)

// Logger 结构化日志记录器
type Logger struct {
	mu       sync.RWMutex
	level    LogLevel
	output   *os.File
	logger   *log.Logger
	fields   map[string]interface{}
	prefix   string
}

// New 创建新的日志记录器
func New() *Logger {
	return &Logger{
		level:  LevelInfo,
		output: os.Stdout,
		logger: log.New(os.Stdout, "", 0),
		fields: make(map[string]interface{}),
	}
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel 获取当前日志级别
func (l *Logger) GetLevel() LogLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// WithField 添加结构化字段
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.RLock()
	newFields := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		newFields[k] = v
	}
	l.mu.RUnlock()
	newFields[key] = value
	return &Logger{
		level:  l.level,
		output: l.output,
		logger: l.logger,
		fields: newFields,
		prefix: l.prefix,
	}
}

// WithFields 添加多个结构化字段
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.RLock()
	newFields := make(map[string]interface{}, len(l.fields)+len(fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	l.mu.RUnlock()
	for k, v := range fields {
		newFields[k] = v
	}
	return &Logger{
		level:  l.level,
		output: l.output,
		logger: l.logger,
		fields: newFields,
		prefix: l.prefix,
	}
}

// Debug 输出调试日志
func (l *Logger) Debug(msg string) {
	l.log(LevelDebug, msg)
}

// Debugf 输出格式化调试日志
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logf(LevelDebug, format, args...)
}

// Info 输出信息日志
func (l *Logger) Info(msg string) {
	l.log(LevelInfo, msg)
}

// Infof 输出格式化信息日志
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logf(LevelInfo, format, args...)
}

// Warn 输出警告日志
func (l *Logger) Warn(msg string) {
	l.log(LevelWarn, msg)
}

// Warnf 输出格式化警告日志
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logf(LevelWarn, format, args...)
}

// Error 输出错误日志
func (l *Logger) Error(msg string) {
	l.log(LevelError, msg)
}

// Errorf 输出格式化错误日志
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logf(LevelError, format, args...)
}

// Fatal 输出致命日志
func (l *Logger) Fatal(msg string) {
	l.log(LevelFatal, msg)
	os.Exit(1)
}

// Fatalf 输出格式化致命日志
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logf(LevelFatal, format, args...)
	os.Exit(1)
}

// log 输出日志
func (l *Logger) log(level LogLevel, msg string) {
	if level < l.GetLevel() {
		return
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelStr := levelToString(level)
	
	var output string
	output = "[" + timestamp + "] [" + levelStr + "] "
	
	if l.prefix != "" {
		output += "[" + l.prefix + "] "
	}
	
	output += msg
	
	if len(l.fields) > 0 {
		output += " |"
		for k, v := range l.fields {
			output += " " + k + "=" + formatValue(v)
		}
	}
	
	l.logger.Println(output)
}

// logf 输出格式化日志
func (l *Logger) logf(level LogLevel, format string, args ...interface{}) {
	if level < l.GetLevel() {
		return
	}
	msg := formatMsg(format, args...)
	l.log(level, msg)
}

// SetOutput 设置输出目标
func (l *Logger) SetOutput(f *os.File) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = f
	l.logger = log.New(f, "", 0)
}

// SetPrefix 设置前缀
func (l *Logger) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}

// levelToString 日志级别转字符串
func levelToString(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// formatValue 格式化值
func formatValue(v interface{}) string {
	if v == nil {
		return "nil"
	}
	switch val := v.(type) {
	case string:
		return val
	case int, int64, float64:
		return itoaValue(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case time.Time:
		return val.Format("2006-01-02T15:04:05")
	default:
		return fmtValue(v)
	}
}

// itoaValue 整型转字符串
func itoaValue(v interface{}) string {
	switch val := v.(type) {
	case int:
		return intToString(int64(val))
	case int64:
		return intToString(val)
	case float64:
		return floatToString(val)
	default:
		return fmtValue(v)
	}
}

// intToString 整数转字符串
func intToString(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	var buf [20]byte
	pos := len(buf)
	for v > 0 {
		pos--
		buf[pos] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// floatToString 浮点转字符串
func floatToString(v float64) string {
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	intPart := int64(v)
	fracPart := v - float64(intPart)
	
	result := intToString(intPart)
	if fracPart > 0 {
		result += "."
		// 简单小数处理，最多6位
		for i := 0; i < 6; i++ {
			fracPart *= 10
			digit := int64(fracPart)
			result += intToString(digit)
			fracPart -= float64(digit)
			if fracPart < 0.0001 {
				break
			}
		}
	}
	if neg {
		return "-" + result
	}
	return result
}

// formatMsg 格式化消息
func formatMsg(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	// 简单格式化实现
	result := make([]byte, 0, len(format)+len(args)*20)
	argIdx := 0
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 's':
				if argIdx < len(args) {
					result = append(result, fmtValue(args[argIdx])...)
					argIdx++
				}
				i++
			case 'd':
				if argIdx < len(args) {
					result = append(result, fmtValue(args[argIdx])...)
					argIdx++
				}
				i++
			case 'v':
				if argIdx < len(args) {
					result = append(result, fmtValue(args[argIdx])...)
					argIdx++
				}
				i++
			case 'f':
				if argIdx < len(args) {
					result = append(result, fmtValue(args[argIdx])...)
					argIdx++
				}
				i++
			case 't':
				if argIdx < len(args) {
					result = append(result, fmtValue(args[argIdx])...)
					argIdx++
				}
				i++
			default:
				result = append(result, format[i])
			}
		} else {
			result = append(result, format[i])
		}
	}
	return string(result)
}

// fmtValue 将任意值转为字符串
func fmtValue(v interface{}) string {
	if v == nil {
		return "nil"
	}
	switch val := v.(type) {
	case string:
		return val
	case int:
		return intToString(int64(val))
	case int64:
		return intToString(val)
	case float64:
		return floatToString(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case time.Time:
		return val.Format("2006-01-02T15:04:05")
	case error:
		return val.Error()
	default:
		return "<complex>"
	}
}
