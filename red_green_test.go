// 缺陷植入验收测试
// 缺陷ID: logalert-error-19
// 缺陷模式: err被:=遮蔽导致原始错误丢失
// 涉及文件:
//   1. internal/store/memory_log_store.go Query()
//   2. internal/service/alert_service.go EvaluateRule()
//
// 测试结论:
//   所有用例通过 = GREEN (缺陷已修复)
//   任何用例失败 = RED (缺陷存在)

package logalert_test

import (
	"context"
	"testing"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

// TestDefectErrShadowing 测试err被:=遮蔽导致原始错误丢失的缺陷
func TestDefectErrShadowing(t *testing.T) {
	t.Run("T1: Query应返回校验器错误(但被:=遮蔽丢失)", func(t *testing.T) {
		logStore := store.NewMemoryLogStore(1000)

		expectedErr := model.NewValidationError("query", "mock validator error")
		logStore.SetQueryValidator(func(query *model.LogQuery) error {
			return expectedErr
		})

		entry := model.NewLogEntry(model.LevelError, "srv-a", "err msg")
		logStore.Store(entry)

		query := model.DefaultLogQuery()
		query.Source = "srv-a"

		_, err := logStore.Query(query)

		if err == nil {
			t.Fatal("RED: Query() 未返回校验器错误——err被:=遮蔽丢失")
		}
		if err.Error() != expectedErr.Error() {
			t.Fatalf("RED: Query() 返回了错误但不是期望的错误, 期望: %v, 实际: %v", expectedErr, err)
		}
		// GREEN: err正确返回
	})

	t.Run("T2: EvaluateRule应传递Query错误(但被:=遮蔽丢失)", func(t *testing.T) {
		logStore := store.NewMemoryLogStore(1000)

		// 添加足够日志以满足阈值触发条件
		for i := 0; i < 5; i++ {
			entry := model.NewLogEntry(model.LevelError, "srv-b", "boom")
			logStore.Store(entry)
		}

		expectedErr := model.NewValidationError("query", "query failed")
		logStore.SetQueryValidator(func(query *model.LogQuery) error {
			return expectedErr
		})

		ruleStore := store.NewMemoryRuleStore()
		alertStore := store.NewMemoryAlertStore()
		cfg := config.DefaultConfig()
		log := logger.New()

		svc := service.NewAlertService(alertStore, logStore, ruleStore, cfg, log)

		rule := model.NewAlertRule("rule-b", "srv-b", model.LevelError, 5, 3)

		_, err := svc.EvaluateRule(context.Background(), rule)

		if err == nil {
			t.Fatal("RED: EvaluateRule() 未返回Query错误——err被:=遮蔽丢失")
		}
		if err.Error() != expectedErr.Error() {
			t.Fatalf("RED: EvaluateRule() 返回的错误不匹配, 期望: %v, 实际: %v", expectedErr, err)
		}
		// GREEN: err正确传递
	})

	t.Run("T3: 正常场景-无错误时Query正常返回结果", func(t *testing.T) {
		logStore := store.NewMemoryLogStore(1000)

		entry := model.NewLogEntry(model.LevelInfo, "srv-c", "ok")
		logStore.Store(entry)

		query := model.DefaultLogQuery()
		query.Source = "srv-c"

		result, err := logStore.Query(query)

		if err != nil {
			t.Fatalf("RED: 正常场景Query不应报错, 实际: %v", err)
		}
		if result == nil || result.Total != 1 {
			t.Fatalf("RED: 期望1条结果, 实际: %v", result)
		}
		// GREEN: 正常
	})

	t.Run("T4: 正常场景-无错误时EvaluateRule正常触发告警", func(t *testing.T) {
		logStore := store.NewMemoryLogStore(1000)

		for i := 0; i < 5; i++ {
			entry := model.NewLogEntry(model.LevelError, "srv-d", "fail")
			logStore.Store(entry)
		}

		ruleStore := store.NewMemoryRuleStore()
		alertStore := store.NewMemoryAlertStore()
		cfg := config.DefaultConfig()
		log := logger.New()

		svc := service.NewAlertService(alertStore, logStore, ruleStore, cfg, log)

		rule := model.NewAlertRule("rule-d", "srv-d", model.LevelError, 5, 3)

		event, err := svc.EvaluateRule(context.Background(), rule)

		if err != nil {
			t.Fatalf("RED: 正常场景EvaluateRule不应报错, 实际: %v", err)
		}
		if event == nil {
			t.Fatal("RED: 期望触发告警事件, 实际为nil")
		}
		// GREEN: 正常
	})
}