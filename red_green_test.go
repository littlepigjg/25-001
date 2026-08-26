package logalert

import (
	"context"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/internal/service"
	"logalert/pkg/logger"
	"os"
	"testing"
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Storage.MaxEntries = 5
	log := logger.New()

	memStore := store.NewMemoryLogStore(5)
	svc := service.NewLogService(memStore, cfg, log)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		entry := model.NewLogEntry(model.LevelInfo, "test-svc", fmt.Sprintf("msg-%d", i))
		_, err := svc.CreateLog(ctx, entry.Level, entry.Source, entry.Message, entry.Keywords)
		if err != nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			os.Exit(1)
		}
	}

	extra := model.NewLogEntry(model.LevelInfo, "test-svc", "msg-extra")
	_, err := svc.SyncAndVerify(ctx, []*model.LogEntry{extra})

	if err != nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		os.Exit(1)
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}
