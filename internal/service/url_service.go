package service

import (
	"context"
	"errors"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	pkgerrors "logalert/pkg/errors"
	"logalert/pkg/logger"
	"time"
)

// URLService URL短链服务
type URLService struct {
	cfg    *config.Config
	store  *store.URLStore
	logger *logger.Logger
}

// NewURLService 创建URL服务
func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, pkgerrors.New(40000, "config cannot be nil")
	}
	if s == nil {
		return nil, pkgerrors.New(40000, "store cannot be nil")
	}
	return &URLService{
		cfg:    cfg,
		store:  s,
		logger: logger.New(),
	}, nil
}

// Create 创建短链
func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if req == nil {
		return nil, pkgerrors.New(40000, "request cannot be nil")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	code := req.CustomCode
	if code == "" {
		code = generateShortCode()
	}

	url := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    req.CustomCode != "",
		Disabled:  false,
	}

	if err := s.store.Save(url, false); err != nil {
		s.logger.Errorf("failed to save short url: %v", err)
		wrappedErr := pkgerrors.Wrap(50000, "failed to create short url", err)
		var conflictErr *model.ConflictError
		if errors.As(wrappedErr, &conflictErr) {
			s.logger.Warnf("conflict error on create: %v", conflictErr)
		}
		return nil, fmt.Errorf(wrappedErr.Error())
	}

	s.logger.Infof("short url created: code=%s, url=%s", code, req.RawURL)
	return url, nil
}

// GenerateShortCode 生成短码
func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

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
	logger    *logger.Logger
}

// NewRedirectService 创建重定向服务
func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, pkgerrors.New(40000, "url store cannot be nil")
	}
	if ls == nil {
		return nil, pkgerrors.New(40000, "log store cannot be nil")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
		logger:   logger.New(),
	}, nil
}

// HandleRedirect 处理重定向
func (rs *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req == nil || req.Code == "" {
		return nil, pkgerrors.New(40000, "code cannot be empty")
	}

	url, err := rs.urlStore.Get(req.Code)
	if err != nil {
		rs.logger.Errorf("failed to get url for code %s: %v", req.Code, err)
		wrappedErr := pkgerrors.Wrap(50000, "failed to handle redirect", err)
		var notFoundErr *model.NotFoundError
		if errors.As(wrappedErr, &notFoundErr) {
			rs.logger.Warnf("url not found for code: %s", req.Code)
		}
		return nil, wrappedErr
	}

	if url.Disabled {
		return &RedirectResult{
			RawURL: "",
			Status: 410,
		}, nil
	}

	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	rs.logStore.Append(store.AccessLogEntry{
		Code:      url.Code,
		RawURL:    url.RawURL,
		Timestamp: req.Timestamp,
	})

	return &RedirectResult{
		RawURL: url.RawURL,
		Status: 302,
	}, nil
}
