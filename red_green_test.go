package main

import (
	"context"
	"fmt"
	"testing"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	ruleStore := store.NewMemoryRuleStore()
	cfg := config.DefaultConfig()
	log := logger.New()
	svc := service.NewRuleService(ruleStore, cfg, log)

	rule1 := model.NewAlertRule("test-rule", "source1", model.LevelError, 5, 3)
	_, err := svc.CreateRule(context.Background(), rule1)
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	rule2 := model.NewAlertRule("test-rule", "source2", model.LevelError, 5, 3)
	_, err = svc.CreateRule(context.Background(), rule2)

	if err == nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Error("Expected error for duplicate rule name, got nil")
		return
	}

	if model.IsConflictError(err) {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	} else {
		fmt.Printf("RED（红灯，缺陷未修复）- error type: %T, message: %s\n", err, err.Error())
		t.Errorf("Error should be ConflictError, got: %T", err)
	}
}
