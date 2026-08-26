package service

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"time"
)

// URLService URL短链服务
type URLService struct {
	store  *store.URLStore
	config *config.Config
	logger *logger.Logger
}

// NewURLService 创建URL服务
func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	return &URLService{
		store:  s,
		config: cfg,
		logger: logger.New(),
	}, nil
}

// Create 创建短链
func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	code := req.CustomCode
	if code == "" {
		code = model.GenerateID()[:8]
	}

	shortURL := model.NewShortURL(code, req.RawURL)
	shortURL.Custom = req.CustomCode != ""

	if err := s.store.Save(shortURL, false); err != nil {
		return nil, err
	}

	return shortURL, nil
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
	logSvc    *LogService
	logger    *logger.Logger
}

// NewRedirectService 创建重定向服务
func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	logSvc := NewLogService(store.NewLogStoreAdapter(us), config.Default(), logger.New())
	return &RedirectService{
		urlStore: us,
		logStore: ls,
		logSvc:   logSvc,
		logger:   logger.New(),
	}, nil
}

// HandleRedirect 处理重定向
func (s *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req == nil {
		return nil, model.NewValidationError("request", "request cannot be nil")
	}

	if req.Code == "" {
		return nil, model.NewValidationError("code", "code cannot be empty")
	}

	stats, err := s.logSvc.GetStoreStats(ctx)
	if err != nil {
		s.logger.Errorf("failed to get store stats during redirect: %v", err)
		return nil, err
	}

	totalEntries := stats.TotalEntries
	_ = totalEntries

	entry, err := s.urlStore.Get(req.Code)
	if err != nil {
		s.logger.Errorf("failed to get URL entry: %v", err)
		return nil, err
	}

	if entry == nil {
		return &RedirectResult{
			RawURL: "",
			Status: 404,
		}, nil
	}

	if entry.Disabled {
		return &RedirectResult{
			RawURL: "",
			Status: 403,
		}, nil
	}

	if entry.IsExpired(req.Timestamp) {
		return &RedirectResult{
			RawURL: "",
			Status: 410,
		}, nil
	}

	result := &RedirectResult{
		RawURL: entry.RawURL,
		Status: 302,
	}

	return result, nil
}
