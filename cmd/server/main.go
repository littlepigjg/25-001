// Package main 应用入口
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"logalert/internal/config"
	"logalert/internal/handler"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func main() {
	// 初始化日志
	log := logger.New()
	log.SetLevel(logger.LevelInfo)
	log.Info("starting logalert service...")

	// 加载配置
	cfg := loadConfig(log)

	// 初始化存储层
	logStore := store.NewMemoryLogStore(cfg.Storage.MaxEntries)
	ruleStore := store.NewMemoryRuleStore()
	alertStore := store.NewMemoryAlertStore()

	log.Info("storage layer initialized")

	// 初始化服务层
	logSvc := service.NewLogService(logStore, cfg, log)
	ruleSvc := service.NewRuleService(ruleStore, cfg, log)
	alertSvc := service.NewAlertService(alertStore, logStore, ruleStore, cfg, log)
	statsSvc := service.NewStatsService(logStore, cfg, log)
	schedulerSvc := service.NewSchedulerService(alertSvc, logSvc, cfg, log)

	log.Info("service layer initialized")

	// 初始化处理器层
	logHandler := handler.NewLogHandler(logSvc, log)
	ruleHandler := handler.NewRuleHandler(ruleSvc, log)
	alertHandler := handler.NewAlertHandler(alertSvc, log)
	statsHandler := handler.NewStatsHandler(statsSvc, log)
	healthHandler := handler.NewHealthHandler(logSvc, alertSvc, ruleSvc, schedulerSvc, cfg, log)
	middleware := handler.NewMiddleware(log)

	log.Info("handler layer initialized")

	// 创建根上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动调度服务
	schedulerSvc.Start(ctx)
	log.Info("scheduler service started")

	// 创建HTTP服务器
	mux := http.NewServeMux()

	// 健康检查端点
	mux.HandleFunc("/health", healthHandler.Healthy)
	mux.HandleFunc("/ready", healthHandler.Ready)
	mux.HandleFunc("/info", healthHandler.Info)

	// API路由
	apiMux := http.NewServeMux()

	// 日志API
	apiMux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			logHandler.CreateLog(w, r)
		case http.MethodGet:
			logHandler.QueryLogs(w, r)
		case http.MethodDelete:
			if r.URL.Query().Get("source") != "" {
				logHandler.DeleteBySource(w, r)
			} else {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiMux.HandleFunc("/api/logs/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			logHandler.CreateBatchLogs(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiMux.HandleFunc("/api/logs/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			logHandler.GetLog(w, r)
		case http.MethodDelete:
			logHandler.DeleteLog(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 规则API
	apiMux.HandleFunc("/api/rules", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			ruleHandler.CreateRule(w, r)
		case http.MethodGet:
			ruleHandler.ListRules(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiMux.HandleFunc("/api/rules/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > len("/api/rules/") {
			// 检查子路径
			rest := path[len("/api/rules/"):]
			if rest == "" {
				switch r.Method {
				case http.MethodGet:
					ruleHandler.GetRule(w, r)
				case http.MethodPut:
					ruleHandler.UpdateRule(w, r)
				case http.MethodDelete:
					ruleHandler.DeleteRule(w, r)
				default:
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
			// 子路由处理
			if rest == "enable" {
				if r.Method == http.MethodPost {
					ruleHandler.EnableRule(w, r)
				} else {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
			if rest == "disable" {
				if r.Method == http.MethodPost {
					ruleHandler.DisableRule(w, r)
				} else {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
			// 假设是 /api/rules/{id}/action 格式
			// 这里简化处理
			http.Error(w, "not found", http.StatusNotFound)
		}
	})

	// 告警API
	apiMux.HandleFunc("/api/alerts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			alertHandler.ListAlerts(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiMux.HandleFunc("/api/alerts/active/count", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			alertHandler.CountActive(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiMux.HandleFunc("/api/alerts/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > len("/api/alerts/") {
			rest := path[len("/api/alerts/"):]
			if rest == "" {
				switch r.Method {
				case http.MethodGet:
					alertHandler.GetAlert(w, r)
				case http.MethodDelete:
					alertHandler.DeleteAlert(w, r)
				default:
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
			if rest == "acknowledge" {
				if r.Method == http.MethodPost {
					alertHandler.AcknowledgeAlert(w, r)
				} else {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
			if rest == "resolve" {
				if r.Method == http.MethodPost {
					alertHandler.ResolveAlert(w, r)
				} else {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}
	})

	// 统计API
	apiMux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			statsHandler.GetStatistics(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiMux.HandleFunc("/api/stats/hourly", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			statsHandler.GetHourlyTrends(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiMux.HandleFunc("/api/stats/sources", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			statsHandler.GetSourceStats(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiMux.HandleFunc("/api/stats/daily", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			statsHandler.GetDailyReport(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	apiMux.HandleFunc("/api/stats/error-rate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			statsHandler.GetErrorRateTrend(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 查找前端目录
	webDir := findWebDir()
	if webDir != "" {
		log.Infof("static files served from: %s", webDir)
	}

	// 组合中间件
	var handlerChain http.Handler = apiMux
	handlerChain = middleware.RequestLogger(handlerChain)
	handlerChain = middleware.CORS(handlerChain)
	handlerChain = middleware.Recover(handlerChain)
	handlerChain = middleware.RequestID(handlerChain)
	handlerChain = middleware.ContentTypeChecker(handlerChain)

	// 将API挂载到根mux
	mux.Handle("/api/", handlerChain)
	// 确保health、ready、info不在/api前缀下
	// 重新挂载根路径处理
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/":
			// 首页
			if webDir == "" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>LogAlert</title></head>
<body>
<h1>LogAlert - Real-time Log Aggregation & Alert Engine</h1>
<p>API: <a href="/info">/info</a></p>
<p>Health: <a href="/health">/health</a></p>
<p>Ready: <a href="/ready">/ready</a></p>
</body>
</html>`))
			} else {
				http.FileServer(http.Dir(webDir)).ServeHTTP(w, r)
			}
		default:
			if webDir != "" {
				http.FileServer(http.Dir(webDir)).ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
		}
	})

	// 创建服务器
	addr := cfg.GetServerAddress()
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// 启动服务器
	go func() {
		log.Infof("server listening on: %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// 等待信号进行优雅关闭
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Infof("received signal: %v, shutting down...", sig)

	// 取消上下文
	cancel()

	// 停止调度服务
	schedulerSvc.Stop()

	// 优雅关闭HTTP服务器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Errorf("server shutdown error: %v", err)
	}

	log.Info("server stopped gracefully")
	fmt.Println("LogAlert service stopped")
}

// loadConfig 加载配置
func loadConfig(log *logger.Logger) *config.Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.json"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Warnf("failed to load config from %s: %v, using defaults", configPath, err)
		return config.DefaultConfig()
	}

	log.Infof("config loaded from: %s", configPath)
	return cfg
}

// findWebDir 查找前端目录
func findWebDir() string {
	// 检查常见的前端目录
	dirs := []string{
		"web",
		"static",
		"frontend",
		"ui",
	}

	for _, dir := range dirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			indexPath := filepath.Join(dir, "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				absPath, _ := filepath.Abs(dir)
				return absPath
			}
		}
	}

	return ""
}
