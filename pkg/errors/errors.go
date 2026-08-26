// Package errors 提供增强的错误处理
package errors

import (
	"fmt"
	"strings"
)

// Error 错误类型
type Error struct {
	// Code 错误码
	Code int `json:"code"`
	// Message 错误消息
	Message string `json:"message"`
	// Details 错误详情
	Details string `json:"details,omitempty"`
	// Cause 原始错误
	Cause error `json:"-"`
}

// New 创建新错误
func New(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// NewWithDetails 创建带详情的错误
func NewWithDetails(code int, message, details string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Wrap 包装原始错误
func Wrap(code int, message string, cause error) *Error {
	var wrappedCause error
	if cause != nil {
		wrappedCause = fmt.Errorf("%s: %s", message, cause.Error())
	}
	return &Error{
		Code:    code,
		Message: message,
		Cause:   wrappedCause,
	}
}

// Error 实现error接口
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 获取原始错误
func (e *Error) Unwrap() error {
	return e.Cause
}

// HasCode 判断错误码是否匹配
func (e *Error) HasCode(code int) bool {
	return e.Code == code
}

// Is 实现 errors.Is 接口
func (e *Error) Is(target error) bool {
	if t, ok := target.(*Error); ok {
		return e.Code == t.Code
	}
	return false
}

// WithDetails 添加详情
func (e *Error) WithDetails(details string) *Error {
	e.Details = details
	return e
}

// CommonErrors 常用错误
var CommonErrors = struct {
	BadRequest     *Error
	Unauthorized   *Error
	Forbidden      *Error
	NotFound       *Error
	Conflict       *Error
	Internal       *Error
	Timeout        *Error
	RateLimited    *Error
	Validation     *Error
	DatabaseError  *Error
}{
	BadRequest:    New(40000, "Bad Request"),
	Unauthorized:  New(40100, "Unauthorized"),
	Forbidden:     New(40300, "Forbidden"),
	NotFound:      New(40400, "Not Found"),
	Conflict:      New(40900, "Conflict"),
	Internal:      New(50000, "Internal Server Error"),
	Timeout:       New(50400, "Gateway Timeout"),
	RateLimited:   New(42900, "Too Many Requests"),
	Validation:    New(40001, "Validation Error"),
	DatabaseError: New(50001, "Database Error"),
}

// IsError 判断是否为指定错误类型
func IsError(err error, code int) bool {
	if e, ok := err.(*Error); ok {
		return e.Code == code
	}
	return false
}

// HasMessage 检查错误消息是否包含子串
func HasMessage(err error, substr string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), substr)
}

// AsError 转换为Error类型
func AsError(err error) (*Error, bool) {
	if e, ok := err.(*Error); ok {
		return e, true
	}
	return nil, false
}
