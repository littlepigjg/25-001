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
	log := logger.New()
	logStore := store.NewMemoryLogStore(50000)
	logStore.SetQuerySpeed(100 * time.Microsecond)

	now := time.Now()
	for i := 0; i < 500; i++ {
		entry := &model.LogEntry{
			ID:        fmt.Sprintf("log-%d", i),
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
			Level:     model.LevelInfo,
			Source:    fmt.Sprintf("source-%d", i%10),
			Message:   fmt.Sprintf("log message number %d from service", i),
			Keywords:  []string{"log", "message"},
		}
		if err := logStore.Store(entry); err != nil {
			t.Fatalf("failed to store entry %d: %v", i, err)
		}
	}

	statsSvc := service.NewStatsService(logStore, cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	startTime := now.Add(-1 * time.Hour)
	endTime := now

	result, err := statsSvc.GetStatistics(ctx, startTime, endTime)

	if err != nil {
		if ctx.Err() != nil {
			fmt.Println("GREEN（绿灯，缺陷已修复）")
			return
		}
		t.Fatalf("unexpected error: %v", err)
	}

	if result != nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Logf("query returned %d entries despite context cancellation", result.TotalCount)
		return
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
