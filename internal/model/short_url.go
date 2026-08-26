package model

import (
	"fmt"
	"regexp"
	"time"
)

// CreateReq 创建短链请求
type CreateReq struct {
	RawURL     string `json:"raw_url"`
	CustomCode string `json:"custom_code"`
	MaxVisits  int    `json:"max_visits"`
}

// Validate 验证创建请求
func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return NewValidationError("raw_url", "raw_url is required")
	}
	if len(r.RawURL) > 2048 {
		return NewValidationError("raw_url", "raw_url too long")
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) < 4 || len(r.CustomCode) > 16 {
			return NewValidationError("custom_code", "custom_code must be 4-16 characters")
		}
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, r.CustomCode)
		if !matched {
			return NewValidationError("custom_code", "custom_code contains invalid characters")
		}
	}
	if r.MaxVisits < 0 {
		return NewValidationError("max_visits", "max_visits cannot be negative")
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
	MaxVisits int       `json:"max_visits"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Validate 验证短链记录
func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return NewValidationError("code", "code is required")
	}
	if s.RawURL == "" {
		return NewValidationError("raw_url", "raw_url is required")
	}
	if len(s.RawURL) > 2048 {
		return NewValidationError("raw_url", "raw_url too long")
	}
	return nil
}

// IsExpired 检查短链是否过期
func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.ExpiresAt == nil {
		return false
	}
	return now.After(*s.ExpiresAt)
}

// IsMaxVisitsReached 检查是否达到最大访问数
func (s *ShortURL) IsMaxVisitsReached() bool {
	if s.MaxVisits <= 0 {
		return false
	}
	return s.Visits >= s.MaxVisits
}

// CanRedirect 检查是否可以重定向
func (s *ShortURL) CanRedirect(now time.Time) bool {
	if s.Disabled {
		return false
	}
	if s.IsExpired(now) {
		return false
	}
	if s.IsMaxVisitsReached() {
		return false
	}
	return true
}

// String 返回字符串表示
func (s *ShortURL) String() string {
	return fmt.Sprintf("ShortURL{Code: %s, RawURL: %s, Visits: %d}", s.Code, s.RawURL, s.Visits)
}
