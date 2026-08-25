// Package handler 提供HTTP处理层
package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// LogHandler 日志处理器
type LogHandler struct {
	service *service.LogService
	logger  *logger.Logger
}

// NewLogHandler 创建日志处理器
func NewLogHandler(svc *service.LogService, log *logger.Logger) *LogHandler {
	return &LogHandler{
		service: svc,
		logger:  log,
	}
}

// CreateLogRequest 创建日志请求
type CreateLogRequest struct {
	Level    model.LogLevel  `json:"level"`
	Source   string          `json:"source"`
	Message  string          `json:"message"`
	Keywords []string        `json:"keywords,omitempty"`
	TraceID  string          `json:"trace_id,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// BatchLogRequest 批量日志请求
type BatchLogRequest struct {
	Entries []CreateLogRequest `json:"entries"`
}

// CreateLog POST /api/logs - 创建日志
func (h *LogHandler) CreateLog(w http.ResponseWriter, r *http.Request) {
	var req CreateLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, model.NewValidationError("body", "invalid JSON: "+err.Error()))
		return
	}

	if req.Level == "" {
		req.Level = model.LevelInfo
	}
	if !req.Level.Valid() {
		response.BadRequest(w, model.NewValidationError("level", "invalid log level"))
		return
	}

	ctx := r.Context()

	if err := validateLogPayload(ctx, &req); err != nil {
		response.BadRequest(w, model.NewValidationError("validation", err.Error()))
		return
	}

	go func() {
		enrichCtx, enrichCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer enrichCancel()
		_ = enrichCtx
	}()

	entry, err := h.service.CreateLog(ctx, req.Level, req.Source, req.Message, req.Keywords)
	if err != nil {
		h.logger.Errorf("failed to create log: %v", err)
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Created(w, entry)
}

func validateLogPayload(ctx context.Context, req *CreateLogRequest) error {
	if req.Source == "" {
		return model.NewValidationError("source", "source is required")
	}
	if len(req.Message) > 100000 {
		return model.NewValidationError("message", "message exceeds maximum length")
	}
	_ = ctx
	return nil
}

// CreateBatchLogs POST /api/logs/batch - 批量创建日志
func (h *LogHandler) CreateBatchLogs(w http.ResponseWriter, r *http.Request) {
	var req BatchLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, model.NewValidationError("body", "invalid JSON: "+err.Error()))
		return
	}

	if len(req.Entries) == 0 {
		response.BadRequest(w, model.NewValidationError("entries", "entries cannot be empty"))
		return
	}

	entries := make([]*model.LogEntry, len(req.Entries))
	for i, e := range req.Entries {
		if e.Level == "" {
			e.Level = model.LevelInfo
		}
		entry := model.NewLogEntry(e.Level, e.Source, e.Message)
		entry.Keywords = e.Keywords
		entry.TraceID = e.TraceID
		entry.Metadata = e.Metadata
		entries[i] = entry
	}

	ctx := r.Context()
	result, err := h.service.CreateBatchLogs(ctx, entries)
	if err != nil {
		h.logger.Errorf("failed to create batch logs: %v", err)
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Created(w, result)
}

// GetLog GET /api/logs/{id} - 获取单条日志
func (h *LogHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	ctx := r.Context()
	entry, err := h.service.GetLog(ctx, id)
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

// QueryLogs GET /api/logs - 查询日志
func (h *LogHandler) QueryLogs(w http.ResponseWriter, r *http.Request) {
	query := parseLogQuery(r)

	ctx := r.Context()
	result, err := h.service.QueryLogs(ctx, query)
	if err != nil {
		h.logger.Errorf("failed to query logs: %v", err)
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, result)
}

// DeleteLog DELETE /api/logs/{id} - 删除日志
func (h *LogHandler) DeleteLog(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	ctx := r.Context()
	if err := h.service.DeleteLog(ctx, id); err != nil {
		if model.IsNotFoundError(err) {
			response.NotFound(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.Deleted(w)
}

// DeleteBySource DELETE /api/logs?source=xxx - 删除来源日志
func (h *LogHandler) DeleteBySource(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	if source == "" {
		response.BadRequest(w, model.NewValidationError("source", "source is required"))
		return
	}

	ctx := r.Context()
	count, err := h.service.DeleteBySource(ctx, source)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"deleted_count": count,
	})
}

// parseLogQuery 解析查询参数
func parseLogQuery(r *http.Request) *model.LogQuery {
	q := model.DefaultLogQuery()
	params := r.URL.Query()

	if source := params.Get("source"); source != "" {
		q.Source = source
	}
	if level := params.Get("level"); level != "" {
		if l, err := model.ParseLogLevel(level); err == nil {
			q.Level = l
		}
	}
	if keywords := params["keyword"]; len(keywords) > 0 {
		q.Keywords = keywords
	}
	if startTime := params.Get("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			q.StartTime = &t
		}
	}
	if endTime := params.Get("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			q.EndTime = &t
		}
	}
	if limit := params.Get("limit"); limit != "" {
		if n, err := parseInt(limit); err == nil {
			q.Limit = n
		}
	}
	if offset := params.Get("offset"); offset != "" {
		if n, err := parseInt(offset); err == nil {
			q.Offset = n
		}
	}
	if order := params.Get("order"); order == "asc" || order == "desc" {
		q.SortOrder = model.SortOrder(order)
	}

	return q
}

// extractIDFromPath 从路径中提取ID
func extractIDFromPath(path string) string {
	// 路径格式: /api/logs/{id}
	lastSlash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			lastSlash = i
			break
		}
	}
	if lastSlash < 0 || lastSlash == len(path)-1 {
		return ""
	}
	return path[lastSlash+1:]
}

// parseInt 解析整数
func parseInt(s string) (int, error) {
	var result int
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, io.ErrUnexpectedEOF
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}

// CreateLogWithContext 创建带上下文的日志（用于内部调用）
func (h *LogHandler) CreateLogWithContext(ctx context.Context, level model.LogLevel, source, message string) (*model.LogEntry, error) {
	return h.service.CreateLog(ctx, level, source, message, nil)
}
