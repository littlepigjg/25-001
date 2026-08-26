package main

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
	// 初始化配置和组件
	cfg := config.DefaultConfig()
	cfg.Storage.WriteBufferSize = 1000

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxEntries)
	log := logger.New()
	svc := service.NewLogService(logStore, cfg, log)

	// 创建测试数据：1000 条日志，3 个来源交替
	// 消息中使用固定关键词，不使用唯一数字
	entries := make([]*model.LogEntry, 0, 1000)
	for i := 0; i < 1000; i++ {
		entry := model.NewLogEntry(
			model.LevelInfo,
			fmt.Sprintf("svc-%d", i%3),
			"processing request with error warning timeout keywords buffer overflow slow response",
		)
		entries = append(entries, entry)
	}

	// 批量存储
	ctx := context.Background()
	_, err := svc.CreateBatchLogs(ctx, entries)
	if err != nil {
		t.Fatalf("CreateBatchLogs failed: %v", err)
	}

	// 检查切片扩容次数
	growthCount := logStore.SliceGrowthCount()
	if growthCount > 100 {
		fmt.Printf("RED（红灯，缺陷未修复） - sliceGrowthCount=%d\n", growthCount)
		t.FailNow()
	} else {
		fmt.Printf("GREEN（绿灯，缺陷已修复） - sliceGrowthCount=%d\n", growthCount)
	}
}
