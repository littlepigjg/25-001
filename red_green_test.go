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
	ctx := context.Background()
	cfg := config.DefaultConfig()
	log := logger.New()
	log.SetLevel(logger.LevelError)

	alertStore := store.NewMemoryAlertStore()
	logStore := store.NewMemoryLogStore(1000)
	ruleStore := store.NewMemoryRuleStore()

	svc := service.NewAlertService(alertStore, logStore, ruleStore, cfg, log)

	now := time.Now()
	t1 := now.Add(-3 * time.Hour)
	t2 := now.Add(-2 * time.Hour)
	t3 := now.Add(-1 * time.Hour)

	events := []*model.AlertEvent{
		{ID: "evt-001", RuleID: "rule-1", Source: "app", Level: model.LevelError, TriggeredAt: t1, Status: model.AlertFired, Count: 10, Threshold: 5},
		{ID: "evt-002", RuleID: "rule-1", Source: "app", Level: model.LevelError, TriggeredAt: t2, Status: model.AlertFired, Count: 8, Threshold: 5},
		{ID: "evt-003", RuleID: "rule-1", Source: "app", Level: model.LevelError, TriggeredAt: t3, Status: model.AlertFired, Count: 15, Threshold: 5},
	}

	for _, evt := range events {
		if err := alertStore.Store(evt); err != nil {
			t.Fatalf("failed to store event %s: %v", evt.ID, err)
		}
	}

	time.Sleep(10 * time.Millisecond)

	latest, err := svc.GetLatestActiveAlert(ctx)

	if err != nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Printf("排序验证失败: %v\n", err)
		os.Exit(1)
	}

	if latest == nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Println("返回的最新告警为nil")
		os.Exit(1)
	}

	if latest.ID != "evt-003" {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Printf("期望最新告警ID为evt-003(t3=%s)，实际为%s(t=%s)\n",
			t3.Format(time.RFC3339),
			latest.ID,
			latest.TriggeredAt.Format(time.RFC3339))
		os.Exit(1)
	}

	listEvents, err := alertStore.ListActive()
	if err != nil {
		t.Fatalf("ListActive failed: %v", err)
	}

	for i := 1; i < len(listEvents); i++ {
		prev := listEvents[i-1]
		curr := listEvents[i]
		if prev.TriggeredAt.Before(curr.TriggeredAt) {
			fmt.Println("RED（红灯，缺陷未修复）")
			fmt.Printf("排序错误: %s(t=%s) 应在 %s(t=%s) 之后\n",
				prev.ID, prev.TriggeredAt.Format(time.RFC3339),
				curr.ID, curr.TriggeredAt.Format(time.RFC3339))
			os.Exit(1)
		}
		if prev.TriggeredAt.Equal(curr.TriggeredAt) && prev.ID > curr.ID {
			fmt.Println("RED（红灯，缺陷未修复）")
			fmt.Printf("同时间戳排序错误: %s 应在 %s 之前\n", curr.ID, prev.ID)
			os.Exit(1)
		}
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
