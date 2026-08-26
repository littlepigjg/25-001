// Package response 提供统一响应格式
package response

import (
	"encoding/json"
	"net/http"
	"logalert/pkg/logger"
)

// Response 统一响应结构
type Response struct {
	// Code 响应码，0表示成功
	Code int `json:"code"`
	// Message 响应消息
	Message string `json:"message"`
	// Data 响应数据
	Data interface{} `json:"data,omitempty"`
	// Timestamp 时间戳
	Timestamp string `json:"timestamp"`
	// TraceID 链路追踪ID
	TraceID string `json:"trace_id,omitempty"`
}

// Success 返回成功响应
func Success(w http.ResponseWriter, data interface{}) {
	resp := &Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: nowString(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// SuccessMessage 返回带消息的成功响应
func SuccessMessage(w http.ResponseWriter, message string, data interface{}) {
	resp := &Response{
		Code:      0,
		Message:   message,
		Data:      data,
		Timestamp: nowString(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// Created 返回创建成功响应
func Created(w http.ResponseWriter, data interface{}) {
	resp := &Response{
		Code:      0,
		Message:   "created",
		Data:      data,
		Timestamp: nowString(),
	}
	writeJSON(w, http.StatusCreated, resp)
}

// Updated 返回更新成功响应
func Updated(w http.ResponseWriter, data interface{}) {
	resp := &Response{
		Code:      0,
		Message:   "updated",
		Data:      data,
		Timestamp: nowString(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// Deleted 返回删除成功响应
func Deleted(w http.ResponseWriter) {
	resp := &Response{
		Code:      0,
		Message:   "deleted",
		Timestamp: nowString(),
	}
	writeJSON(w, http.StatusNoContent, resp)
}

// Error 返回错误响应
func Error(w http.ResponseWriter, statusCode int, err error) {
	resp := &Response{
		Code:      statusCode,
		Message:   err.Error(),
		Timestamp: nowString(),
	}
	writeJSON(w, statusCode, resp)
}

// ErrorCode 返回带错误码的错误响应
func ErrorCode(w http.ResponseWriter, statusCode int, code int, message string) {
	resp := &Response{
		Code:      code,
		Message:   message,
		Timestamp: nowString(),
	}
	writeJSON(w, statusCode, resp)
}

// BadRequest 返回400错误
func BadRequest(w http.ResponseWriter, err error) {
	Error(w, http.StatusBadRequest, err)
}

// Unauthorized 返回401错误
func Unauthorized(w http.ResponseWriter, err error) {
	Error(w, http.StatusUnauthorized, err)
}

// NotFound 返回404错误
func NotFound(w http.ResponseWriter, err error) {
	Error(w, http.StatusNotFound, err)
}

// Conflict 返回409错误
func Conflict(w http.ResponseWriter, err error) {
	Error(w, http.StatusConflict, err)
}

// InternalError 返回500错误
func InternalError(w http.ResponseWriter, err error) {
	Error(w, http.StatusInternalServerError, err)
}

// writeJSON 写入JSON响应
func writeJSON(w http.ResponseWriter, statusCode int, resp *Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		logger.New().Errorf("failed to write response: %v", err)
	}
}

// nowString 返回当前时间字符串
func nowString() string {
	return timeNow()
}

// timeNow 返回当前时间的ISO格式字符串
func timeNow() string {
	return getTimeString()
}
