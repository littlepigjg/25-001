package service

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"time"
)

type URLService struct {
	urlStore *store.URLStore
	logger   *logger.Logger
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, model.NewValidationError("config", "config cannot be nil")
	}
	if s == nil {
		return nil, model.NewValidationError("store", "store cannot be nil")
	}
	return &URLService{
		urlStore: s,
		logger:   logger.New(),
	}, nil
}

func (svc *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if req == nil {
		return nil, model.NewValidationError("req", "req cannot be nil")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	code := req.CustomCode
	if code == "" {
		code = model.GenerateID()[:8]
	}

	u := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    req.CustomCode != "",
		Disabled:  false,
	}

	if err := svc.urlStore.Save(u, false); err != nil {
		svc.logger.Errorf("failed to save short url: %v", err)
		return nil, err
	}

	return u, nil
}

func (svc *URLService) Get(ctx context.Context, code string) (*model.ShortURL, error) {
	if code == "" {
		return nil, model.NewValidationError("code", "code cannot be empty")
	}
	return svc.urlStore.Get(code)
}

func (svc *URLService) ListAll(ctx context.Context) ([]*model.ShortURL, error) {
	snapshot := svc.urlStore.RawSnapshot()
	results := make([]*model.ShortURL, 0, len(snapshot))
	for _, u := range snapshot {
		uCopy := u
		results = append(results, &uCopy)
	}
	return results, nil
}

func (svc *URLService) Delete(ctx context.Context, code string) error {
	if code == "" {
		return model.NewValidationError("code", "code cannot be empty")
	}
	return svc.urlStore.Delete(code)
}
