// Package store 提供数据存储层
package store

import (
	"context"
	"logalert/internal/config"
	"logalert/internal/model"
	"sync"
	"time"
)

// AccessLogStore 访问日志存储
type AccessLogStore struct {
	mu      sync.Mutex
	logs    []*AccessLogEntry
	opened  bool
	closed  bool
	alerts  map[string]*model.AlertEvent
	alertMu sync.RWMutex
}

// AccessLogEntry 访问日志条目
type AccessLogEntry struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	Timestamp time.Time `json:"timestamp"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
}

// NewAccessLogStore 创建访问日志存储
func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	return &AccessLogStore{
		logs:   make([]*AccessLogEntry, 0),
		alerts: make(map[string]*model.AlertEvent),
	}, nil
}

// Open 打开存储
func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opened = true
	return nil
}

// Close 关闭存储
func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

// Record 记录一次访问
func (s *AccessLogStore) Record(code, rawURL, ip, userAgent string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := &AccessLogEntry{
		Code:      code,
		RawURL:    rawURL,
		Timestamp: time.Now(),
		IP:        ip,
		UserAgent: userAgent,
	}
	s.logs = append(s.logs, entry)
	return nil
}

// TrackAlert 追踪一个告警事件
func (s *AccessLogStore) TrackAlert(alert *model.AlertEvent) {
	s.alertMu.Lock()
	defer s.alertMu.Unlock()
	s.alerts[alert.ID] = alert
}

// GetAlertDuration 获取告警的持续时间
func (s *AccessLogStore) GetAlertDuration(alertID string) time.Duration {
	s.alertMu.RLock()
	defer s.alertMu.RUnlock()

	alert, ok := s.alerts[alertID]
	if !ok {
		return 0
	}
	return alert.Duration()
}

// GetLogs 获取所有访问日志
func (s *AccessLogStore) GetLogs() []*AccessLogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]*AccessLogEntry, len(s.logs))
	copy(result, s.logs)
	return result
}

// CountByCode 按code统计访问次数
func (s *AccessLogStore) CountByCode(code string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for _, log := range s.logs {
		if log.Code == code {
			count++
		}
	}
	return count
}
