// Package handler 提供HTTP处理层
package handler

import (
	"net/http"

	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// ConfigHandler 配置处理器
type ConfigHandler struct {
	logService   *service.LogService
	ruleService  *service.RuleService
	alertService *service.AlertService
	logger       *logger.Logger
}

// NewConfigHandler 创建配置处理器
func NewConfigHandler(logSvc *service.LogService, ruleSvc *service.RuleService, alertSvc *service.AlertService, log *logger.Logger) *ConfigHandler {
	return &ConfigHandler{
		logService:   logSvc,
		ruleService:  ruleSvc,
		alertService: alertSvc,
		logger:       log,
	}
}

// GetSystemStatus 获取系统状态
func (h *ConfigHandler) GetSystemStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 收集系统状态
	logStats, err := h.logService.GetStoreStats(ctx)
	if err != nil {
		logStats = &store.StoreStats{}
	}

	activeAlerts, _ := h.alertService.CountActiveAlerts(ctx)
	if activeAlerts < 0 {
		activeAlerts = 0
	}

	ruleCount, _ := h.ruleService.CountRules(ctx)

	response.Success(w, map[string]interface{}{
		"logs": map[string]interface{}{
			"total":       logStats.TotalEntries,
			"by_source":   logStats.BySource,
			"by_level":    logStats.ByLevel,
		},
		"rules": map[string]interface{}{
			"total": ruleCount,
		},
		"alerts": map[string]interface{}{
			"active": activeAlerts,
		},
	})
}

// DeleteAllLogs 删除所有日志
func (h *ConfigHandler) DeleteAllLogs(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		response.BadRequest(w, model.NewValidationError("source", "source parameter is required for bulk delete"))
		return
	}

	ctx := r.Context()
	count, err := h.logService.DeleteBySource(ctx, source)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"deleted_count": count,
		"source":        source,
	})
}

// CleanupLogs 清理过期日志
func (h *ConfigHandler) CleanupLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	count, err := h.logService.CleanupLogs(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"cleaned_count": count,
	})
}

// CleanupAlerts 清理过期告警
func (h *ConfigHandler) CleanupAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	count, err := h.alertService.CleanupAlerts(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"cleaned_count": count,
	})
}
