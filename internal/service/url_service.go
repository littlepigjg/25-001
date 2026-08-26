package service

import (
	"context"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/errors"
	"time"
)

// URLService 短链服务
type URLService struct {
	cfg   *config.Config
	store *store.URLStore
}

// NewURLService 创建URL服务
func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if s == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}
	return &URLService{
		cfg:   cfg,
		store: s,
	}, nil
}

// Create 创建短链
func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, errors.New(40000, err.Error())
	}

	var code string
	var isCustom bool

	if req.CustomCode != "" {
		code = req.CustomCode
		isCustom = true
	} else {
		code = generateShortCode()
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    isCustom,
		Disabled:  false,
		MaxVisits: req.MaxVisits,
		ExpiresAt: &expiresAt,
	}

	overwrite := isCustom
	err := s.store.Save(shortURL, overwrite)
	if err != nil {
		return nil, err
	}

	return shortURL, nil
}

// Get 根据code获取短链
func (s *URLService) Get(ctx context.Context, code string) (*model.ShortURL, error) {
	if code == "" {
		return nil, model.NewValidationError("code", "code cannot be empty")
	}
	return s.store.Get(code)
}

// generateShortCode 生成短码
func generateShortCode() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Millisecond)
	}
	return string(b)
}
