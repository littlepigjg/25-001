package main

import (
	"context"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"testing"
	"time"
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Alert.ScanInterval = 1 * time.Millisecond
	cfg.Alert.EnableAutoResolve = true
	cfg.Alert.AutoResolveAfter = 1 * time.Millisecond
	cfg.Storage.EnableAutoCleanup = true
	cfg.Storage.CleanupInterval = 1 * time.Millisecond
	cfg.Alert.MaxEventsRetention = 24 * time.Hour

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxEntries)
	ruleStore := store.NewMemoryRuleStore()
	alertStore := store.NewMemoryAlertStore()

	log := logger.New()
	log.SetLevel(logger.LevelError)

	logSvc := service.NewLogService(logStore, cfg, log)
	ruleSvc := service.NewRuleService(ruleStore, cfg, log)
	alertSvc := service.NewAlertService(alertStore, logStore, ruleStore, cfg, log)

	for i := 0; i < 3; i++ {
		rule := model.NewAlertRule(
			fmt.Sprintf("test-rule-%d", i),
			fmt.Sprintf("source-%d", i),
			model.LevelError,
			5,
			1,
		)
		ruleSvc.CreateRule(context.Background(), rule)
	}

	scheduler := service.NewSchedulerService(alertSvc, logSvc, cfg, log)

	ctx, cancel := context.WithCancel(context.Background())
	scheduler.Start(ctx)

	time.Sleep(200 * time.Millisecond)

	cancel()

	start := time.Now()
	scheduler.Stop()
	elapsed := time.Since(start)

	if elapsed > 5*time.Second {
		fmt.Printf("RED (红灯，缺陷未修复) - Stop() took %v\n", elapsed)
		t.Fail()
	} else {
		fmt.Printf("GREEN (绿灯，缺陷已修复) - Stop() took %v\n", elapsed)
	}
}
