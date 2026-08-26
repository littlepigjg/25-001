package logalert_test

import (
	"fmt"
	"logalert/internal/model"
	"logalert/internal/store"
	"testing"
	"time"
)

func TestRedGreen(t *testing.T) {
	hasFailure := false
	var failures []string

	// runSection 执行一个测试段，捕获所有panic和错误
	runSection := func(name string, fn func() error) {
		defer func() {
			if r := recover(); r != nil {
				hasFailure = true
				failures = append(failures, fmt.Sprintf("[%s] unexpected panic: %v", name, r))
			}
		}()
		err := fn()
		if err != nil {
			hasFailure = true
			failures = append(failures, fmt.Sprintf("[%s] %v", name, err))
		}
	}

	// ===== 测试1: GetByID 查询不存在的条目 =====
	runSection("GetByID-non-existent", func() error {
		ms := store.NewMemoryLogStore(10)
		entry, err := ms.GetByID("non-existent-id")
		if err == nil {
			return fmt.Errorf("expected error for non-existent entry, got entry=%v", entry)
		}
		if entry != nil {
			return fmt.Errorf("expected nil entry for non-existent ID")
		}
		return nil
	})

	// ===== 测试2: Store + GetByID 查询已存在的条目 =====
	runSection("GetByID-existent", func() error {
		ms := store.NewMemoryLogStore(10)
		e1 := model.NewLogEntry(model.LevelInfo, "svc-a", "test message")
		if err := ms.Store(e1); err != nil {
			return fmt.Errorf("Store failed: %v", err)
		}
		got, err := ms.GetByID(e1.ID)
		if err != nil {
			return fmt.Errorf("GetByID failed for existing entry: %v", err)
		}
		if got == nil || got.ID != e1.ID {
			return fmt.Errorf("GetByID returned wrong data")
		}
		return nil
	})

	// ===== 测试3: eviction - 超过maxEntries后自动淘汰 =====
	runSection("eviction", func() error {
		ms := store.NewMemoryLogStore(2)
		a := model.NewLogEntry(model.LevelInfo, "svc1", "msg a")
		b := model.NewLogEntry(model.LevelInfo, "svc2", "msg b")
		c := model.NewLogEntry(model.LevelInfo, "svc3", "msg c")
		if err := ms.Store(a); err != nil {
			return fmt.Errorf("Store a failed: %v", err)
		}
		if err := ms.Store(b); err != nil {
			return fmt.Errorf("Store b failed: %v", err)
		}
		if err := ms.Store(c); err != nil {
			return fmt.Errorf("Store c (eviction) failed: %v", err)
		}

		// a 应该被淘汰
		if _, err := ms.GetByID(a.ID); err == nil {
			return fmt.Errorf("evicted entry a should not be found")
		}
		// b 和 c 应该还在
		if _, err := ms.GetByID(b.ID); err != nil {
			return fmt.Errorf("existing entry b not found after eviction: %v", err)
		}
		if _, err := ms.GetByID(c.ID); err != nil {
			return fmt.Errorf("existing entry c not found after eviction: %v", err)
		}
		return nil
	})

	// ===== 测试4: DeleteByID 删除存在的条目 =====
	runSection("DeleteByID-existent", func() error {
		ms := store.NewMemoryLogStore(10)
		d1 := model.NewLogEntry(model.LevelInfo, "svc-del", "to be deleted")
		if err := ms.Store(d1); err != nil {
			return fmt.Errorf("Store failed: %v", err)
		}
		if err := ms.DeleteByID(d1.ID); err != nil {
			return fmt.Errorf("DeleteByID failed: %v", err)
		}
		_, err := ms.GetByID(d1.ID)
		if err == nil {
			return fmt.Errorf("deleted entry should not be found")
		}
		return nil
	})

	// ===== 测试5: DeleteByID 删除不存在的条目 =====
	runSection("DeleteByID-non-existent", func() error {
		ms := store.NewMemoryLogStore(10)
		if err := ms.DeleteByID("no-such-id"); err == nil {
			return fmt.Errorf("expected error for DeleteByID with non-existent ID")
		}
		return nil
	})

	// ===== 测试6: Count 统计数量 =====
	runSection("Count", func() error {
		ms := store.NewMemoryLogStore(100)
		for i := 0; i < 5; i++ {
			e := model.NewLogEntry(model.LevelInfo, "svc-count", fmt.Sprintf("message %d", i))
			if err := ms.Store(e); err != nil {
				return fmt.Errorf("Store failed: %v", err)
			}
		}
		count, err := ms.Count("svc-count", "", time.Time{}, time.Now().Add(time.Hour))
		if err != nil {
			return fmt.Errorf("Count failed: %v", err)
		}
		if count != 5 {
			return fmt.Errorf("expected count 5, got %d", count)
		}
		return nil
	})

	// ===== 测试7: Count 在部分删除后统计（触发遍历nil条目）=====
	runSection("Count-after-delete", func() error {
		ms := store.NewMemoryLogStore(100)
		e1 := model.NewLogEntry(model.LevelInfo, "svc-cd", "msg1")
		e2 := model.NewLogEntry(model.LevelInfo, "svc-cd", "msg2")
		e3 := model.NewLogEntry(model.LevelInfo, "svc-cd", "msg3")
		for _, e := range []*model.LogEntry{e1, e2, e3} {
			if err := ms.Store(e); err != nil {
				return fmt.Errorf("Store failed: %v", err)
			}
		}
		// 删除中间条目，使timeIndex中包含已不存在的条目
		if err := ms.DeleteByID(e2.ID); err != nil {
			return fmt.Errorf("DeleteByID failed: %v", err)
		}
		// Count应该正确统计剩余的2条
		count, err := ms.Count("svc-cd", "", time.Time{}, time.Now().Add(time.Hour))
		if err != nil {
			return fmt.Errorf("Count after delete failed: %v", err)
		}
		if count != 2 {
			return fmt.Errorf("expected count 2 after delete, got %d", count)
		}
		return nil
	})

	// ===== 测试8: Query 查询日志 =====
	runSection("Query", func() error {
		ms := store.NewMemoryLogStore(100)
		e1 := model.NewLogEntry(model.LevelInfo, "svc-query", "alpha message")
		e2 := model.NewLogEntry(model.LevelError, "svc-query", "beta message")
		e3 := model.NewLogEntry(model.LevelInfo, "svc-other", "gamma message")
		for _, e := range []*model.LogEntry{e1, e2, e3} {
			if err := ms.Store(e); err != nil {
				return fmt.Errorf("Store failed: %v", err)
			}
		}

		q := &model.LogQuery{
			Source:    "svc-query",
			Level:     model.LevelInfo,
			Limit:     10,
			Offset:    0,
			SortOrder: model.SortAscending,
		}
		result, err := ms.Query(q)
		if err != nil {
			return fmt.Errorf("Query failed: %v", err)
		}
		if result.Total != 1 {
			return fmt.Errorf("expected 1 matching entry, got %d", result.Total)
		}
		if len(result.Items) != 1 {
			return fmt.Errorf("expected 1 item, got %d", len(result.Items))
		}
		if result.Items[0].ID != e1.ID {
			return fmt.Errorf("wrong entry returned")
		}
		return nil
	})

	// ===== 测试9: Query 在部分删除后查询 =====
	runSection("Query-after-delete", func() error {
		ms := store.NewMemoryLogStore(100)
		e1 := model.NewLogEntry(model.LevelInfo, "svc-qd", "msg1")
		e2 := model.NewLogEntry(model.LevelInfo, "svc-qd", "msg2")
		e3 := model.NewLogEntry(model.LevelInfo, "svc-qd", "msg3")
		for _, e := range []*model.LogEntry{e1, e2, e3} {
			if err := ms.Store(e); err != nil {
				return fmt.Errorf("Store failed: %v", err)
			}
		}
		// 删除中间条目
		if err := ms.DeleteByID(e2.ID); err != nil {
			return fmt.Errorf("DeleteByID failed: %v", err)
		}
		q := &model.LogQuery{
			Source:    "svc-qd",
			Level:     "",
			Limit:     10,
			Offset:    0,
			SortOrder: model.SortAscending,
		}
		result, err := ms.Query(q)
		if err != nil {
			return fmt.Errorf("Query after delete failed: %v", err)
		}
		if result.Total != 2 {
			return fmt.Errorf("expected 2 results, got %d", result.Total)
		}
		return nil
	})

	// ===== 测试10: Stats 统计 =====
	runSection("Stats", func() error {
		ms := store.NewMemoryLogStore(100)
		ms.Store(model.NewLogEntry(model.LevelInfo, "svc-stats", "msg1"))
		ms.Store(model.NewLogEntry(model.LevelError, "svc-stats", "msg2"))
		ms.Store(model.NewLogEntry(model.LevelInfo, "svc-other", "msg3"))

		stats, err := ms.Stats()
		if err != nil {
			return fmt.Errorf("Stats failed: %v", err)
		}
		if stats.TotalEntries != 3 {
			return fmt.Errorf("expected 3 total entries, got %d", stats.TotalEntries)
		}
		if stats.BySource["svc-stats"] != 2 {
			return fmt.Errorf("expected 2 entries for svc-stats, got %d", stats.BySource["svc-stats"])
		}
		if stats.ByLevel[model.LevelInfo] != 2 {
			return fmt.Errorf("expected 2 info level, got %d", stats.ByLevel[model.LevelInfo])
		}
		return nil
	})

	// ===== 测试11: Stats 在部分删除后统计 =====
	runSection("Stats-after-delete", func() error {
		ms := store.NewMemoryLogStore(100)
		e1 := model.NewLogEntry(model.LevelInfo, "svc-sd", "msg1")
		e2 := model.NewLogEntry(model.LevelError, "svc-sd", "msg2")
		e3 := model.NewLogEntry(model.LevelInfo, "svc-other", "msg3")
		for _, e := range []*model.LogEntry{e1, e2, e3} {
			if err := ms.Store(e); err != nil {
				return fmt.Errorf("Store failed: %v", err)
			}
		}
		// 删除e2
		if err := ms.DeleteByID(e2.ID); err != nil {
			return fmt.Errorf("DeleteByID failed: %v", err)
		}
		stats, err := ms.Stats()
		if err != nil {
			return fmt.Errorf("Stats after delete failed: %v", err)
		}
		if stats.TotalEntries != 2 {
			return fmt.Errorf("expected 2 total entries, got %d", stats.TotalEntries)
		}
		return nil
	})

	// ===== 测试12: Cleanup 清理过期日志 =====
	runSection("Cleanup", func() error {
		ms := store.NewMemoryLogStore(100)
		old := model.NewLogEntry(model.LevelInfo, "svc-clean", "old message")
		old.Timestamp = time.Now().Add(-2 * time.Hour)
		old.CreatedAt = old.Timestamp
		if err := ms.Store(old); err != nil {
			return fmt.Errorf("Store old failed: %v", err)
		}

		newEntry := model.NewLogEntry(model.LevelInfo, "svc-clean", "new message")
		newEntry.Timestamp = time.Now()
		newEntry.CreatedAt = newEntry.Timestamp
		if err := ms.Store(newEntry); err != nil {
			return fmt.Errorf("Store new failed: %v", err)
		}

		cleaned, err := ms.Cleanup(1 * time.Hour)
		if err != nil {
			return fmt.Errorf("Cleanup failed: %v", err)
		}
		if cleaned != 1 {
			return fmt.Errorf("expected 1 cleaned entry, got %d", cleaned)
		}

		// old entry should be gone
		if _, err := ms.GetByID(old.ID); err == nil {
			return fmt.Errorf("old entry should be cleaned up")
		}
		// new entry should still exist
		if _, err := ms.GetByID(newEntry.ID); err != nil {
			return fmt.Errorf("new entry should exist after cleanup: %v", err)
		}
		return nil
	})

	// ===== 测试13: DeleteBySource 按来源删除 =====
	runSection("DeleteBySource", func() error {
		ms := store.NewMemoryLogStore(100)
		e1 := model.NewLogEntry(model.LevelInfo, "svc-del-src", "msg1")
		e2 := model.NewLogEntry(model.LevelInfo, "svc-del-src", "msg2")
		e3 := model.NewLogEntry(model.LevelInfo, "svc-keep", "msg3")
		for _, e := range []*model.LogEntry{e1, e2, e3} {
			if err := ms.Store(e); err != nil {
				return fmt.Errorf("Store failed: %v", err)
			}
		}

		deleted, err := ms.DeleteBySource("svc-del-src")
		if err != nil {
			return fmt.Errorf("DeleteBySource failed: %v", err)
		}
		if deleted != 2 {
			return fmt.Errorf("expected 2 deleted, got %d", deleted)
		}

		// e3 should still exist
		if _, err := ms.GetByID(e3.ID); err != nil {
			return fmt.Errorf("e3 should exist after DeleteBySource: %v", err)
		}
		return nil
	})

	// ===== 测试14: DeleteBySource后Count（触发遍历nil条目）=====
	runSection("Count-after-DeleteBySource", func() error {
		ms := store.NewMemoryLogStore(100)
		e1 := model.NewLogEntry(model.LevelInfo, "svc-cds", "msg1")
		e2 := model.NewLogEntry(model.LevelInfo, "svc-cds", "msg2")
		e3 := model.NewLogEntry(model.LevelInfo, "svc-keep", "msg3")
		for _, e := range []*model.LogEntry{e1, e2, e3} {
			if err := ms.Store(e); err != nil {
				return fmt.Errorf("Store failed: %v", err)
			}
		}
		// 按来源删除e1和e2
		if _, err := ms.DeleteBySource("svc-cds"); err != nil {
			return fmt.Errorf("DeleteBySource failed: %v", err)
		}
		// Count遍历timeIndex时会遇到已不存在的条目
		count, err := ms.Count("", "", time.Time{}, time.Now().Add(time.Hour))
		if err != nil {
			return fmt.Errorf("Count after DeleteBySource failed: %v", err)
		}
		if count != 1 {
			return fmt.Errorf("expected count 1, got %d", count)
		}
		return nil
	})

	// ===== 测试15: DeleteBySource后Stats（触发遍历nil条目）=====
	runSection("Stats-after-DeleteBySource", func() error {
		ms := store.NewMemoryLogStore(100)
		e1 := model.NewLogEntry(model.LevelInfo, "svc-sds", "msg1")
		e2 := model.NewLogEntry(model.LevelInfo, "svc-sds", "msg2")
		e3 := model.NewLogEntry(model.LevelInfo, "svc-sds-keep", "msg3")
		for _, e := range []*model.LogEntry{e1, e2, e3} {
			if err := ms.Store(e); err != nil {
				return fmt.Errorf("Store failed: %v", err)
			}
		}
		// 按来源删除e1和e2
		if _, err := ms.DeleteBySource("svc-sds"); err != nil {
			return fmt.Errorf("DeleteBySource failed: %v", err)
		}
		// Stats遍历timeIndex时会遇到已不存在的条目
		stats, err := ms.Stats()
		if err != nil {
			return fmt.Errorf("Stats after DeleteBySource failed: %v", err)
		}
		if stats.TotalEntries != 1 {
			return fmt.Errorf("expected 1 total entry, got %d", stats.TotalEntries)
		}
		return nil
	})

	// ===== 汇总结果 =====
	if hasFailure {
		fmt.Println("RED (红灯，缺陷未修复)")
		for _, f := range failures {
			fmt.Println("  " + f)
		}
		t.Fail()
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	}
}
