package service

import (
	"context"
	"logalert/internal/store"
	"logalert/pkg/errors"
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
	urlStore  *store.URLStore
	logStore  *store.AccessLogStore
}

// NewRedirectService 创建重定向服务
func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, nil
	}
	if ls == nil {
		return nil, nil
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
	}, nil
}

// HandleRedirect 处理重定向
func (s *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req == nil {
		return nil, errors.New(40000, "request cannot be nil")
	}
	if req.Code == "" {
		return nil, errors.New(40000, "code cannot be empty")
	}

	shortURL, err := s.urlStore.Get(req.Code)
	if err != nil {
		return nil, err
	}

	now := req.Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	if shortURL.IsExpired(now) {
		return nil, errors.New(40400, "short URL expired: "+req.Code)
	}

	if shortURL.Disabled {
		return nil, errors.New(40400, "short URL disabled: "+req.Code)
	}

	if shortURL.IsMaxVisitsReached() {
		return nil, errors.New(40400, "max visits reached: "+req.Code)
	}

	shortURL.Visits++

	s.logStore.Append(store.AccessLogEntry{
		Code:      req.Code,
		RawURL:    shortURL.RawURL,
		Timestamp: now,
		Status:    302,
	})

	return &RedirectResult{
		RawURL: shortURL.RawURL,
		Status: 302,
	}, nil
}
