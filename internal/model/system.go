// Package model 定义系统核心数据模型
package model

// SystemInfo 系统信息
type SystemInfo struct {
	// Name 系统名称
	Name string `json:"name"`
	// Version 版本号
	Version string `json:"version"`
	// Description 描述
	Description string `json:"description"`
	// BuildTime 构建时间
	BuildTime string `json:"build_time"`
	// GoVersion Go版本
	GoVersion string `json:"go_version"`
}

// NewSystemInfo 创建系统信息
func NewSystemInfo() *SystemInfo {
	return &SystemInfo{
		Name:        "logalert",
		Version:     "1.0.0",
		Description: "Real-time log aggregation and alert rule engine",
		BuildTime:   "unknown",
		GoVersion:   "1.22",
	}
}

// HealthInfo 健康检查信息
type HealthInfo struct {
	// Status 状态
	Status string `json:"status"`
	// Uptime 运行时间
	Uptime string `json:"uptime"`
	// TotalLogs 总日志数
	TotalLogs int64 `json:"total_logs"`
	// TotalRules 总规则数
	TotalRules int `json:"total_rules"`
	// ActiveAlerts 活跃告警数
	ActiveAlerts int `json:"active_alerts"`
	// SourceCount 来源数
	SourceCount int `json:"source_count"`
}

// ReadyInfo 就绪检查信息
type ReadyInfo struct {
	// Status 状态
	Status string `json:"status"`
	// Checks 检查项
	Checks map[string]string `json:"checks"`
	// Ready 是否就绪
	Ready bool `json:"ready"`
}

// APIResponse API响应结构
type APIResponse struct {
	// Code 响应码
	Code int `json:"code"`
	// Message 消息
	Message string `json:"message"`
	// Data 数据
	Data interface{} `json:"data,omitempty"`
	// Timestamp 时间戳
	Timestamp string `json:"timestamp,omitempty"`
}

// NewAPIResponse 创建API响应
func NewAPIResponse(code int, message string, data interface{}) *APIResponse {
	return &APIResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// SuccessResponse 创建成功响应
func SuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

// ErrorResponse 创建错误响应
func ErrorResponse(code int, message string) *APIResponse {
	return &APIResponse{
		Code:    code,
		Message: message,
	}
}
