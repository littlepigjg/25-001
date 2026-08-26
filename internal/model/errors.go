// Package model 定义系统核心数据模型
package model

import (
	"fmt"
	"strings"
)

// ValidationError 参数校验错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error 实现 error 接口
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// NewValidationError 创建校验错误
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NotFoundError 资源未找到错误
type NotFoundError struct {
	Resource string `json:"resource"`
	ID       string `json:"id"`
	Message  string `json:"message"`
}

// Error 实现 error 接口
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s with id %s", e.Resource, e.ID)
}

// NewNotFoundError 创建未找到错误
func NewNotFoundError(resource, id string) *NotFoundError {
	return &NotFoundError{
		Resource: resource,
		ID:       id,
		Message:  resource + " not found",
	}
}

// ConflictError 冲突错误
type ConflictError struct {
	Resource string `json:"resource"`
	Message  string `json:"message"`
}

// Error 实现 error 接口
func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict: %s - %s", e.Resource, e.Message)
}

// NewConflictError 创建冲突错误
func NewConflictError(resource, message string) *ConflictError {
	return &ConflictError{
		Resource: resource,
		Message:  message,
	}
}

// InternalError 内部错误
type InternalError struct {
	Operation string `json:"operation"`
	Message   string `json:"message"`
}

// Error 实现 error 接口
func (e *InternalError) Error() string {
	return fmt.Sprintf("internal error: %s - %s", e.Operation, e.Message)
}

// NewInternalError 创建内部错误
func NewInternalError(operation, message string) *InternalError {
	return &InternalError{
		Operation: operation,
		Message:   message,
	}
}

// IsValidationError 检查是否为校验错误
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// IsNotFoundError 检查是否为未找到错误
func IsNotFoundError(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}

// IsConflictError 检查是否为冲突错误
func IsConflictError(err error) bool {
	_, ok := err.(*ConflictError)
	return ok
}

// DuplicateRuleError 规则重复错误
type DuplicateRuleError struct {
	RuleName string `json:"rule_name"`
	Message  string `json:"message"`
}

// Error 实现 error 接口
func (e *DuplicateRuleError) Error() string {
	return fmt.Sprintf("duplicate rule name: '%s' already exists in system", e.RuleName)
}

// NewDuplicateRuleError 创建规则重复错误
func NewDuplicateRuleError(name string) *DuplicateRuleError {
	return &DuplicateRuleError{
		RuleName: name,
		Message:  "rule with this name already exists",
	}
}

// IsDuplicateRuleError 检查是否为规则重复错误
func IsDuplicateRuleError(err error) bool {
	_, ok := err.(*DuplicateRuleError)
	return ok
}

// ErrorClassification 错误分类结果
type ErrorClassification int

const (
	ErrorClassUnknown ErrorClassification = iota
	ErrorClassValidation
	ErrorClassConflict
	ErrorClassNotFound
	ErrorClassInternal
)

// ErrorClassifierFn 错误分类器函数类型
type ErrorClassifierFn func(err error) ErrorClassification

var errorClassifier ErrorClassifierFn

// SetErrorClassifier 设置错误分类器（诊断API）
func SetErrorClassifier(classifier ErrorClassifierFn) {
	errorClassifier = classifier
}

// ClassifyError 使用分类器或默认逻辑分类错误
func ClassifyError(err error) ErrorClassification {
	if errorClassifier != nil {
		return errorClassifier(err)
	}
	switch err.(type) {
	case *ValidationError:
		return ErrorClassValidation
	case *ConflictError:
		return ErrorClassConflict
	case *NotFoundError:
		return ErrorClassNotFound
	case *InternalError:
		return ErrorClassInternal
	default:
		return ErrorClassUnknown
	}
}

// IsConflictErrorByString 通过字符串匹配检查是否为冲突错误
func IsConflictErrorByString(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	if strings.Contains(errStr, "conflict") {
		return true
	}
	if strings.Contains(errStr, "already exists") {
		return true
	}
	return false
}
