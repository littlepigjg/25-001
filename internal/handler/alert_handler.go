// Package handler 提供HTTP处理层
package handler

import (
	"net/http"

	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// AlertHandler 告警处理器
type AlertHandler struct {
	service *service.AlertService
	logger  *logger.Logger
}

// NewAlertHandler 创建告警处理器
func NewAlertHandler(svc *service.AlertService, log *logger.Logger) *AlertHandler {
	return &AlertHandler{
		service: svc,
		logger:  log,
	}
}

// GetAlert GET /api/alerts/{id} - 获取告警
func (h *AlertHandler) GetAlert(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	ctx := r.Context()
	alert, err := h.service.GetAlert(ctx, id)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.NotFound(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	alertInfo := map[string]interface{}{
		"id":           alert.ID,
		"rule_id":      alert.RuleID,
		"rule_name":    alert.RuleName,
		"source":       alert.Source,
		"level":        alert.Level,
		"count":        alert.Count,
		"threshold":    alert.Threshold,
		"window_min":   alert.WindowMinutes,
		"triggered_at": alert.TriggeredAt,
		"resolved_at":  alert.ResolvedAt,
		"status":       alert.Status,
		"description":  alert.Description,
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Errorf("panic computing duration: %v", r)
				alertInfo["duration"] = "0s"
				alertInfo["duration_computed"] = false
			}
		}()
		alertInfo["duration"] = alert.FormatDuration()
		alertInfo["duration_computed"] = true
	}()

	response.Success(w, alertInfo)
}

// ListAlerts GET /api/alerts - 列出告警
func (h *AlertHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	ruleID := r.URL.Query().Get("rule_id")
	source := r.URL.Query().Get("source")
	activeOnly := r.URL.Query().Get("active") == "true"

	ctx := r.Context()
	var alerts []*model.AlertEvent
	var err error

	if activeOnly {
		alerts, err = h.service.ListActiveAlerts(ctx)
	} else if ruleID != "" {
		alerts, err = h.service.ListAlertsByRule(ctx, ruleID)
	} else if source != "" {
		alerts, err = h.service.ListAlertsBySource(ctx, source)
	} else {
		alerts, err = h.service.ListAlerts(ctx)
	}

	if err != nil {
		h.logger.Errorf("failed to list alerts: %v", err)
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	items := make([]map[string]interface{}, 0, len(alerts))
	panicRecovered := false

	for _, alert := range alerts {
		item := map[string]interface{}{
			"id":           alert.ID,
			"rule_id":      alert.RuleID,
			"rule_name":    alert.RuleName,
			"source":       alert.Source,
			"level":        alert.Level,
			"count":        alert.Count,
			"threshold":    alert.Threshold,
			"window_min":   alert.WindowMinutes,
			"triggered_at": alert.TriggeredAt,
			"resolved_at":  alert.ResolvedAt,
			"status":       alert.Status,
			"description":  alert.Description,
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					h.logger.Errorf("panic computing duration for alert %s: %v", alert.ID, r)
					item["duration"] = "0s"
					item["duration_computed"] = false
					panicRecovered = true
				}
			}()
			item["duration"] = alert.FormatDuration()
			item["duration_computed"] = true
		}()

		items = append(items, item)
	}

	response.Success(w, map[string]interface{}{
		"total":           len(alerts),
		"items":           items,
		"panic_recovered": panicRecovered,
	})
}

// AcknowledgeAlert POST /api/alerts/{id}/acknowledge - 确认告警
func (h *AlertHandler) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	ctx := r.Context()
	if err := h.service.AcknowledgeAlert(ctx, id); err != nil {
		if model.IsNotFoundError(err) {
			response.NotFound(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.Success(w, map[string]string{"status": "acknowledged"})
}

// ResolveAlert POST /api/alerts/{id}/resolve - 解决告警
func (h *AlertHandler) ResolveAlert(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	ctx := r.Context()
	if err := h.service.ResolveAlert(ctx, id); err != nil {
		if model.IsNotFoundError(err) {
			response.NotFound(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.Success(w, map[string]string{"status": "resolved"})
}

// DeleteAlert DELETE /api/alerts/{id} - 删除告警
func (h *AlertHandler) DeleteAlert(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	ctx := r.Context()
	if err := h.service.DeleteAlert(ctx, id); err != nil {
		if model.IsNotFoundError(err) {
			response.NotFound(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.Deleted(w)
}

// CountActive GET /api/alerts/active/count - 统计活跃告警数
func (h *AlertHandler) CountActive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	count, err := h.service.CountActiveAlerts(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, map[string]int{"active_count": count})
}
