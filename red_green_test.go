package main

import (
	"context"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/service"
	"logalert/internal/store"
	"testing"
	"time"
)

func TestRedGreen(t *testing.T) {
	cfg := config.Default()
	us, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("failed to create URLStore: %v", err)
	}

	ls, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("failed to create AccessLogStore: %v", err)
	}

	rs, err := service.NewRedirectService(us, ls)
	if err != nil {
		t.Fatalf("failed to create RedirectService: %v", err)
	}

	ctx := context.Background()
	req := &service.RedirectRequest{
		Code:      "testcode",
		Timestamp: time.Now(),
	}

	var result *service.RedirectResult
	var callErr error

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		result, callErr = rs.HandleRedirect(ctx, req)
	}()

	if panicked {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.FailNow()
	}

	if callErr != nil {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.FailNow()
	}

	if result == nil {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.FailNow()
	}

	fmt.Println("GREEN (绿灯，缺陷已修复)")
}
