package main

import (
	"context"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"sync"
	"testing"
	"time"
)

func TestRedGreen(t *testing.T) {
	const numIterations = 5
	var totalURLMissing, totalAlertMissing int

	for iteration := 0; iteration < numIterations; iteration++ {
		cfg := config.Default()
		cfg.Storage.URLFilePath(fmt.Sprintf("/tmp/test_redgreen_urls_%d.json", iteration))
		cfg.Storage.LogFilePath(fmt.Sprintf("/tmp/test_redgreen_logs_%d.json", iteration))

		us, err := store.NewURLStore(cfg)
		if err != nil {
			t.Fatalf("NewURLStore failed: %v", err)
		}

		ctx := context.Background()
		if err := us.Load(ctx); err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		svc, err := service.NewURLService(cfg, us)
		if err != nil {
			t.Fatalf("NewURLService failed: %v", err)
		}

		// Test URLStore - concurrent Save with same RawURL
		numURLs := 60
		const sharedRawURL = "https://example.com/shared-resource"

		start := make(chan struct{})
		var urlWg sync.WaitGroup
		urlErrChan := make(chan error, numURLs)

		for i := 0; i < numURLs; i++ {
			urlWg.Add(1)
			go func(idx int) {
				defer urlWg.Done()
				<-start
				req := &model.CreateReq{
					RawURL:    sharedRawURL,
					CustomCode: fmt.Sprintf("code-%d-%d", iteration, idx),
					MaxVisits: 200,
				}
				_, err := svc.Create(ctx, req)
				if err != nil {
					urlErrChan <- fmt.Errorf("URL %d: %v", idx, err)
				}
			}(i)
		}

		close(start)
		urlWg.Wait()
		close(urlErrChan)

		for e := range urlErrChan {
			t.Fatalf("Iteration %d Create error: %v", iteration, e)
		}

		time.Sleep(30 * time.Millisecond)
		snapshot := us.RawSnapshot()
		totalURLMissing += numURLs - len(snapshot)

		us.Close()

		// Test MemoryAlertStore - concurrent Store with same RuleID/Source
		alertStore := store.NewMemoryAlertStore()
		numAlerts := 60
		const sharedRuleID = "rule-shared-001"
		const sharedSource = "source-shared-001"

		start2 := make(chan struct{})
		var alertWg sync.WaitGroup
		alertErrChan := make(chan error, numAlerts)

		for i := 0; i < numAlerts; i++ {
			alertWg.Add(1)
			go func(idx int) {
				defer alertWg.Done()
				<-start2
				event := &model.AlertEvent{
					ID:         fmt.Sprintf("alert-%d-%d", iteration, idx),
					RuleID:     sharedRuleID,
					RuleName:   "Shared Rule",
					Source:     sharedSource,
					Level:      model.LevelError,
					Count:      10,
					Threshold:  5,
					TriggeredAt: time.Now(),
					Status:     model.AlertFired,
				}
				err := alertStore.Store(event)
				if err != nil {
					alertErrChan <- fmt.Errorf("Alert %d: %v", idx, err)
				}
			}(i)
		}

		close(start2)
		alertWg.Wait()
		close(alertErrChan)

		for e := range alertErrChan {
			t.Fatalf("Iteration %d Store error: %v", iteration, e)
		}

		time.Sleep(30 * time.Millisecond)
		activeAlerts, err := alertStore.ListActive()
		if err != nil {
			t.Fatalf("ListActive failed: %v", err)
		}
		totalAlertMissing += numAlerts - len(activeAlerts)
	}

	if totalURLMissing > 0 || totalAlertMissing > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）\n")
		fmt.Printf("URLStore: 累计期望 %d, 实际缺失 %d\n", 60*numIterations, totalURLMissing)
		fmt.Printf("MemoryAlertStore: 累计期望 %d, 实际缺失 %d\n", 60*numIterations, totalAlertMissing)
		if totalURLMissing > 0 {
			t.Errorf("URLStore: 累计丢失 %d 个URL", totalURLMissing)
		}
		if totalAlertMissing > 0 {
			t.Errorf("MemoryAlertStore: 累计丢失 %d 个告警", totalAlertMissing)
		}
		t.FailNow()
	} else {
		fmt.Printf("GREEN（绿灯，缺陷已修复）\n")
	}
}
