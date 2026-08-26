package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/handler"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/httputil"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	logStore := store.NewMemoryLogStore(10000)

	cfg := config.DefaultConfig()
	logSvc := service.NewLogService(logStore, cfg, logger.New())
	logHandler := handler.NewLogHandler(logSvc, logger.New())

	timeoutMiddleware := httputil.TimeoutMiddleware(50 * time.Millisecond)

	wrappedHandler := timeoutMiddleware(http.HandlerFunc(logHandler.CreateLog))
	server := httptest.NewServer(wrappedHandler)
	defer server.Close()

	reqBody := `{"level":"INFO","source":"test_source","message":"this is a test log message for timeout verification"}`
	resp, err := http.Post(server.URL, "application/json", strings.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	time.Sleep(400 * time.Millisecond)

	stats, err := logStore.Stats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.TotalEntries > 0 {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Errorf("RED (红灯，缺陷未修复) - 日志在超时后仍被写入，TotalEntries=%d, HTTP响应码=%d", stats.TotalEntries, resp.StatusCode)
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	}
}

func TestRedGreenMultiple(t *testing.T) {
	logStore := store.NewMemoryLogStore(10000)

	cfg := config.DefaultConfig()
	logSvc := service.NewLogService(logStore, cfg, logger.New())
	logHandler := handler.NewLogHandler(logSvc, logger.New())

	timeoutMiddleware := httputil.TimeoutMiddleware(50 * time.Millisecond)

	wrappedHandler := timeoutMiddleware(http.HandlerFunc(logHandler.CreateLog))
	server := httptest.NewServer(wrappedHandler)
	defer server.Close()

	for i := 0; i < 5; i++ {
		body := fmt.Sprintf(`{"level":"WARN","source":"src_%d","message":"message number %d"}`, i, i)
		resp, err := http.Post(server.URL, "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()
	}

	time.Sleep(600 * time.Millisecond)

	stats, err := logStore.Stats()
	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.TotalEntries > 0 {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Errorf("RED (红灯，缺陷未修复) - 并发请求超时后日志仍被写入，TotalEntries=%d", stats.TotalEntries)
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	}
}
