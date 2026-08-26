package logalert

import (
	"fmt"
	"logalert/internal/model"
	"logalert/internal/store"
	"sync"
	"testing"
)

func TestRedGreen(t *testing.T) {
	maxEntries := 1000
	s := store.NewMemoryLogStore(maxEntries)

	for i := 0; i < 48; i++ {
		entry := model.NewLogEntry(model.LevelInfo, fmt.Sprintf("init-src-%d", i), fmt.Sprintf("init-msg-%d", i))
		s.Store(entry)
	}

	var wg sync.WaitGroup
	var failMu sync.Mutex
	panicCount := 0

	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					failMu.Lock()
					panicCount++
					failMu.Unlock()
				}
			}()
			for j := 0; j < 5; j++ {
				entry := model.NewLogEntry(
					model.LogLevel(fmt.Sprintf("LEVEL-%d", idx%5)),
					fmt.Sprintf("src-%d", idx),
					fmt.Sprintf("msg-%d-%d", idx, j),
				)
				s.Store(entry)
			}
		}(i)
	}

	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					failMu.Lock()
					panicCount++
					failMu.Unlock()
				}
			}()
			for j := 0; j < 5; j++ {
				s.Stats()
			}
		}()
	}

	wg.Wait()

	failMu.Lock()
	pc := panicCount
	failMu.Unlock()

	if pc > 0 || t.Failed() {
		t.Log("RED (红灯，缺陷未修复)")
		t.Fail()
	} else {
		t.Log("GREEN (绿灯，缺陷已修复)")
	}
}
