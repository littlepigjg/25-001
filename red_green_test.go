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
	origLocal := time.Local
	time.Local = time.FixedZone("CST", 8*3600)
	defer func() { time.Local = origLocal }()

	logStore := store.NewMemoryLogStore(10000)
	alertStore := store.NewMemoryAlertStore()
	ruleStore := store.NewMemoryRuleStore()

	rule := model.NewAlertRule("test-rule", "test-source", model.LevelError, 10, 3)
	rule.Enabled = true
	if err := ruleStore.Create(rule); err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	now := time.Now()
	for i := 0; i < 3; i++ {
		entry := model.NewLogEntry(model.LevelError, "test-source", "error message")
		entry.Timestamp = now.Add(-time.Duration(i) * time.Minute)
		if err := logStore.Store(entry); err != nil {
			t.Fatalf("failed to store log entry: %v", err)
		}
	}

	cfg := config.DefaultConfig()
	log := logger.New()
	alertService := service.NewAlertService(alertStore, logStore, ruleStore, cfg, log)

	ctx := context.Background()
	event, err := alertService.EvaluateRule(ctx, rule)

	if err != nil {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Logf("EvaluateRule returned error: %v", err)
		return
	}

	if event != nil {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	} else {
		fmt.Println("RED (红灯，缺陷未修复)")
	}
}
