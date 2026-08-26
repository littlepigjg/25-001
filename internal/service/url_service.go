// Package service 提供业务逻辑层
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"time"
)

// URLService URL短链服务
type URLService struct {
	cfg       *config.Config
	urlStore  *store.URLStore
	alertStore *store.MemoryAlertStore
}

// NewURLService 创建URL服务
func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	return &URLService{
		cfg:      cfg,
		urlStore: s,
	}, nil
}

// SetAlertStore 设置告警存储
func (svc *URLService) SetAlertStore(alertStore *store.MemoryAlertStore) {
	svc.alertStore = alertStore
}

// Create 创建短链
func (svc *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var code string
	var isCustom bool

	if req.CustomCode != "" {
		code = req.CustomCode
		isCustom = true
	} else {
		var err error
		code, err = generateCode()
		if err != nil {
			return nil, model.NewInternalError("create", "failed to generate code")
		}
	}

	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    isCustom,
		Disabled:  false,
	}

	if err := svc.urlStore.Save(shortURL, false); err != nil {
		return nil, err
	}

	if svc.alertStore != nil {
		alert := &model.AlertEvent{
			ID:        model.GenerateID(),
			RuleID:    "url_create",
			RuleName:  "URL Create Monitor",
			Source:    "url_service",
			Level:     model.LevelInfo,
			Count:     1,
			Threshold: 1,
			WindowMinutes: 1,
			TriggeredAt: time.Now(),
			Status:    model.AlertFired,
			Description: "Short URL created: " + code,
		}
		svc.alertStore.Store(alert)
	}

	return shortURL, nil
}

// Get 获取短链
func (svc *URLService) Get(ctx context.Context, code string) (*model.ShortURL, error) {
	return svc.urlStore.Get(code)
}

// generateCode 生成随机短链码
func generateCode() (string, error) {
	b := make([]byte, 6)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:6], nil
}
