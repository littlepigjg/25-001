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
	cfg.Storage.RetentionPeriod = 100 * time.Millisecond

	logStore := store.NewMemoryLogStore(1000)
	alertStore := store.NewMemoryAlertStore()
	log := logger.New()

	svc := service.NewCleanupService(logStore, alertStore, cfg, log)

	now := time.Now()
	expired := &model.LogEntry{
		ID:        model.GenerateID(),
		Timestamp: now.Add(-2 * time.Hour),
		Level:     model.LevelInfo,
		Source:    "test-app",
		Message:   "expired log message for cleanup test",
	}
	if err := logStore.Store(expired); err != nil {
		t.Fatalf("failed to store expired entry: %v", err)
	}

	errInjector := fmt.Errorf("simulated cleanup failure")
	logStore.SetFailInjector(func() error {
		return errInjector
	})

	ctx := context.Background()
	count, err := svc.CleanupExpiredLogs(ctx)
	if err == nil {
		t.Fatal("expected error from failed cleanup, got nil")
	}
	if count != 0 {
		t.Fatalf("expected 0 count on failed cleanup, got %d", count)
	}

	fresh := &model.LogEntry{
		ID:        model.GenerateID(),
		Timestamp: now,
		Level:     model.LevelInfo,
		Source:    "test-app",
		Message:   "fresh log entry after cleanup failure",
	}
	err = logStore.Store(fresh)
	if err != nil {
		fmt.Println("RED")
		t.Fatalf("cleanup state leaked: cannot store new entry after failed cleanup: %v", err)
	}

	_, err = logStore.Cleanup(100 * time.Millisecond)
	if err != nil {
		fmt.Println("RED")
		t.Fatalf("cleanup state leaked: cannot start new cleanup after failed cleanup: %v", err)
	}

	logStore.SetFailInjector(nil)

	count2, err := svc.CleanupExpiredLogs(ctx)
	if err != nil {
		fmt.Println("RED")
		t.Fatalf("second cleanup attempt failed: %v", err)
	}
	if count2 < 0 {
		fmt.Println("RED")
		t.Fatalf("invalid cleanup count: %d", count2)
	}

	fmt.Println("GREEN")
}
