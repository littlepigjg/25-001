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
	cfg := config.DefaultConfig()
	cfg.Alert.ScanInterval = 10 * time.Millisecond
	cfg.Alert.AutoResolveAfter = 10 * time.Millisecond
	cfg.Alert.EnableAutoResolve = true
	cfg.Storage.EnableAutoCleanup = true
	cfg.Storage.CleanupInterval = 10 * time.Millisecond

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxEntries)
	ruleStore := store.NewMemoryRuleStore()
	alertStore := store.NewMemoryAlertStore()
	logSvc := service.NewLogService(logStore, cfg, logger.New())
	alertSvc := service.NewAlertService(alertStore, logStore, ruleStore, cfg, logger.New())
	scheduler := service.NewSchedulerService(alertSvc, logSvc, cfg, logger.New())

	alertSvc.SetProcessingChecker(scheduler.IsProcEnabled)

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scheduler.Start(ctx)
	}()

	time.Sleep(time.Microsecond)

	go func() {
		defer wg.Done()
		scheduler.Stop()
	}()

	wg.Wait()

	time.Sleep(200 * time.Millisecond)

	count := alertSvc.GetTotalProcCount()

	if count > 0 {
		fmt.Println("RED (Defect not fixed)")
		t.Errorf("RED (Defect not fixed): GetTotalProcCount should be 0 after Stop, got %d", count)
	} else {
		fmt.Println("GREEN (Defect fixed)")
	}
}

func TestQuickRestart(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Alert.ScanInterval = 10 * time.Millisecond
	cfg.Alert.AutoResolveAfter = 10 * time.Millisecond
	cfg.Alert.EnableAutoResolve = true
	cfg.Storage.EnableAutoCleanup = true
	cfg.Storage.CleanupInterval = 10 * time.Millisecond

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxEntries)
	ruleStore := store.NewMemoryRuleStore()
	alertStore := store.NewMemoryAlertStore()
	logSvc := service.NewLogService(logStore, cfg, logger.New())
	alertSvc := service.NewAlertService(alertStore, logStore, ruleStore, cfg, logger.New())
	scheduler := service.NewSchedulerService(alertSvc, logSvc, cfg, logger.New())

	alertSvc.SetProcessingChecker(scheduler.IsProcEnabled)

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scheduler.Start(ctx)
	}()

	time.Sleep(time.Microsecond)

	go func() {
		defer wg.Done()
		scheduler.Stop()
	}()

	wg.Wait()

	time.Sleep(200 * time.Millisecond)

	count := alertSvc.GetTotalProcCount()

	if count > 0 {
		fmt.Println("RED (Defect not fixed)")
		t.Errorf("RED (Defect not fixed): GetTotalProcCount should be 0 after quick restart, got %d", count)
	} else {
		fmt.Println("GREEN (Defect fixed)")
	}
}

func TestStartStopRace(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Alert.ScanInterval = 10 * time.Millisecond
	cfg.Alert.AutoResolveAfter = 10 * time.Millisecond
	cfg.Alert.EnableAutoResolve = true
	cfg.Storage.EnableAutoCleanup = true
	cfg.Storage.CleanupInterval = 10 * time.Millisecond

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxEntries)
	ruleStore := store.NewMemoryRuleStore()
	alertStore := store.NewMemoryAlertStore()
	logSvc := service.NewLogService(logStore, cfg, logger.New())
	alertSvc := service.NewAlertService(alertStore, logStore, ruleStore, cfg, logger.New())
	scheduler := service.NewSchedulerService(alertSvc, logSvc, cfg, logger.New())

	alertSvc.SetProcessingChecker(scheduler.IsProcEnabled)

	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			scheduler.Start(ctx)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			time.Sleep(time.Microsecond)
			scheduler.Stop()
		}
	}()

	wg.Wait()

	time.Sleep(300 * time.Millisecond)

	count := alertSvc.GetTotalProcCount()

	if count > 0 {
		fmt.Println("RED (Defect not fixed)")
		t.Errorf("RED (Defect not fixed): GetTotalProcCount should be 0 after race condition, got %d", count)
	} else {
		fmt.Println("GREEN (Defect fixed)")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	if code != 0 {
		fmt.Println("RED (Defect not fixed)")
	}
}

var _ = model.DefaultLogQuery
