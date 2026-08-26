// Package handler 提供HTTP处理层
package handler

import (
	"net/http"
	"time"

	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// QueryHandler 查询处理器
type QueryHandler struct {
	logService  *service.LogService
	statsService *service.StatsService
	logger      *logger.Logger
}

// NewQueryHandler 创建查询处理器
func NewQueryHandler(logSvc *service.LogService, statsSvc *service.StatsService, log *logger.Logger) *QueryHandler {
	return &QueryHandler{
		logService:  logSvc,
		statsService: statsSvc,
		logger:      log,
	}
}

// SearchLogs 高级搜索
func (h *QueryHandler) SearchLogs(w http.ResponseWriter, r *http.Request) {
	query := model.DefaultLogQuery()
	params := r.URL.Query()

	// 构建搜索条件
	if keyword := params.Get("keyword"); keyword != "" {
		query.Keywords = append(query.Keywords, keyword)
	}
	if keywords := params["keyword"]; len(keywords) > 0 {
		query.Keywords = keywords
	}
	if source := params.Get("source"); source != "" {
		query.Source = source
	}
	if level := params.Get("level"); level != "" {
		if l, err := model.ParseLogLevel(level); err == nil {
			query.Level = l
		}
	}
	if startTime := params.Get("from"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			query.StartTime = &t
		}
	}
	if endTime := params.Get("to"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			query.EndTime = &t
		}
	}
	if limit := params.Get("limit"); limit != "" {
		if n, err := parseInt(limit); err == nil {
			query.Limit = n
		}
	}
	if offset := params.Get("offset"); offset != "" {
		if n, err := parseInt(offset); err == nil {
			query.Offset = n
		}
	}

	ctx := r.Context()
	result, err := h.logService.QueryLogs(ctx, query)
	if err != nil {
		h.logger.Errorf("search failed: %v", err)
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, result)
}

// GetLogByID 获取单条日志
func (h *QueryHandler) GetLogByID(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	ctx := r.Context()
	entry, err := h.logService.GetLog(ctx, id)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.NotFound(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.Success(w, entry)
}

// GetLogsBySource 按来源获取日志
func (h *QueryHandler) GetLogsBySource(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		response.BadRequest(w, model.NewValidationError("source", "source is required"))
		return
	}

	query := &model.LogQuery{
		Source: source,
		Limit:  100,
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if n, err := parseInt(limit); err == nil {
			query.Limit = n
		}
	}

	ctx := r.Context()
	result, err := h.logService.QueryLogs(ctx, query)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, result)
}

// GetErrorLogs 获取错误日志
func (h *QueryHandler) GetErrorLogs(w http.ResponseWriter, r *http.Request) {
	query := &model.LogQuery{
		Level: model.LevelError,
		Limit:  100,
	}

	if source := r.URL.Query().Get("source"); source != "" {
		query.Source = source
	}
	if hours := r.URL.Query().Get("hours"); hours != "" {
		if n, err := parseInt(hours); err == nil && n > 0 {
			startTime := time.Now().Add(-time.Duration(n) * time.Hour)
			query.StartTime = &startTime
		}
	}

	ctx := r.Context()
	result, err := h.logService.QueryLogs(ctx, query)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, result)
}

// GetRecentLogs 获取最近日志
func (h *QueryHandler) GetRecentLogs(w http.ResponseWriter, r *http.Request) {
	minutes := 60
	if mStr := r.URL.Query().Get("minutes"); mStr != "" {
		if n, err := parseInt(mStr); err == nil && n > 0 {
			minutes = n
		}
	}

	query := &model.LogQuery{
		Limit: 100,
	}

	startTime := time.Now().Add(-time.Duration(minutes) * time.Minute)
	query.StartTime = &startTime

	ctx := r.Context()
	result, err := h.logService.QueryLogs(ctx, query)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, result)
}
