package service

import (
	"context"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"time"
)

type RedirectRequest struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

type RedirectResult struct {
	RawURL string `json:"raw_url"`
	Status int    `json:"status"`
}

type RedirectService struct {
	urlStore  *store.URLStore
	logStore  *store.AccessLogStore
	logger    *logger.Logger
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, model.NewValidationError("urlStore", "urlStore cannot be nil")
	}
	if ls == nil {
		return nil, model.NewValidationError("logStore", "logStore cannot be nil")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
		logger:   logger.New(),
	}, nil
}

func (svc *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req == nil {
		return nil, model.NewValidationError("req", "req cannot be nil")
	}
	if req.Code == "" {
		return nil, model.NewValidationError("code", "code cannot be empty")
	}

	u, err := svc.urlStore.Get(req.Code)
	if err != nil {
		if model.IsNotFoundError(err) {
			return &RedirectResult{
				RawURL: "",
				Status: 404,
			}, nil
		}
		return nil, err
	}

	if u.Disabled {
		return &RedirectResult{
			RawURL: "",
			Status: 410,
		}, nil
	}

	if u.MaxVisits > 0 && u.Visits >= u.MaxVisits {
		return &RedirectResult{
			RawURL: "",
			Status: 410,
		}, nil
	}

	_ = svc.urlStore.IncrementVisits(req.Code)

	return &RedirectResult{
		RawURL: u.RawURL,
		Status: 302,
	}, nil
}
