package logalert

import (
	"context"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"testing"
	"time"
)

func TestRedGreen(t *testing.T) {
	cfg := config.Default()

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("failed to create URLStore: %v", err)
	}
	defer urlStore.Close()

	logStore, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("failed to create AccessLogStore: %v", err)
	}
	defer logStore.Close()

	ctx := context.Background()
	if err := urlStore.Load(ctx); err != nil {
		t.Fatalf("failed to load URLStore: %v", err)
	}
	if err := logStore.Open(ctx); err != nil {
		t.Fatalf("failed to open AccessLogStore: %v", err)
	}

	urlService, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URLService: %v", err)
	}

	redirectService, err := service.NewRedirectService(urlStore, logStore)
	if err != nil {
		t.Fatalf("failed to create RedirectService: %v", err)
	}

	createReq := &model.CreateReq{
		RawURL:    "https://example.com/test-page",
		CustomCode: "test01",
		MaxVisits: 100,
	}

	shortURL, err := urlService.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create short URL: %v", err)
	}
	if shortURL.Code != "test01" {
		t.Fatalf("expected code 'test01', got '%s'", shortURL.Code)
	}

	testPassed := true
	panicCaptured := false
	var panicValue interface{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicCaptured = true
				panicValue = r
			}
		}()

		req := &service.RedirectRequest{
			Code:      shortURL.Code,
			Timestamp: time.Now(),
		}

		result, err := redirectService.HandleRedirect(ctx, req)
		if err != nil {
			testPassed = false
			t.Logf("HandleRedirect returned error: %v", err)
			return
		}
		if result == nil {
			testPassed = false
			t.Log("HandleRedirect returned nil result")
			return
		}
		if result.Status != 302 {
			testPassed = false
			t.Logf("expected status 302, got %d", result.Status)
			return
		}
	}()

	if panicCaptured {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Logf("panic detected: %v", panicValue)
		t.Errorf("RED (红灯，缺陷未修复): panic occurred during HandleRedirect - %v", panicValue)
	} else if testPassed {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	} else {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Errorf("RED (红灯，缺陷未修复): test failed without panic")
	}

	snapshot := urlStore.RawSnapshot()
	if len(snapshot) != 1 {
		t.Errorf("expected 1 URL in snapshot, got %d", len(snapshot))
	}
}
