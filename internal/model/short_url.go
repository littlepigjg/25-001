package model

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// CreateReq 创建短链请求
type CreateReq struct {
	RawURL     string `json:"raw_url"`
	CustomCode string `json:"custom_code"`
	MaxVisits  int    `json:"max_visits"`
}

// Validate 校验请求参数
func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return NewValidationError("raw_url", "raw url is required")
	}
	if len(r.RawURL) > 2048 {
		return NewValidationError("raw_url", "raw url too long")
	}
	if r.MaxVisits < 0 {
		return NewValidationError("max_visits", "max visits cannot be negative")
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) < 4 || len(r.CustomCode) > 16 {
			return NewValidationError("custom_code", "custom code must be 4-16 characters")
		}
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, r.CustomCode)
		if !matched {
			return NewValidationError("custom_code", "custom code contains invalid characters")
		}
	}
	if !strings.HasPrefix(r.RawURL, "http://") && !strings.HasPrefix(r.RawURL, "https://") {
		return NewValidationError("raw_url", "raw url must start with http:// or https://")
	}
	return nil
}

// ShortURL 短链记录
type ShortURL struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	CreatedAt time.Time `json:"created_at"`
	Visits    int       `json:"visits"`
	Custom    bool      `json:"custom"`
	Disabled  bool      `json:"disabled"`
}

// Validate 校验短链记录
func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return NewValidationError("code", "code is required")
	}
	if s.RawURL == "" {
		return NewValidationError("raw_url", "raw url is required")
	}
	if !strings.HasPrefix(s.RawURL, "http://") && !strings.HasPrefix(s.RawURL, "https://") {
		return NewValidationError("raw_url", "raw url must start with http:// or https://")
	}
	return nil
}

// IsExpired 检查是否过期
func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.CreatedAt.IsZero() {
		return false
	}
	return now.Sub(s.CreatedAt) > 90*24*time.Hour
}

// ErrURLNotFound URL未找到错误
var ErrURLNotFound = errors.New("url not found")
