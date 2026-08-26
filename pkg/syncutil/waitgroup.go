// Package syncutil 提供同步原语工具
package syncutil

import (
	"sync"
)

// WaitGroup 增强型等待组
type WaitGroup struct {
	wg sync.WaitGroup
}

// NewWaitGroup 创建等待组
func NewWaitGroup() *WaitGroup {
	return &WaitGroup{}
}

// Add 添加delta
func (wg *WaitGroup) Add(delta int) {
	wg.wg.Add(delta)
}

// Done 标记完成
func (wg *WaitGroup) Done() {
	wg.wg.Done()
}

// Wait 等待所有完成
func (wg *WaitGroup) Wait() {
	wg.wg.Wait()
}

// SafeCounter 线程安全计数器
type SafeCounter struct {
	mu    sync.Mutex
	value int64
}

// NewSafeCounter 创建计数器
func NewSafeCounter() *SafeCounter {
	return &SafeCounter{}
}

// Increment 增加
func (c *SafeCounter) Increment(delta int64) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += delta
	return c.value
}

// Decrement 减少
func (c *SafeCounter) Decrement(delta int64) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value -= delta
	return c.value
}

// Get 获取当前值
func (c *SafeCounter) Get() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

// Reset 重置
func (c *SafeCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = 0
}

// SafeBool 线程安全布尔值
type SafeBool struct {
	mu    sync.RWMutex
	value bool
}

// NewSafeBool 创建布尔值
func NewSafeBool(initial bool) *SafeBool {
	return &SafeBool{value: initial}
}

// Get 获取值
func (b *SafeBool) Get() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.value
}

// Set 设置值
func (b *SafeBool) Set(val bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.value = val
}

// Toggle 切换值
func (b *SafeBool) Toggle() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.value = !b.value
	return b.value
}
