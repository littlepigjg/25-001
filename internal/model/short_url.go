// Package model 定义系统核心数据模型
package model

import (
	"regexp"
	"time"
)

// CreateReq 创建短链请求
type CreateReq struct {
	RawURL     string `json:"raw_url"`
	CustomCode string `json:"custom_code"`
	MaxVisits  int    `json:"max_visits"`
}

// ShortURL 短链实体
type ShortURL struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	CreatedAt time.Time `json:"created_at"`
	Visits    int       `json:"visits"`
	Custom    bool      `json:"custom"`
	Disabled  bool      `json:"disabled"`
}

// urlRegex URL合法性正则
var urlRegex = regexp.MustCompile(`^https?://.+`)

// codeRegex 短链码合法性正则
var codeRegex = regexp.MustCompile(`^[a-zA-Z0-9]{4,16}$`)

// Validate 验证创建请求
func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return NewValidationError("raw_url", "raw_url cannot be empty")
	}
	if !urlRegex.MatchString(r.RawURL) {
		return NewValidationError("raw_url", "raw_url must start with http:// or https://")
	}
	if r.CustomCode != "" && !codeRegex.MatchString(r.CustomCode) {
		return NewValidationError("custom_code", "custom_code must be 4-16 alphanumeric characters")
	}
	if r.MaxVisits < 0 {
		return NewValidationError("max_visits", "max_visits cannot be negative")
	}
	return nil
}

// Validate 验证短链
func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return NewValidationError("code", "code cannot be empty")
	}
	if s.RawURL == "" {
		return NewValidationError("raw_url", "raw_url cannot be empty")
	}
	return nil
}

// IsExpired 检查短链是否过期
func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.CreatedAt.IsZero() {
		return false
	}
	return now.Sub(s.CreatedAt) > 30*24*time.Hour
}
