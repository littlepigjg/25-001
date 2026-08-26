package store

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"sync"
	"time"
)

type AccessLogStore struct {
	mu     sync.RWMutex
	logs   map[string]*AccessLog
	loaded bool
}

type AccessLog struct {
	ID        string    `json:"id"`
	URLCode   string    `json:"url_code"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer"`
	AccessedAt time.Time `json:"accessed_at"`
	RawURL    string    `json:"raw_url"`
	Status    int       `json:"status"`
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, model.NewValidationError("config", "config cannot be nil")
	}
	return &AccessLogStore{
		logs: make(map[string]*AccessLog),
	}, nil
}

func (s *AccessLogStore) Open(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.mu.Lock()
	s.loaded = true
	s.mu.Unlock()
	return nil
}

func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = make(map[string]*AccessLog)
	s.loaded = false
	return nil
}

func (s *AccessLogStore) Store(log *AccessLog) error {
	if log == nil {
		return model.NewValidationError("log", "log cannot be nil")
	}
	if log.ID == "" {
		log.ID = model.GenerateID()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs[log.ID] = log
	return nil
}

func (s *AccessLogStore) ListByCode(code string) ([]*AccessLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*AccessLog
	for _, log := range s.logs {
		if log.URLCode == code {
			results = append(results, log)
		}
	}
	return results, nil
}

var _ = time.Now
