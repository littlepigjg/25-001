package model

import (
	"time"
)

// ShortURL 短链记录
type ShortURL struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	CreatedAt time.Time `json:"created_at"`
	Visits    int       `json:"visits"`
	Custom    bool      `json:"custom"`
	Disabled  bool      `json:"disabled"`
}

// NewShortURL 创建短链记录
func NewShortURL(code, rawURL string) *ShortURL {
	return &ShortURL{
		Code:      code,
		RawURL:    rawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    false,
		Disabled:  false,
	}
}

// Validate 验证短链记录
func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return NewValidationError("code", "code cannot be empty")
	}
	if s.RawURL == "" {
		return NewValidationError("raw_url", "raw_url cannot be empty")
	}
	if len(s.Code) > 32 {
		return NewValidationError("code", "code too long")
	}
	if len(s.RawURL) > 2048 {
		return NewValidationError("raw_url", "raw_url too long")
	}
	return nil
}

// IsExpired 检查短链是否过期
func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.CreatedAt.IsZero() {
		return false
	}
	return now.Sub(s.CreatedAt) > 90*24*time.Hour
}

// CreateReq 创建短链请求
type CreateReq struct {
	RawURL    string `json:"raw_url"`
	CustomCode string `json:"custom_code"`
	MaxVisits int    `json:"max_visits"`
}

// Validate 验证创建请求
func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return NewValidationError("raw_url", "raw_url cannot be empty")
	}
	if len(r.RawURL) > 2048 {
		return NewValidationError("raw_url", "raw_url too long")
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) > 32 {
			return NewValidationError("custom_code", "custom_code too long")
		}
		if len(r.CustomCode) < 4 {
			return NewValidationError("custom_code", "custom_code too short, minimum 4 characters")
		}
	}
	if r.MaxVisits < 0 {
		return NewValidationError("max_visits", "max_visits cannot be negative")
	}
	return nil
}
