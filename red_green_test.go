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
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	logStore := store.NewMemoryLogStore(1000)
	log := logger.New()
	logService := service.NewLogService(logStore, cfg, log)

	ctx := context.Background()

	_, err := logService.CreateLog(ctx, model.LevelInfo, "auth", "user login success", nil)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}
	_, err = logService.CreateLog(ctx, model.LevelError, "payment", "payment failed", nil)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}
	_, err = logService.CreateLog(ctx, model.LevelWarn, "gateway", "slow response", nil)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}
	_, err = logService.CreateLog(ctx, model.LevelInfo, "auth", "user logout", nil)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}
	_, err = logService.CreateLog(ctx, model.LevelDebug, "scheduler", "task triggered", nil)
	if err != nil {
		t.Fatalf("Failed to create log: %v", err)
	}

	query := model.DefaultLogQuery()
	query.Limit = 10
	result, err := logService.QueryLogs(ctx, query)
	if err != nil {
		t.Fatalf("QueryLogs failed: %v", err)
	}

	if len(result.Items) != 5 {
		t.Fatalf("Expected 5 items, got %d", len(result.Items))
	}

	originalMessages := map[string]bool{
		"user login success":  false,
		"payment failed":      false,
		"slow response":       false,
		"user logout":         false,
		"task triggered":      false,
	}

	corruptionFound := false
	for _, item := range result.Items {
		if item == nil {
			corruptionFound = true
			break
		}
		if _, ok := originalMessages[item.Message]; ok {
			originalMessages[item.Message] = true
		} else {
			corruptionFound = true
			break
		}
	}

	if !corruptionFound {
		for _, found := range originalMessages {
			if !found {
				corruptionFound = true
				break
			}
		}
	}

	query2 := model.DefaultLogQuery()
	query2.Limit = 10
	result2, err := logService.QueryLogs(ctx, query2)
	if err != nil {
		t.Fatalf("Second QueryLogs failed: %v", err)
	}

	if len(result2.Items) != 5 {
		t.Fatalf("Expected 5 items on second query, got %d", len(result2.Items))
	}

	for _, item := range result2.Items {
		if item == nil {
			corruptionFound = true
			break
		}
	}

	if corruptionFound {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Fail()
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	}
}
