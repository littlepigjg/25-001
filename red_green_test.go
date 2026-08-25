package main

import (
	"context"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"os"
	"testing"
	"time"
)

func TestRedGreen(t *testing.T) {
	log := logger.New()
	log.SetLevel(logger.LevelError)

	cfg := config.DefaultConfig()

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxEntries)
	ruleStore := store.NewMemoryRuleStore()
	alertStore := store.NewMemoryAlertStore()

	logSvc := service.NewLogService(logStore, cfg, log)
	alertSvc := service.NewAlertService(alertStore, logStore, ruleStore, cfg, log)

	ctx := context.Background()

	alertSvc.StartEvaluation(ctx)

	rule := model.NewAlertRule("test_error_rule", "test-service", model.LevelError, 1, 50)
	err := ruleStore.Create(rule)
	if err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	for i := 0; i < 100; i++ {
		_, err := logSvc.CreateLog(ctx, model.LevelError, "test-service", fmt.Sprintf("error message %d", i), nil)
		if err != nil {
			t.Fatalf("failed to create log %d: %v", i, err)
		}
	}

	shortCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	_, err = alertSvc.EvaluateRule(shortCtx, rule)
	if err != nil {
		t.Logf("got expected error from context timeout: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	stopDone := make(chan struct{})
	go func() {
		alertSvc.StopEvaluation()
		close(stopDone)
	}()

	select {
	case <-stopDone:
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		os.Exit(0)
	case <-time.After(2 * time.Second):
		fmt.Println("RED（红灯，缺陷未修复）")
		os.Exit(1)
	}
}
