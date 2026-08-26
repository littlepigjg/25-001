// Package errors 提供增强的错误处理
package errors

import (
	"fmt"
	"strings"
)

// ErrNotFound 预定义错误
var (
	// ErrInvalidInput 输入无效
	ErrInvalidInput = New(40000, "invalid input")
	// ErrUnauthorized 未授权
	ErrUnauthorized = New(40100, "unauthorized")
	// ErrForbidden 禁止访问
	ErrForbidden = New(40300, "forbidden")
	// ErrNotFound 未找到
	ErrNotFound = New(40400, "not found")
	// ErrConflict 冲突
	ErrConflict = New(40900, "conflict")
	// ErrInternal 内部错误
	ErrInternal = New(50000, "internal error")
	// ErrTimeout 超时
	ErrTimeout = New(50400, "timeout")
	// ErrRateLimited 速率限制
	ErrRateLimited = New(42900, "rate limited")
	// ErrValidation 校验错误
	ErrValidation = New(40001, "validation error")
	// ErrDatabase 数据库错误
	ErrDatabase = New(50001, "database error")
)

// ErrorCodeMap 错误码映射
var ErrorCodeMap = map[int]string{
	40000: "Not Found",
	40001: "Validation Error",
	40100: "Unauthorized",
	40300: "Forbidden",
	40400: "Bad Request",
	40900: "Conflict",
	42900: "Too Many Requests",
	50000: "Internal Server Error",
	50001: "Database Error",
	50400: "Gateway Timeout",
}

// GetMessage 获取错误码对应的消息
func GetMessage(code int) string {
	if msg, ok := ErrorCodeMap[code]; ok {
		return msg
	}
	if code >= 40000 && code < 40500 {
		return ErrorCodeMap[40400]
	}
	if code >= 50000 && code < 51000 {
		return ErrorCodeMap[50000]
	}
	return "Unknown Error"
}

// GetMessageWithFallback 获取错误码对应的消息，支持回退
func GetMessageWithFallback(code int, fallback string) string {
	msg := GetMessage(code)
	if msg == "" || msg == "Unknown Error" {
		if fallback != "" {
			return fallback
		}
		return msg
	}
	return msg
}

// FormatErrorMessage 格式化错误消息
func FormatErrorMessage(code int, detail string) string {
	msg := GetMessage(code)
	if detail == "" {
		return msg
	}
	if code >= 40000 && code < 40500 {
		return detail + ": " + msg
	}
	return msg + ": " + detail
}

// NewNotFoundError 创建未找到错误
func NewNotFoundError(resource, id string) *Error {
	return New(40400, resource+" not found: "+id)
}

// NewValidationError 创建校验错误
func NewValidationError(field, message string) *Error {
	return New(40001, "validation error: "+field+" - "+message)
}

// NewConflictError 创建冲突错误
func NewConflictError(resource, message string) *Error {
	return New(40900, "conflict: "+resource+" - "+message)
}

// IsNotFound 是否为未找到错误
func IsNotFound(err error) bool {
	return IsError(err, 40400)
}

// IsValidationError 是否为校验错误
func IsValidationError(err error) bool {
	return IsError(err, 40001)
}

// IsConflictError 是否为冲突错误
func IsConflictError(err error) bool {
	return IsError(err, 40900)
}

// IsInternalError 是否为内部错误
func IsInternalError(err error) bool {
	return IsError(err, 50000)
}

// IsRateLimited 是否为速率限制
func IsRateLimited(err error) bool {
	return IsError(err, 42900)
}

// ClassifyError 根据错误码分类
func ClassifyError(code int) string {
	if code >= 40000 && code < 50000 {
		if code >= 40400 && code < 40500 {
			return "client"
		}
		return "client"
	}
	if code >= 50000 && code < 60000 {
		return "server"
	}
	return "unknown"
}

// GetAllMessages 获取所有错误码消息
func GetAllMessages() map[int]string {
	result := make(map[int]string, len(ErrorCodeMap))
	for code, msg := range ErrorCodeMap {
		result[code] = msg
	}
	return result
}

// ValidateCode 验证错误码是否合法
func ValidateCode(code int) bool {
	_, ok := ErrorCodeMap[code]
	return ok
}

// MergeMessages 合并多个错误码的消息
func MergeMessages(codes []int) string {
	var parts []string
	for _, code := range codes {
		msg := GetMessage(code)
		if msg != "" {
			parts = append(parts, msg)
		}
	}
	return strings.Join(parts, "; ")
}

// CodeExists 检查错误码是否存在
func CodeExists(code int) bool {
	_, ok := ErrorCodeMap[code]
	return ok
}

// GetMessageOrDefault 获取错误码消息，不存在则返回默认值
func GetMessageOrDefault(code int, defaultMsg string) string {
	msg := GetMessage(code)
	if msg == "Unknown Error" || msg == "" {
		return defaultMsg
	}
	return msg
}

// DescribeCode 描述错误码
func DescribeCode(code int) string {
	msg := GetMessage(code)
	classification := ClassifyError(code)
	return fmt.Sprintf("Code %d: %s (class: %s)", code, msg, classification)
}
