package main

import (
	"fmt"
	"logalert/internal/model"
	"logalert/internal/store"
	"testing"
	"time"
)

func TestRedGreen(t *testing.T) {
	logStore := store.NewMemoryLogStore(1000)

	for i := 0; i < 5; i++ {
		entry := model.NewLogEntry(model.LevelInfo, "auth-service", "user login successful")
		entry.Timestamp = time.Now().Add(-time.Duration(i) * time.Minute)
		_ = logStore.Store(entry)
	}

	query := &model.LogQuery{
		Source:  "non-existent-service",
		Offset:  100,
		Limit:   10,
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.FailNow()
		}
	}()

	result, err := logStore.Query(query)
	if err != nil {
		fmt.Printf("RED（红灯，缺陷未修复）: %v\n", err)
		t.FailNow()
	}

	if result.Total == 0 && len(result.Items) == 0 {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	} else {
		fmt.Println("RED（红灯，缺陷未修复）: expected empty result")
		t.FailNow()
	}
}
