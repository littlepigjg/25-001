// Package handler 提供HTTP处理层
package handler

import (
	"encoding/json"
	"net/http"

	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// RuleHandler 规则处理器
type RuleHandler struct {
	service *service.RuleService
	logger  *logger.Logger
}

// NewRuleHandler 创建规则处理器
func NewRuleHandler(svc *service.RuleService, log *logger.Logger) *RuleHandler {
	return &RuleHandler{
		service: svc,
		logger:  log,
	}
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Source        string         `json:"source"`
	Level         model.LogLevel `json:"level"`
	WindowMinutes int            `json:"window_minutes"`
	Threshold     int            `json:"threshold"`
	Priority      int            `json:"priority"`
	Tags          []string       `json:"tags,omitempty"`
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name          string         `json:"name,omitempty"`
	Description   string         `json:"description,omitempty"`
	Source        string         `json:"source,omitempty"`
	Level         model.LogLevel `json:"level,omitempty"`
	WindowMinutes int            `json:"window_minutes,omitempty"`
	Threshold     int            `json:"threshold,omitempty"`
	Priority      int            `json:"priority,omitempty"`
	Enabled       *bool          `json:"enabled,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
}

// CreateRule POST /api/rules - 创建规则
func (h *RuleHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, model.NewValidationError("body", "invalid JSON: "+err.Error()))
		return
	}

	rule := model.NewAlertRule(req.Name, req.Source, req.Level, req.WindowMinutes, req.Threshold)
	if req.Description != "" {
		rule.Description = req.Description
	}
	if req.Priority > 0 {
		rule.Priority = req.Priority
	}
	if len(req.Tags) > 0 {
		rule.Tags = req.Tags
	}

	ctx := r.Context()
	created, err := h.service.CreateRule(ctx, rule)
	if err != nil {
		h.logger.Errorf("failed to create rule: %v", err)
		if model.IsConflictError(err) {
			response.Conflict(w, err)
		} else if model.IsValidationError(err) {
			response.BadRequest(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.Created(w, created)
}

// GetRule GET /api/rules/{id} - 获取规则
func (h *RuleHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	ctx := r.Context()
	rule, err := h.service.GetRule(ctx, id)
	if err != nil {
		if model.IsNotFoundError(err) {
			response.NotFound(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.Success(w, rule)
}

// ListRules GET /api/rules - 列出规则
func (h *RuleHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	ctx := r.Context()

	var rules []*model.AlertRule
	var err error

	if source != "" {
		rules, err = h.service.ListRulesBySource(ctx, source)
	} else {
		rules, err = h.service.ListRules(ctx)
	}

	if err != nil {
		h.logger.Errorf("failed to list rules: %v", err)
		response.Error(w, http.StatusInternalServerError, err)
		return
	}

	response.Success(w, map[string]interface{}{
		"total": len(rules),
		"items": rules,
	})
}

// UpdateRule PUT /api/rules/{id} - 更新规则
func (h *RuleHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	var req UpdateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, model.NewValidationError("body", "invalid JSON: "+err.Error()))
		return
	}

	rule := &model.AlertRule{
		ID:            id,
		Name:          req.Name,
		Description:   req.Description,
		Source:        req.Source,
		Level:         req.Level,
		WindowMinutes: req.WindowMinutes,
		Threshold:     req.Threshold,
		Priority:      req.Priority,
		Tags:          req.Tags,
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}

	ctx := r.Context()
	updated, err := h.service.UpdateRule(ctx, rule)
	if err != nil {
		h.logger.Errorf("failed to update rule: %v", err)
		if model.IsNotFoundError(err) {
			response.NotFound(w, err)
		} else if model.IsValidationError(err) {
			response.BadRequest(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.Updated(w, updated)
}

// DeleteRule DELETE /api/rules/{id} - 删除规则
func (h *RuleHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	ctx := r.Context()
	if err := h.service.DeleteRule(ctx, id); err != nil {
		if model.IsNotFoundError(err) {
			response.NotFound(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.Deleted(w)
}

// EnableRule POST /api/rules/{id}/enable - 启用规则
func (h *RuleHandler) EnableRule(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	ctx := r.Context()
	if err := h.service.EnableRule(ctx, id); err != nil {
		if model.IsNotFoundError(err) {
			response.NotFound(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.Success(w, map[string]string{"status": "enabled"})
}

// DisableRule POST /api/rules/{id}/disable - 禁用规则
func (h *RuleHandler) DisableRule(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		response.BadRequest(w, model.NewValidationError("id", "id is required"))
		return
	}

	ctx := r.Context()
	if err := h.service.DisableRule(ctx, id); err != nil {
		if model.IsNotFoundError(err) {
			response.NotFound(w, err)
		} else {
			response.Error(w, http.StatusInternalServerError, err)
		}
		return
	}

	response.Success(w, map[string]string{"status": "disabled"})
}
