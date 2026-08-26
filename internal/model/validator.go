// Package model 定义系统核心数据模型
package model

// ValidationRule 校验规则
type ValidationRule struct {
	Field    string      `json:"field"`
	Type     string      `json:"type"`
	Required bool        `json:"required"`
	Min      interface{} `json:"min,omitempty"`
	Max      interface{} `json:"max,omitempty"`
	Pattern  string      `json:"pattern,omitempty"`
	Message  string      `json:"message,omitempty"`
}

// LogEntryValidator 日志条目校验器
type LogEntryValidator struct {
	rules []ValidationRule
}

// NewLogEntryValidator 创建日志校验器
func NewLogEntryValidator() *LogEntryValidator {
	return &LogEntryValidator{
		rules: []ValidationRule{
			{Field: "source", Type: "string", Required: true, Message: "source is required"},
			{Field: "message", Type: "string", Required: true, Min: 1, Max: 10000, Message: "message must be between 1 and 10000 chars"},
			{Field: "level", Type: "enum", Required: true, Message: "level must be one of INFO/WARN/ERROR/DEBUG/FATAL"},
		},
	}
}

// Validate 校验日志条目
func (v *LogEntryValidator) Validate(entry *LogEntry) error {
	if entry == nil {
		return NewValidationError("entry", "entry cannot be nil")
	}
	if entry.Source == "" {
		return NewValidationError("source", "source is required")
	}
	if len(entry.Message) == 0 {
		return NewValidationError("message", "message is required")
	}
	if len(entry.Message) > 10000 {
		return NewValidationError("message", "message cannot exceed 10000 characters")
	}
	if !entry.Level.Valid() {
		return NewValidationError("level", "invalid log level")
	}
	return nil
}

// RuleValidator 规则校验器
type RuleValidator struct {
	maxWindow    int
	minThreshold int
}

// NewRuleValidator 创建规则校验器
func NewRuleValidator(maxWindow, minThreshold int) *RuleValidator {
	return &RuleValidator{
		maxWindow:    maxWindow,
		minThreshold: minThreshold,
	}
}

// Validate 校验规则
func (v *RuleValidator) Validate(rule *AlertRule) error {
	if rule == nil {
		return NewValidationError("rule", "rule cannot be nil")
	}
	if rule.Name == "" {
		return NewValidationError("name", "name is required")
	}
	if !rule.Level.Valid() {
		return NewValidationError("level", "invalid level")
	}
	if rule.WindowMinutes <= 0 {
		return NewValidationError("window_minutes", "window must be positive")
	}
	if v.maxWindow > 0 && rule.WindowMinutes > v.maxWindow {
		return NewValidationError("window_minutes", "window exceeds maximum")
	}
	if rule.Threshold < v.minThreshold {
		return NewValidationError("threshold", "threshold below minimum")
	}
	return nil
}
