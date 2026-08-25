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
	log := logger.New()
	memStore := store.NewMemoryLogStore(1000)

	memStore.SetPanicGuard(func(entry *model.LogEntry) bool {
		return entry == nil || entry.Message == ""
	})

	svc := service.NewLogService(memStore, cfg, log)
	ctx := context.Background()

	t.Run("CreateLog should work with metadata", func(t *testing.T) {
		entry, err := svc.CreateLog(ctx, model.LevelInfo, "test-service", "hello world", nil)
		if err != nil {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("CreateLog returned error: %v", err)
		}
		if entry == nil {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("CreateLog returned nil entry")
		}
		if !entry.HasMetadata("service") {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("Expected metadata 'service' to be set")
		}
		if !entry.HasMetadata("operation") {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("Expected metadata 'operation' to be set")
		}
		fmt.Printf("GREEN（绿灯，缺陷已修复）\n")
	})

	t.Run("CreateBatchLogs should work with metadata", func(t *testing.T) {
		entries := []*model.LogEntry{
			model.NewLogEntry(model.LevelWarn, "batch-service", "batch message 1"),
			model.NewLogEntry(model.LevelError, "batch-service", "batch message 2"),
		}
		results, err := svc.CreateBatchLogs(ctx, entries)
		if err != nil {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("CreateBatchLogs returned error: %v", err)
		}
		if len(results) != 2 {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("Expected 2 results, got %d", len(results))
		}
		for _, r := range results {
			if !r.HasMetadata("batch") {
				fmt.Printf("RED（红灯，缺陷未修复）\n")
				t.Fatalf("Expected metadata 'batch' to be set on entry %s", r.ID)
			}
		}
		fmt.Printf("GREEN（绿灯，缺陷已修复）\n")
	})

	t.Run("StoreWithGuard should store entry", func(t *testing.T) {
		entry := model.NewLogEntry(model.LevelInfo, "guard-service", "guard test message")
		err := memStore.StoreWithGuard(entry)
		if err != nil {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("StoreWithGuard returned error: %v", err)
		}
		fmt.Printf("GREEN（绿灯，缺陷已修复）\n")
	})

	t.Run("RawSnapshot should return stored entries", func(t *testing.T) {
		snapshot := memStore.RawSnapshot()
		if len(snapshot) == 0 {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("Expected snapshot to have entries")
		}
		fmt.Printf("GREEN（绿灯，缺陷已修复）\n")
	})

	t.Run("IncrementVisitsWithGuard should work", func(t *testing.T) {
		memStore2 := store.NewMemoryLogStore(100)
		entry := model.NewLogEntry(model.LevelInfo, "visit-service", "visit count test")
		err := memStore2.Store(entry)
		if err != nil {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("Store returned error: %v", err)
		}
		err = memStore2.IncrementVisitsWithGuard(entry.ID)
		if err != nil {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("IncrementVisitsWithGuard returned error: %v", err)
		}
		got, err := memStore2.GetByID(entry.ID)
		if err != nil {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("GetByID returned error: %v", err)
		}
		if got.Metadata == nil || got.Metadata["access_count"] == "" {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Fatalf("Expected access_count metadata to be set")
		}
		fmt.Printf("GREEN（绿灯，缺陷已修复）\n")
	})

	fmt.Println("All tests completed")
}