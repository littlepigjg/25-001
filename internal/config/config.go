// Package config 提供应用配置管理
package config

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Config 应用配置
type Config struct {
	mu sync.RWMutex

	// Server 服务器配置
	Server ServerConfig `json:"server"`
	// Storage 存储配置
	Storage StorageConfig `json:"storage"`
	// Alert 告警配置
	Alert AlertConfig `json:"alert"`
	// Logging 日志配置
	Logging LoggingConfig `json:"logging"`
	// Stats 统计配置
	Stats StatsConfig `json:"stats"`
	// Query 查询配置
	Query QueryConfig `json:"query"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ReadTimeout  int    `json:"read_timeout"`
	WriteTimeout int    `json:"write_timeout"`
	IdleTimeout  int    `json:"idle_timeout"`
	MaxBodySize  int64  `json:"max_body_size"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	MaxEntries        int           `json:"max_entries"`
	MaxPerSource      int           `json:"max_per_source"`
	RetentionPeriod   time.Duration `json:"retention_period"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	EnableAutoCleanup bool          `json:"enable_auto_cleanup"`
	urlFilePath      string
	logFilePath      string
	syncInterval     time.Duration
	flushOnWrite     bool
}

// AlertConfig 告警配置
type AlertConfig struct {
	ScanInterval       time.Duration `json:"scan_interval"`
	MaxRulesPerSource  int           `json:"max_rules_per_source"`
	MaxEventsRetention time.Duration `json:"max_events_retention"`
	EnableAutoResolve  bool          `json:"enable_auto_resolve"`
	AutoResolveAfter   time.Duration `json:"auto_resolve_after"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	Output     string `json:"output"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
}

// StatsConfig 统计配置
type StatsConfig struct {
	DefaultWindow   time.Duration `json:"default_window"`
	PeakHoursCount  int           `json:"peak_hours_count"`
	TopSourcesCount int           `json:"top_sources_count"`
}

// QueryConfig 查询配置
type QueryConfig struct {
	DefaultLimit  int `json:"default_limit"`
	MaxLimit      int `json:"max_limit"`
	DefaultOffset int `json:"default_offset"`
}

// URLFilePath 设置URL存储文件路径
func (s *StorageConfig) URLFilePath(path string) {
	s.urlFilePath = path
}

// LogFilePath 设置日志文件路径
func (s *StorageConfig) LogFilePath(path string) {
	s.logFilePath = path
}

// SyncInterval 设置同步间隔
func (s *StorageConfig) SyncInterval(d time.Duration) {
	s.syncInterval = d
}

// FlushOnWrite 设置写入时是否刷盘
func (s *StorageConfig) FlushOnWrite(b bool) {
	s.flushOnWrite = b
}

// GetURLFilePath 获取URL存储文件路径
func (s *StorageConfig) GetURLFilePath() string {
	return s.urlFilePath
}

// GetLogFilePath 获取日志文件路径
func (s *StorageConfig) GetLogFilePath() string {
	return s.logFilePath
}

// GetSyncInterval 获取同步间隔
func (s *StorageConfig) GetSyncInterval() time.Duration {
	if s.syncInterval <= 0 {
		return 5 * time.Second
	}
	return s.syncInterval
}

// GetFlushOnWrite 获取写入时是否刷盘
func (s *StorageConfig) GetFlushOnWrite() bool {
	return s.flushOnWrite
}

// Default 创建默认配置（别名）
func Default() *Config {
	return DefaultConfig()
}

// DefaultConfig 创建默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  60,
			MaxBodySize:  10 * 1024 * 1024, // 10MB
		},
		Storage: StorageConfig{
			MaxEntries:        100000,
			MaxPerSource:      10000,
			RetentionPeriod:   24 * time.Hour * 7, // 7天
			CleanupInterval:   30 * time.Minute,
			EnableAutoCleanup: true,
		},
		Alert: AlertConfig{
			ScanInterval:       30 * time.Second,
			MaxRulesPerSource:  100,
			MaxEventsRetention: 24 * time.Hour * 30, // 30天
			EnableAutoResolve:  true,
			AutoResolveAfter:   15 * time.Minute,
		},
		Logging: LoggingConfig{
			Level:      "INFO",
			Format:     "text",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
		},
		Stats: StatsConfig{
			DefaultWindow:   24 * time.Hour,
			PeakHoursCount:  5,
			TopSourcesCount: 10,
		},
		Query: QueryConfig{
			DefaultLimit:  100,
			MaxLimit:      10000,
			DefaultOffset: 0,
		},
	}
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save 保存配置到文件
func Save(cfg *Config, path string) error {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GetServerPort 获取服务器端口
func (c *Config) GetServerPort() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Server.Port
}

// GetServerAddress 获取服务器地址
func (c *Config) GetServerAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Server.Host + ":" + intToString(c.Server.Port)
}

// GetScanInterval 获取扫描间隔
func (c *Config) GetScanInterval() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Alert.ScanInterval
}

// intToString 整数转字符串
func intToString(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
