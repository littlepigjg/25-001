// Package handler 提供HTTP处理层
package handler

import (
	"net/http"
	"strconv"
	"time"

	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// StatsHandler 统计处理器
type StatsHandler struct {
	service *service.StatsService
	logger  *logger.Logger
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(svc *service.StatsService, log *logger.Logger) *StatsHandler {
	return &StatsHandler{
		service: svc,
		logger:  log,
	}
}

// GetStatistics GET /api/stats - 获取日志统计
func (h *StatsHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	startStr := r.URL.Query().Get("start_time")
	endStr := r.URL.Query().Get("end_time")

	var startTime, endTime time.Time

	if startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			response.BadRequest(w, model.NewValidationError("start_time", "invalid time format"))
			return
		}
		startTime = t
	}

	if endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			response.BadRequest(w, model.NewValidationError("end_time", "invalid time format"))
			return
		}
		endTime = t
	}

	ctx := r.Context()
	stats, err := h.service.GetStatistics(ctx, startTime, endTime)
	if err != nil {
		h.logger.Errorf("failed to get statistics: %v", err)
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, stats)
}

// GetHourlyTrends GET /api/stats/hourly - 获取每小时趋势
func (h *StatsHandler) GetHourlyTrends(w http.ResponseWriter, r *http.Request) {
	hours := 24
	if hStr := r.URL.Query().Get("hours"); hStr != "" {
		if n, err := strconv.Atoi(hStr); err == nil && n > 0 && n <= 168 {
			hours = n
		}
	}

	ctx := r.Context()
	trends, err := h.service.GetHourlyTrends(ctx, hours)
	if err != nil {
		h.logger.Errorf("failed to get hourly trends: %v", err)
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"hours": hours,
		"items": trends,
	})
}

// GetSourceStats GET /api/stats/sources - 获取来源统计
func (h *StatsHandler) GetSourceStats(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if n, err := strconv.Atoi(lStr); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	ctx := r.Context()
	sourceStats, err := h.service.GetSourceStats(ctx, limit)
	if err != nil {
		h.logger.Errorf("failed to get source stats: %v", err)
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"total_sources": len(sourceStats),
		"items":         sourceStats,
	})
}

// GetDailyReport GET /api/stats/daily - 获取日报
func (h *StatsHandler) GetDailyReport(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	date := time.Now()
	if dateStr != "" {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			response.BadRequest(w, model.NewValidationError("date", "invalid date format, use YYYY-MM-DD"))
			return
		}
		date = t
	}

	ctx := r.Context()
	report, err := h.service.GetDailyReport(ctx, date)
	if err != nil {
		h.logger.Errorf("failed to get daily report: %v", err)
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, report)
}

// GetErrorRateTrend GET /api/stats/error-rate - 获取错误率趋势
func (h *StatsHandler) GetErrorRateTrend(w http.ResponseWriter, r *http.Request) {
	hours := 24
	if hStr := r.URL.Query().Get("hours"); hStr != "" {
		if n, err := strconv.Atoi(hStr); err == nil && n > 0 && n <= 168 {
			hours = n
		}
	}

	ctx := r.Context()
	trends, err := h.service.GetErrorRateTrend(ctx, hours)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"hours": hours,
		"items": trends,
	})
}
