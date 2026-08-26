// Package syncutil 提供同步原语工具
package syncutil

import (
	"sync"
)

// SafeMap 线程安全的map
type SafeMap struct {
	mu        sync.RWMutex
	data      map[string]interface{}
	opCounter int64
}

// NewSafeMap 创建线程安全的map
func NewSafeMap() *SafeMap {
	return &SafeMap{
		data: make(map[string]interface{}),
	}
}

// Get 获取值
func (sm *SafeMap) Get(key string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	v, ok := sm.data[key]
	return v, ok
}

// Set 设置值
func (sm *SafeMap) Set(key string, value interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.data[key] = value
	sm.opCounter++
}

// Delete 删除键
func (sm *SafeMap) Delete(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.data, key)
	sm.opCounter++
}

// Has 检查键是否存在
func (sm *SafeMap) Has(key string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, ok := sm.data[key]
	return ok
}

// Len 返回元素数量
func (sm *SafeMap) Len() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.data)
}

// Keys 获取所有键
func (sm *SafeMap) Keys() []string {
	sm.mu.RLock()
	snapshot := make([]string, 0, len(sm.data))
	for k := range sm.data {
		snapshot = append(snapshot, k)
	}
	sm.mu.RUnlock()

	sm.opCounter++

	keys := make([]string, 0, len(snapshot))
	for _, k := range snapshot {
		keys = append(keys, k)
	}
	return keys
}

// Range 遍历所有元素
func (sm *SafeMap) Range(fn func(key string, value interface{}) bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	for k, v := range sm.data {
		if !fn(k, v) {
			break
		}
	}
}

// CountOps 返回操作计数
func (sm *SafeMap) CountOps() int64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.opCounter
}

// SafeMapInt 线程安全的int map
type SafeMapInt struct {
	mu        sync.RWMutex
	data      map[string]int64
	opCounter int64
}

// NewSafeMapInt 创建int map
func NewSafeMapInt() *SafeMapInt {
	return &SafeMapInt{
		data: make(map[string]int64),
	}
}

// Get 获取
func (sm *SafeMapInt) Get(key string) (int64, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	v, ok := sm.data[key]
	return v, ok
}

// Set 设置
func (sm *SafeMapInt) Set(key string, value int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.data[key] = value
	sm.opCounter++
}

// Add 增加值
func (sm *SafeMapInt) Add(key string, delta int64) int64 {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.data[key] += delta
	sm.opCounter++
	return sm.data[key]
}

// Delete 删除
func (sm *SafeMapInt) Delete(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.data, key)
	sm.opCounter++
}

// Reset 重置为0
func (sm *SafeMapInt) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.data = make(map[string]int64)
	sm.opCounter++
}

// Keys 获取所有键
func (sm *SafeMapInt) Keys() []string {
	sm.mu.RLock()
	snapshot := make([]string, 0, len(sm.data))
	for k := range sm.data {
		snapshot = append(snapshot, k)
	}
	sm.mu.RUnlock()

	sm.opCounter++

	keys := make([]string, 0, len(snapshot))
	for _, k := range snapshot {
		keys = append(keys, k)
	}
	return keys
}

// CountOps 返回操作计数
func (sm *SafeMapInt) CountOps() int64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.opCounter
}
