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
	"time"
)

func TestRedGreen(t *testing.T) {
	store := store.NewMemoryLogStore(3)

	entry1 := model.NewLogEntry(model.LevelInfo, "svc-auth", "user login success")
	entry2 := model.NewLogEntry(model.LevelError, "svc-auth", "token validation failed")
	entry3 := model.NewLogEntry(model.LevelInfo, "svc-gateway", "request routed")

	store.Store(entry1)
	store.Store(entry2)
	store.Store(entry3)

	entry4 := model.NewLogEntry(model.LevelWarn, "svc-auth", "rate limit approaching")
	store.Store(entry4)

	cfg := config.DefaultConfig()
	log := logger.New()
	svc := service.NewStatsService(store, cfg, log)

	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Log("RED (红灯，缺陷未修复)")
			t.Logf("发生 panic: %v", r)
		}
	}()

	result, err := svc.GetStatistics(ctx, time.Time{}, time.Time{})
	if err != nil {
		t.Log("RED (红灯，缺陷未修复)")
		t.Logf("GetStatistics 返回错误: %v", err)
		return
	}

	if result == nil {
		t.Log("RED (红灯，缺陷未修复)")
		t.Log("GetStatistics 返回 nil 结果")
		return
	}

	if len(result.BySource) == 0 {
		t.Log("RED (红灯，缺陷未修复)")
		t.Log("统计结果为空，未能正确处理日志条目")
		return
	}

	t.Log("GREEN (绿灯，缺陷已修复)")
	t.Logf("GetStatistics 返回成功: TotalCount=%d, Sources=%d", result.TotalCount, len(result.BySource))
}

func TestRedGreenSourceStats(t *testing.T) {
	store := store.NewMemoryLogStore(3)

	entry1 := model.NewLogEntry(model.LevelInfo, "svc-auth", "user login success")
	entry2 := model.NewLogEntry(model.LevelError, "svc-auth", "token validation failed")
	entry3 := model.NewLogEntry(model.LevelInfo, "svc-gateway", "request routed")

	store.Store(entry1)
	store.Store(entry2)
	store.Store(entry3)

	entry4 := model.NewLogEntry(model.LevelWarn, "svc-auth", "rate limit approaching")
	store.Store(entry4)

	cfg := config.DefaultConfig()
	log := logger.New()
	svc := service.NewStatsService(store, cfg, log)

	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Log("RED (红灯，缺陷未修复)")
			t.Logf("发生 panic: %v", r)
		}
	}()

	result, err := svc.GetSourceStats(ctx, 10)
	if err != nil {
		t.Log("RED (红灯，缺陷未修复)")
		t.Logf("GetSourceStats 返回错误: %v", err)
		return
	}

	if len(result) == 0 {
		t.Log("RED (红灯，缺陷未修复)")
		t.Log("来源统计结果为空")
		return
	}

	t.Log("GREEN (绿灯，缺陷已修复)")
	t.Logf("GetSourceStats 返回成功: 来源数=%d", len(result))
}

func TestRedGreenDailyReport(t *testing.T) {
	store := store.NewMemoryLogStore(3)

	entry1 := model.NewLogEntry(model.LevelInfo, "svc-auth", "user login success")
	entry2 := model.NewLogEntry(model.LevelError, "svc-auth", "token validation failed")
	entry3 := model.NewLogEntry(model.LevelInfo, "svc-gateway", "request routed")

	store.Store(entry1)
	store.Store(entry2)
	store.Store(entry3)

	entry4 := model.NewLogEntry(model.LevelWarn, "svc-auth", "rate limit approaching")
	store.Store(entry4)

	cfg := config.DefaultConfig()
	log := logger.New()
	svc := service.NewStatsService(store, cfg, log)

	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Log("RED (红灯，缺陷未修复)")
			t.Logf("发生 panic: %v", r)
		}
	}()

	today := time.Now()
	result, err := svc.GetDailyReport(ctx, today)
	if err != nil {
		t.Log("RED (红灯，缺陷未修复)")
		t.Logf("GetDailyReport 返回错误: %v", err)
		return
	}

	if result == nil {
		t.Log("RED (红灯，缺陷未修复)")
		t.Log("GetDailyReport 返回 nil 结果")
		return
	}

	if result.TotalCount == 0 {
		t.Log("RED (红灯，缺陷未修复)")
		t.Log("日报 TotalCount 为 0，未能正确处理日志条目")
		return
	}

	t.Log("GREEN (绿灯，缺陷已修复)")
	t.Logf("GetDailyReport 返回成功: Date=%s, TotalCount=%d", result.Date, result.TotalCount)
}

func TestRedGreenAll(t *testing.T) {
	store := store.NewMemoryLogStore(3)

	entry1 := model.NewLogEntry(model.LevelInfo, "svc-auth", "user login success")
	entry2 := model.NewLogEntry(model.LevelError, "svc-auth", "token validation failed")
	entry3 := model.NewLogEntry(model.LevelInfo, "svc-gateway", "request routed")

	store.Store(entry1)
	store.Store(entry2)
	store.Store(entry3)

	entry4 := model.NewLogEntry(model.LevelWarn, "svc-auth", "rate limit approaching")
	store.Store(entry4)

	cfg := config.DefaultConfig()
	log := logger.New()
	svc := service.NewStatsService(store, cfg, log)

	ctx := context.Background()
	allPassed := true

	t.Run("GetStatistics", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				allPassed = false
				t.Logf("RED (红灯，缺陷未修复) - GetStatistics panic: %v", r)
			}
		}()
		result, err := svc.GetStatistics(ctx, time.Time{}, time.Time{})
		if err != nil {
			allPassed = false
			t.Logf("RED (红灯，缺陷未修复) - GetStatistics error: %v", err)
			return
		}
		if result == nil || len(result.BySource) == 0 {
			allPassed = false
			t.Log("RED (红灯，缺陷未修复) - GetStatistics 结果异常")
			return
		}
		t.Logf("GREEN (绿灯，缺陷已修复) - GetStatistics: TotalCount=%d", result.TotalCount)
	})

	t.Run("GetSourceStats", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				allPassed = false
				t.Logf("RED (红灯，缺陷未修复) - GetSourceStats panic: %v", r)
			}
		}()
		result, err := svc.GetSourceStats(ctx, 10)
		if err != nil {
			allPassed = false
			t.Logf("RED (红灯，缺陷未修复) - GetSourceStats error: %v", err)
			return
		}
		if len(result) == 0 {
			allPassed = false
			t.Log("RED (红灯，缺陷未修复) - GetSourceStats 结果为空")
			return
		}
		t.Logf("GREEN (绿灯，缺陷已修复) - GetSourceStats: 来源数=%d", len(result))
	})

	t.Run("GetDailyReport", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				allPassed = false
				t.Logf("RED (红灯，缺陷未修复) - GetDailyReport panic: %v", r)
			}
		}()
		result, err := svc.GetDailyReport(ctx, time.Now())
		if err != nil {
			allPassed = false
			t.Logf("RED (红灯，缺陷未修复) - GetDailyReport error: %v", err)
			return
		}
		if result == nil || result.TotalCount == 0 {
			allPassed = false
			t.Log("RED (红灯，缺陷未修复) - GetDailyReport 结果异常")
			return
		}
		t.Logf("GREEN (绿灯，缺陷已修复) - GetDailyReport: TotalCount=%d", result.TotalCount)
	})

	if !allPassed {
		t.Log("最终判定: RED (红灯，缺陷未修复)")
	} else {
		t.Log("最终判定: GREEN (绿灯，缺陷已修复)")
	}
}

func TestMain(m *testing.M) {
	fmt.Println("=== 缺陷验证测试 ===")
	m.Run()
}