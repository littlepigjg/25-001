package main

import (
	"context"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"sync"
	"testing"
	"time"
)

func TestRedGreen(t *testing.T) {
	log := logger.New()
	log.SetLevel(logger.LevelInfo)

	logStore := store.NewMemoryLogStore(10000)
	alertStore := store.NewMemoryAlertStore()
	ruleStore := store.NewMemoryRuleStore()

	cfg := config.DefaultConfig()
	cfg.Alert.ScanInterval = 10 * time.Millisecond // Set a very short interval to trigger scan quickly

	logService := service.NewLogService(logStore, cfg, log)
	alertService := service.NewAlertService(alertStore, logStore, ruleStore, cfg, log)
	ruleService := service.NewRuleService(ruleStore, cfg, log)

	// Create an invalid rule to trigger error path
	invalidRule := model.NewAlertRule("test_rule", "syslog", "ERROR", 0, 10) // WindowMinutes is 0
	invalidRule.Enabled = true
	_, err := ruleService.CreateRule(context.Background(), invalidRule)
	if err != nil {
		// If creation fails due to validation, we manually add to store to simulate a corrupted state
		invalidRule.ID = "manual_id_12345"
		ruleStore.Create(invalidRule)
	}

	scheduler := service.NewSchedulerService(alertService, logService, cfg, log)

	ctx := context.Background()
	scheduler.Start(ctx)

	// Wait enough time for at least one scan to trigger and fail
	time.Sleep(50 * time.Millisecond)

	// Try to stop with timeout
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scheduler.Stop()
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	case <-time.After(2 * time.Second):
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Log("Timeout waiting for scheduler to stop - goroutine leak suspected")
		// Force stop the test
		t.FailNow()
	}
}
