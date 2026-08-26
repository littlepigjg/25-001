package model

import (
	"time"
)

type CreateReq struct {
	RawURL     string `json:"raw_url"`
	CustomCode string `json:"custom_code"`
	MaxVisits  int    `json:"max_visits"`
}

type ShortURL struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	CreatedAt time.Time `json:"created_at"`
	Visits    int       `json:"visits"`
	Custom    bool      `json:"custom"`
	Disabled  bool      `json:"disabled"`
	MaxVisits int       `json:"max_visits"`
}

func (req *CreateReq) Validate() error {
	if req.RawURL == "" {
		return NewValidationError("raw_url", "raw_url cannot be empty")
	}
	if req.MaxVisits < 0 {
		return NewValidationError("max_visits", "max_visits cannot be negative")
	}
	return nil
}

func (u *ShortURL) Validate() error {
	if u.Code == "" {
		return NewValidationError("code", "code cannot be empty")
	}
	if u.RawURL == "" {
		return NewValidationError("raw_url", "raw_url cannot be empty")
	}
	return nil
}

func (u *ShortURL) IsExpired(now time.Time) bool {
	if u.MaxVisits > 0 && u.Visits >= u.MaxVisits {
		return true
	}
	return false
}
