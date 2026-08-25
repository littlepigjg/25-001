package main

import (
	"context"
	"errors"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
	"testing"
)

func TestRedGreen(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.URLFilePath("/tmp/test_urls.json")
	cfg.Storage.LogFilePath("/tmp/test_logs.json")
	cfg.Storage.SyncInterval(0)
	cfg.Storage.FlushOnWrite(false)

	allGreen := true

	t.Run("errors.As can find NotFoundError in wrapped DeleteLog chain", func(t *testing.T) {
		logStore := store.NewMemoryLogStore(1000)
		logSvc := service.NewLogService(logStore, cfg, logger.New())

		ctx := context.Background()
		err := logSvc.DeleteLog(ctx, "nonexistent-id")

		if err == nil {
			t.Errorf("expected error for non-existent log deletion, got nil")
			return
		}

		var nfErr *model.NotFoundError
		if errors.As(err, &nfErr) {
			fmt.Println("GREEN（绿灯，缺陷已修复）")
		} else {
			fmt.Println("RED（红灯，缺陷未修复）")
			allGreen = false
			t.Errorf("errors.As failed to find NotFoundError in wrapped error chain")
		}
	})

	t.Run("errors.As can find NotFoundError in wrapped chain", func(t *testing.T) {
		logStore := store.NewMemoryLogStore(1000)
		logSvc := service.NewLogService(logStore, cfg, logger.New())

		ctx := context.Background()
		_, err := logSvc.GetLog(ctx, "nonexistent-id")

		if err == nil {
			t.Errorf("expected error for non-existent log, got nil")
			return
		}

		var nfErr *model.NotFoundError
		if errors.As(err, &nfErr) {
			fmt.Println("GREEN（绿灯，缺陷已修复）")
		} else {
			fmt.Println("RED（红灯，缺陷未修复）")
			allGreen = false
			t.Errorf("errors.As failed to find NotFoundError in wrapped error chain")
		}
	})

	t.Run("errors.As works through URLService create flow", func(t *testing.T) {
		urlStore, err := store.NewURLStore(cfg)
		if err != nil {
			t.Fatalf("failed to create URL store: %v", err)
		}
		urlStore.Load(context.Background())

		accessStore, err := store.NewAccessLogStore(cfg)
		if err != nil {
			t.Fatalf("failed to create access log store: %v", err)
		}
		accessStore.Open(context.Background())

		urlSvc, err := service.NewURLService(cfg, urlStore)
		if err != nil {
			t.Fatalf("failed to create URL service: %v", err)
		}

		ctx := context.Background()

		req1 := &model.CreateReq{
			RawURL:     "https://example.com/page1",
			CustomCode: "duptest",
			MaxVisits:  100,
		}
		_, err = urlSvc.Create(ctx, req1)
		if err != nil {
			t.Fatalf("first create failed: %v", err)
		}

		req2 := &model.CreateReq{
			RawURL:     "https://example.com/page2",
			CustomCode: "duptest",
			MaxVisits:  100,
		}
		_, err = urlSvc.Create(ctx, req2)
		if err == nil {
			t.Errorf("expected duplicate error, got nil")
			return
		}

		var conflictErr *model.ConflictError
		if errors.As(err, &conflictErr) {
			fmt.Println("GREEN（绿灯，缺陷已修复）")
		} else {
			fmt.Println("RED（红灯，缺陷未修复）")
			allGreen = false
			t.Errorf("errors.As could not find the original conflict error in wrapped chain")
		}

		redirectSvc, err := service.NewRedirectService(urlStore, accessStore)
		if err != nil {
			t.Fatalf("failed to create redirect service: %v", err)
		}

		_, err = redirectSvc.HandleRedirect(ctx, &service.RedirectRequest{Code: "nonexistent"})
		if err == nil {
			t.Errorf("expected not-found error for nonexistent code, got nil")
			return
		}
	})

	if allGreen {
		fmt.Println("=== FINAL: GREEN（绿灯，缺陷已修复） ===")
	} else {
		fmt.Println("=== FINAL: RED（红灯，缺陷未修复） ===")
	}
}
