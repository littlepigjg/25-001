// Package model 定义系统核心数据模型
package model

import (
	"fmt"
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
