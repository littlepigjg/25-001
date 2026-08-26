package store

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"sync"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu         sync.RWMutex
	urls       map[string]*model.ShortURL
	codeIndex  map[string][]string
	panicGuard PanicGuardFn
	loaded     bool
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, model.NewValidationError("config", "config cannot be nil")
	}
	return &URLStore{
		urls:      make(map[string]*model.ShortURL),
		codeIndex: make(map[string][]string),
	}, nil
}

func (s *URLStore) Load(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	s.mu.Lock()
	s.loaded = true
	s.mu.Unlock()
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls = make(map[string]*model.ShortURL)
	s.codeIndex = make(map[string][]string)
	s.loaded = false
	return nil
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return model.NewValidationError("url", "url cannot be nil")
	}
	if u.Code == "" {
		return model.NewValidationError("code", "code cannot be empty")
	}

	s.mu.Lock()
	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			s.mu.Unlock()
			return model.NewConflictError("short_url", "code already exists: "+u.Code)
		}
	}
	s.urls[u.Code] = u
	s.mu.Unlock()

	rawURLKey := u.RawURL
	codes := s.codeIndex[rawURLKey]
	if !containsCode(codes, u.Code) {
		codes = append(codes, u.Code)
		s.codeIndex[rawURLKey] = codes
	}

	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.urls[code]
	if !ok {
		return nil, model.NewNotFoundError("short_url", code)
	}
	return u, nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]model.ShortURL, len(s.urls))
	for k, v := range s.urls {
		if s.isURLIndexed(v) {
			snapshot[k] = *v
		}
	}
	return snapshot
}

func (s *URLStore) isURLIndexed(u *model.ShortURL) bool {
	codes, ok := s.codeIndex[u.RawURL]
	if !ok {
		return false
	}
	for _, code := range codes {
		if code == u.Code {
			return true
		}
	}
	return false
}

func (s *URLStore) IncrementVisits(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.urls[code]
	if !ok {
		return model.NewNotFoundError("short_url", code)
	}
	u.Visits++
	return nil
}

func (s *URLStore) FindByRawURL(rawURL string) ([]*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	codes, ok := s.codeIndex[rawURL]
	if !ok {
		return nil, nil
	}

	results := make([]*model.ShortURL, 0, len(codes))
	for _, code := range codes {
		if u, ok := s.urls[code]; ok {
			results = append(results, u)
		}
	}
	return results, nil
}

func (s *URLStore) Delete(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.urls[code]
	if !ok {
		return model.NewNotFoundError("short_url", code)
	}

	if ids, ok := s.codeIndex[u.RawURL]; ok {
		for i, c := range ids {
			if c == code {
				s.codeIndex[u.RawURL] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
		if len(s.codeIndex[u.RawURL]) == 0 {
			delete(s.codeIndex, u.RawURL)
		}
	}

	delete(s.urls, code)
	return nil
}

func containsCode(codes []string, code string) bool {
	for _, c := range codes {
		if c == code {
			return true
		}
	}
	return false
}
