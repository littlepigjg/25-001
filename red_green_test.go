package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	log := logger.New()
	logStore := store.NewMemoryLogStore(cfg.Storage.MaxEntries)
	windowMgr := service.NewWindowManager(cfg, log)
	logSvc := service.NewLogService(logStore, cfg, log, windowMgr)

	now := time.Now()
	query := &model.LogQuery{
		StartTime: &now,
		EndTime:   timePtr(now.Add(-1 * time.Hour)),
		Limit:     100,
	}

	_, err := logSvc.QueryLogs(context.Background(), query)
	if err != nil {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	} else {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.FailNow()
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
