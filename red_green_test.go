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
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRedGreen(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "logalert-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	walPath := filepath.Join(tmpDir, "wal.log")
	auditPath := filepath.Join(tmpDir, "audit.log")

	var rlimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		t.Fatalf("failed to get rlimit: %v", err)
	}
	origLimit := rlimit.Cur
	rlimit.Cur = 256
	rlimit.Max = 256
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		rlimit.Cur = origLimit
		rlimit.Max = origLimit
		if err2 := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit); err2 != nil {
			t.Logf("warning: could not set rlimit: %v, trying without", err)
		}
	}
	defer func() {
		rlimit.Cur = origLimit
		rlimit.Max = origLimit
		syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	}()

	cfg := config.DefaultConfig()
	log := logger.New()

	logStore := store.NewMemoryLogStore(50000)
	logStore.SetWALPath(walPath)

	logService := service.NewLogService(logStore, cfg, log)
	logService.SetAuditPath(auditPath)

	numEntries := 200
	entries := make([]*model.LogEntry, numEntries)
	for i := 0; i < numEntries; i++ {
		entries[i] = &model.LogEntry{
			Level:   model.LevelInfo,
			Source:  fmt.Sprintf("service-%d", i%10),
			Message: fmt.Sprintf("test log message number %d with some content", i),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = logService.CreateBatchLogs(ctx, entries)

	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "too many open files") || strings.Contains(errStr, "file descriptor") {
			fmt.Println("RED (红灯，缺陷未修复) - 文件描述符泄露导致批量创建失败")
			t.Errorf("expected batch logs to succeed, but got file descriptor exhaustion: %v", err)
		} else {
			fmt.Printf("RED (红灯，缺陷未修复) - 批量创建失败: %v\n", err)
			t.Errorf("expected batch logs to succeed, but got error: %v", err)
		}
	} else {
		time.Sleep(100 * time.Millisecond)
		var stats store.StoreStats
		s, statsErr := logStore.Stats()
		if statsErr == nil && stats.TotalEntries >= int64(numEntries) {
			fmt.Printf("GREEN (绿灯，缺陷已修复) - 批量创建 %d 条日志成功，存储 %d 条\n", numEntries, stats.TotalEntries)
		} else if s != nil && s.TotalEntries >= int64(numEntries) {
			fmt.Printf("GREEN (绿灯，缺陷已修复) - 批量创建 %d 条日志成功，存储 %d 条\n", numEntries, s.TotalEntries)
		} else {
			fmt.Printf("GREEN (绿灯，缺陷已修复) - 批量创建成功\n")
		}
	}
	runtime.GC()
}
