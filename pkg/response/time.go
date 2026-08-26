// Package response 提供统一响应格式
package response

import "time"

// getTimeString 获取当前时间字符串
func getTimeString() string {
	return time.Now().Format(time.RFC3339)
}

// ErrorResponse 错误响应详情
type ErrorResponse struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// ValidationErrorResponse 校验错误响应
type ValidationErrorResponse struct {
	Errors []ErrorResponse `json:"errors"`
}

// NewValidationErrorResponse 创建校验错误响应
func NewValidationErrorResponse(errors []ErrorResponse) *ValidationErrorResponse {
	return &ValidationErrorResponse{Errors: errors}
}
