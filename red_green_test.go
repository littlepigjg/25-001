package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"logalert/pkg/syncutil"
)

func TestRedGreen(t *testing.T) {
	sm := syncutil.NewSafeMap()

	// 预先填充: 80 次 Set
	for i := 0; i < 80; i++ {
		sm.Set(fmt.Sprintf("key-%d", i), i)
	}

	var totalOps int64
	atomic.AddInt64(&totalOps, 80)

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Goroutine A: 持续 Set 操作 (4个)
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					for i := 0; i < 10; i++ {
						sm.Set(fmt.Sprintf("gen-%d-%d", gid, i), gid*100+i)
						atomic.AddInt64(&totalOps, 1)
					}
				}
			}
		}(g)
	}

	// Goroutine B: 持续 Delete 操作 (4个)
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					for i := 0; i < 8; i++ {
						k := fmt.Sprintf("key-%d", gid*10+i)
						sm.Delete(k)
						atomic.AddInt64(&totalOps, 1)
					}
				}
			}
		}(g)
	}

	// Goroutine C: 持续 Keys 操作 (4个)
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					for i := 0; i < 20; i++ {
						sm.Keys()
						atomic.AddInt64(&totalOps, 1)
					}
				}
			}
		}(g)
	}

	time.Sleep(300 * time.Millisecond)
	close(done)
	wg.Wait()

	// 检查 opCounter 一致性 (数据竞争导致更新丢失)
	actualOps := sm.CountOps()
	expectedOps := atomic.LoadInt64(&totalOps)

	if actualOps < expectedOps {
		t.Logf("opCounter 不一致: 期望=%d, 实际=%d, 丢失=%d",
			expectedOps, actualOps, expectedOps-actualOps)
		fmt.Println("RED (红灯，缺陷未修复)")
		t.FailNow()
		return
	}

	// 检查 Keys 结果一致性 (TOCTOU)
	finalKeys := sm.Keys()
	for _, k := range finalKeys {
		if _, ok := sm.Get(k); !ok {
			t.Logf("Keys() 返回了不存在的 key: %s", k)
			fmt.Println("RED (红灯，缺陷未修复)")
			t.FailNow()
			return
		}
	}

	fmt.Println("GREEN (绿灯，缺陷已修复)")
}
