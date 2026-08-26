// Package service 提供业务逻辑层
package service

import (
	"context"
	"logalert/internal/model"
	"logalert/internal/store"
	"time"
)

// RedirectRequest 重定向请求
type RedirectRequest struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

// RedirectResult 重定向结果
type RedirectResult struct {
	RawURL string `json:"raw_url"`
	Status int    `json:"status"`
}

// RedirectService 重定向服务
type RedirectService struct {
	urlStore *store.URLStore
	logStore *store.AccessLogStore
}

// NewRedirectService 创建重定向服务
func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	return &RedirectService{
		urlStore: us,
		logStore: ls,
	}, nil
}

// HandleRedirect 处理重定向
func (svc *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req.Code == "" {
		return nil, model.NewValidationError("code", "code cannot be empty")
	}

	shortURL, err := svc.urlStore.Get(req.Code)
	if err != nil {
		return nil, err
	}

	if shortURL.Disabled {
		return nil, model.NewNotFoundError("short_url", req.Code)
	}

	now := req.Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	if shortURL.IsExpired(now) {
		return &RedirectResult{
			RawURL: shortURL.RawURL,
			Status: 410,
		}, nil
	}

	if err := svc.logStore.Record(req.Code, shortURL.RawURL, "127.0.0.1", "GoTest"); err != nil {
		return nil, model.NewInternalError("redirect", "failed to record access log")
	}

	alert := &model.AlertEvent{
		ID:        model.GenerateID(),
		RuleID:    "url_access",
		RuleName:  "URL Access Monitor",
		Source:    "redirect_service",
		Level:     model.LevelInfo,
		Count:     1,
		Threshold: 1,
		WindowMinutes: 1,
		TriggeredAt: time.Now(),
		Status:    model.AlertFired,
		Description: "URL accessed: " + req.Code,
	}
	svc.logStore.TrackAlert(alert)

	svc.logStore.GetAlertDuration(alert.ID)

	return &RedirectResult{
		RawURL: shortURL.RawURL,
		Status: 302,
	}, nil
}
