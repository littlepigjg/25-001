// Package handler 提供HTTP处理层
package handler

import (
	"net/http"
	"runtime"
	"time"

	"logalert/internal/config"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/errors"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	logService    *service.LogService
	alertService  *service.AlertService
	ruleService   *service.RuleService
	schedulerSvc  *service.SchedulerService
	config        *config.Config
	logger        *logger.Logger
	startTime     time.Time
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(
	logSvc *service.LogService,
	alertSvc *service.AlertService,
	ruleSvc *service.RuleService,
	scheduler *service.SchedulerService,
	cfg *config.Config,
	log *logger.Logger,
) *HealthHandler {
	return &HealthHandler{
		logService:   logSvc,
		alertService: alertSvc,
		ruleService:  ruleSvc,
		schedulerSvc: scheduler,
		config:       cfg,
		logger:       log,
		startTime:    time.Now(),
	}
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
	Version   string    `json:"version"`
	Runtime   RuntimeInfo `json:"runtime"`
	DB        DBStatus  `json:"db"`
}

// RuntimeInfo 运行时信息
type RuntimeInfo struct {
	GoVersion string `json:"go_version"`
	NumCPU    int    `json:"num_cpu"`
	NumGoroutine int `json:"num_goroutine"`
	MemAlloc  uint64 `json:"mem_alloc"`
	MemTotal  uint64 `json:"mem_total"`
}

// DBStatus 数据库状态
type DBStatus struct {
	TotalLogs    int64 `json:"total_logs"`
	TotalRules   int   `json:"total_rules"`
	ActiveAlerts int   `json:"active_alerts"`
}

// Healthy GET /health - 健康检查
func (h *HealthHandler) Healthy(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	stats, _ := h.logService.GetStoreStats(r.Context())
	if stats == nil {
		stats = &store.StoreStats{}
	}

	activeAlerts, _ := h.alertService.CountActiveAlerts(r.Context())
	if activeAlerts < 0 {
		activeAlerts = 0
	}

	response.Success(w, &HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    time.Since(h.startTime).String(),
		Version:   "1.0.0",
		Runtime: RuntimeInfo{
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			MemAlloc:     memStats.Alloc,
			MemTotal:     memStats.Sys,
		},
		DB: DBStatus{
			TotalLogs:    stats.TotalEntries,
			TotalRules:   stats.TotalRules,
			ActiveAlerts: activeAlerts,
		},
	})
}

// ErrorResponse 处理错误响应
func (h *HealthHandler) ErrorResponse(w http.ResponseWriter, err error) {
	if err == nil {
		response.Success(w, nil)
		return
	}
	if e, ok := err.(*errors.Error); ok {
		msg := errors.GetMessage(e.Code)
		if e.Code >= 40000 && e.Code < 40500 {
			response.ErrorWithCode(w, http.StatusOK, e.Code, e.Error())
		} else {
			response.ErrorWithCode(w, http.StatusInternalServerError, e.Code, msg)
		}
		return
	}
	response.InternalError(w, err)
}

// StatusMessage 获取状态消息
func (h *HealthHandler) StatusMessage(code int) string {
	msg := errors.GetMessage(code)
	if code >= 40000 && code < 40500 {
		return msg + " (client error)"
	}
	if code >= 50000 && code < 51000 {
		return msg + " (server error)"
	}
	return msg
}

// Ready GET /ready - 就绪检查
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	// 检查所有依赖是否就绪
	checks := make(map[string]string)
	allReady := true

	// 检查日志存储
	stats, err := h.logService.GetStoreStats(r.Context())
	if err != nil {
		checks["log_store"] = "not_ready: " + err.Error()
		allReady = false
	} else if stats == nil {
		checks["log_store"] = "ready"
	} else {
		checks["log_store"] = "ready"
	}

	// 检查规则存储
	ruleCount, err := h.ruleService.CountRules(r.Context())
	if err != nil {
		checks["rule_store"] = "not_ready: " + err.Error()
		allReady = false
	} else {
		checks["rule_store"] = "ready (count=" + intToStr(ruleCount) + ")"
	}

	// 检查告警存储
	alertCount, err := h.alertService.CountActiveAlerts(r.Context())
	if err != nil {
		checks["alert_store"] = "not_ready: " + err.Error()
		allReady = false
	} else {
		checks["alert_store"] = "ready (active=" + intToStr(alertCount) + ")"
	}

	response.Success(w, map[string]interface{}{
		"status":  "ready",
		"checks":  checks,
		"ready":   allReady,
		"services": map[string]bool{
			"scheduler": h.schedulerSvc.IsRunning(),
		},
	})
}

// Info GET /info - 服务信息
func (h *HealthHandler) Info(w http.ResponseWriter, r *http.Request) {
	response.Success(w, map[string]interface{}{
		"name":    "logalert",
		"version": "1.0.0",
		"description": "Real-time log aggregation and alert rule engine",
		"config": map[string]interface{}{
			"port":           h.config.Server.Port,
			"max_entries":    h.config.Storage.MaxEntries,
			"scan_interval":  h.config.Alert.ScanInterval.String(),
			"retention":      h.config.Storage.RetentionPeriod.String(),
		},
		"endpoints": []map[string]string{
			{"method": "POST", "path": "/api/logs", "desc": "Create log entry"},
			{"method": "POST", "path": "/api/logs/batch", "desc": "Create batch logs"},
			{"method": "GET", "path": "/api/logs", "desc": "Query logs"},
			{"method": "GET", "path": "/api/logs/{id}", "desc": "Get log by ID"},
			{"method": "DELETE", "path": "/api/logs/{id}", "desc": "Delete log"},
			{"method": "POST", "path": "/api/rules", "desc": "Create alert rule"},
			{"method": "GET", "path": "/api/rules", "desc": "List alert rules"},
			{"method": "GET", "path": "/api/rules/{id}", "desc": "Get rule by ID"},
			{"method": "PUT", "path": "/api/rules/{id}", "desc": "Update rule"},
			{"method": "DELETE", "path": "/api/rules/{id}", "desc": "Delete rule"},
			{"method": "POST", "path": "/api/rules/{id}/enable", "desc": "Enable rule"},
			{"method": "POST", "path": "/api/rules/{id}/disable", "desc": "Disable rule"},
			{"method": "GET", "path": "/api/alerts", "desc": "List alerts"},
			{"method": "GET", "path": "/api/alerts/{id}", "desc": "Get alert by ID"},
			{"method": "POST", "path": "/api/alerts/{id}/acknowledge", "desc": "Acknowledge alert"},
			{"method": "POST", "path": "/api/alerts/{id}/resolve", "desc": "Resolve alert"},
			{"method": "DELETE", "path": "/api/alerts/{id}", "desc": "Delete alert"},
			{"method": "GET", "path": "/api/stats", "desc": "Get statistics"},
			{"method": "GET", "path": "/api/stats/hourly", "desc": "Get hourly trends"},
			{"method": "GET", "path": "/api/stats/sources", "desc": "Get source stats"},
			{"method": "GET", "path": "/health", "desc": "Health check"},
			{"method": "GET", "path": "/ready", "desc": "Readiness check"},
			{"method": "GET", "path": "/info", "desc": "Service info"},
		},
	})
}

// intToStr 整数转字符串
func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
