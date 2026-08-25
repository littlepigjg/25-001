package logalert

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
	logStore := store.NewMemoryLogStore(10000)
	ruleStore := store.NewMemoryRuleStore()
	alertStore := store.NewMemoryAlertStore()
	cfg := config.DefaultConfig()
	log := logger.New()

	rule := model.NewAlertRule("test-rule", "test-source", model.LevelError, 5, 3)
	if err := ruleStore.Create(rule); err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	for i := 0; i < 5; i++ {
		entry := model.NewLogEntry(model.LevelError, "test-source", "critical error detected")
		if err := logStore.Store(entry); err != nil {
			t.Fatalf("failed to store log entry: %v", err)
		}
	}

	alertService := service.NewAlertService(alertStore, logStore, ruleStore, cfg, log)

	expiredCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	time.Sleep(100 * time.Millisecond)

	red := false

	_, err := alertService.EvaluateAllRules(expiredCtx)
	if err != nil {
		red = true
	}

	alertService.SetActiveContext(expiredCtx)
	_, err2 := alertService.EvaluateAllRules(context.Background())
	if err2 != nil {
		red = true
	}

	if red {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.FailNow()
	}

	fmt.Println("GREEN (绿灯，缺陷已修复)")
}
