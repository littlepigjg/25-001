package main

import (
	"context"
	"fmt"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"sync"
	"testing"
)

func TestRedGreen(t *testing.T) {
	fmt.Println("===========================================================")
	fmt.Println("Starting Red/Green Test for Concurrency Defect")
	fmt.Println("===========================================================")

	var wg sync.WaitGroup
	errCh := make(chan error, 20)

	store := store.NewMemoryRuleStore()
	log := logger.New()

	// 1. 初始化规则数据
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		rule := model.NewAlertRule(
			fmt.Sprintf("rule-%d", i),
			"source-test",
			model.LevelError,
			5,
			10,
		)
		if err := store.Create(rule); err != nil {
			t.Fatalf("Failed to create rule: %v", err)
		}
	}

	svc := service.NewRuleService(store, nil, log)
	if svc == nil {
		t.Fatal("Failed to create RuleService")
	}

	// 2. 并发启动：同时执行 Update 和 List 操作
	fmt.Println("Launching concurrent goroutines for Update and List...")
	
	// 并发 List
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, err := svc.ListRules(ctx)
				if err != nil {
					errCh <- fmt.Errorf("ListRules error: %v", err)
					return
				}
			}
		}()
	}

	// 并发 Update
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				rule := &model.AlertRule{
					ID:        fmt.Sprintf("rule-%d", idx%10),
					Name:      fmt.Sprintf("rule-%d", idx%10),
					Source:    "source-updated",
					Level:     model.LevelWarn,
					WindowMinutes: 5,
					Threshold: 10,
					Enabled:   true,
					Priority:  5,
				}
				_, err := svc.UpdateRule(ctx, rule)
				if err != nil {
					errCh <- fmt.Errorf("UpdateRule error: %v", err)
					return
				}
			}
		}(i)
	}

	// 并发 SyncRules (List + Update 组合)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, err := svc.SyncRules(ctx)
				if err != nil {
					errCh <- fmt.Errorf("SyncRules error: %v", err)
					return
				}
			}
		}()
	}

	// 3. 等待所有 goroutine 完成
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// 4. 收集错误
	hasError := false
	for err := range errCh {
		fmt.Printf("Error occurred: %v\n", err)
		hasError = true
	}

	// 5. 判定结果
	if hasError {
		fmt.Println("-----------------------------------------------------------")
		fmt.Println("RED (RED LIGHT - Defect Detected)")
		fmt.Println("Test FAILED: Concurrent map read/write race condition detected!")
		fmt.Println("This indicates the Update/List methods lack proper synchronization.")
		fmt.Println("-----------------------------------------------------------")
		t.FailNow()
	} else {
		// 额外验证数据完整性
		count, err := store.Count()
		if err != nil {
			fmt.Printf("Count verification failed: %v\n", err)
			fmt.Println("RED (RED LIGHT - Defect Detected)")
			t.FailNow()
		}
		
		if count < 10 {
			fmt.Printf("Data integrity check failed: expected 10 rules, got %d\n", count)
			fmt.Println("RED (RED LIGHT - Defect Detected)")
			t.FailNow()
		}
		
		fmt.Println("-----------------------------------------------------------")
		fmt.Println("GREEN (GREEN LIGHT - No Defect Detected)")
		fmt.Println("Test PASSED: No race condition detected, all operations completed successfully.")
		fmt.Printf("Final rule count: %d (expected >= 10)\n", count)
		fmt.Println("-----------------------------------------------------------")
	}
}

func TestConcurrentAccess(t *testing.T) {
	fmt.Println("Testing concurrent access to store...")
	
	s := store.NewMemoryRuleStore()
	
	// 预加载数据
	for i := 0; i < 5; i++ {
		rule := model.NewAlertRule(
			fmt.Sprintf("test-rule-%d", i),
			"src",
			model.LevelError,
			10,
			5,
		)
		s.Create(rule)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 50)

	// 并发更新
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				rule := &model.AlertRule{
					ID: fmt.Sprintf("test-rule-%d", goroutineID%5),
					Name: fmt.Sprintf("test-rule-%d", goroutineID%5),
					Source: "updated-src",
					Level: model.LevelWarn,
					WindowMinutes: 10,
					Threshold: 5,
					Enabled: true,
					Priority: 5,
				}
				if err := s.Update(rule); err != nil {
					errCh <- err
					return
				}
			}
		}(g)
	}

	// 并发列表
	for g := 0; g < 5; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				_, err := s.List()
				if err != nil {
					errCh <- err
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	hasError := false
	for err := range errCh {
		fmt.Printf("Error: %v\n", err)
		hasError = true
	}

	if hasError {
		fmt.Println("RED (Race condition detected)")
		t.Fail()
	} else {
		fmt.Println("GREEN (No race condition)")
	}
}
