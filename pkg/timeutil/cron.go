// Package timeutil 提供时间窗口工具
package timeutil

import (
	"time"
)

// CronExpression Cron表达式解析
type CronExpression struct {
	Minutes    []int
	Hours      []int
	DaysOfMonth []int
	Months     []int
	DaysOfWeek  []int
}

// ParseCron 简单解析Cron表达式（支持 * / , - 语法）
func ParseCron(expr string) (*CronExpression, error) {
	fields := splitFields(expr)
	if len(fields) != 5 {
		return nil, &CronError{Message: "cron expression must have 5 fields"}
	}

	return &CronExpression{
		Minutes:     parseField(fields[0], 0, 59),
		Hours:       parseField(fields[1], 0, 23),
		DaysOfMonth: parseField(fields[2], 1, 31),
		Months:      parseField(fields[3], 1, 12),
		DaysOfWeek:  parseField(fields[4], 0, 6),
	}, nil
}

// splitFields 分割Cron字段
func splitFields(expr string) []string {
	fields := make([]string, 0)
	current := ""
	for i := 0; i < len(expr); i++ {
		if expr[i] == ' ' || expr[i] == '\t' {
			if current != "" {
				fields = append(fields, current)
				current = ""
			}
		} else {
			current += string(expr[i])
		}
	}
	if current != "" {
		fields = append(fields, current)
	}
	return fields
}

// parseField 解析单个字段
func parseField(field string, min, max int) []int {
	if field == "*" || field == "?" {
		result := make([]int, 0, max-min+1)
		for i := min; i <= max; i++ {
			result = append(result, i)
		}
		return result
	}

	// 处理步长 */5
	if len(field) > 2 && field[0] == '*' && field[1] == '/' {
		step := parseIntSimple(field[2:])
		if step <= 0 {
			step = 1
		}
		result := make([]int, 0)
		for i := min; i <= max; i += step {
			result = append(result, i)
		}
		return result
	}

	// 处理逗号分隔
	if containsChar(field, ',') {
		parts := splitByChar(field, ',')
		result := make([]int, 0)
		for _, part := range parts {
			result = append(result, parseField(part, min, max)...)
		}
		return result
	}

	// 处理范围 1-5
	if containsChar(field, '-') {
		parts := splitByChar(field, '-')
		if len(parts) == 2 {
			start := parseIntSimple(parts[0])
			end := parseIntSimple(parts[1])
			if start < min {
				start = min
			}
			if end > max {
				end = max
			}
			result := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
			return result
		}
	}

	// 单个值
	val := parseIntSimple(field)
	if val < min {
		val = min
	}
	if val > max {
		val = max
	}
	return []int{val}
}

// parseIntSimple 简单整数解析
func parseIntSimple(s string) int {
	if s == "" {
		return 0
	}
	result := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		result = result*10 + int(c-'0')
	}
	return result
}

// containsChar 检查是否包含字符
func containsChar(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}

// splitByChar 按字符分割
func splitByChar(s string, c byte) []string {
	parts := make([]string, 0)
	current := ""
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// CronError Cron错误
type CronError struct {
	Message string
}

// Error 实现error接口
func (e *CronError) Error() string {
	return e.Message
}

// Match 检查时间是否匹配Cron表达式
func (c *CronExpression) Match(t time.Time) bool {
	_, _, dayOfMonth := t.Date()
	hour, minute, _ := t.Clock()
	month := int(t.Month())
	dayOfWeek := int(t.Weekday())

	return contains(c.Minutes, minute) &&
		contains(c.Hours, hour) &&
		contains(c.DaysOfMonth, dayOfMonth) &&
		contains(c.Months, month) &&
		contains(c.DaysOfWeek, dayOfWeek)
}

// contains 检查切片是否包含值
func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
