// Package logger 提供结构化日志功能
package logger

import (
	"io"
	"log"
	"os"
	"sync"
)

// Sink 日志输出目标
type Sink interface {
	// Write 写入日志
	Write(p []byte) (n int, err error)
	// Close 关闭
	Close() error
}

// FileSink 文件日志输出
type FileSink struct {
	mu   sync.Mutex
	file *os.File
}

// NewFileSink 创建文件日志输出
func NewFileSink(path string) (*FileSink, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &FileSink{file: f}, nil
}

// Write 写入日志
func (fs *FileSink) Write(p []byte) (n int, err error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.file.Write(p)
}

// Close 关闭文件
func (fs *FileSink) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.file.Close()
}

// MultiSink 多输出目标
type MultiSink struct {
	sinks []Sink
}

// NewMultiSink 创建多输出目标
func NewMultiSink(sinks ...Sink) *MultiSink {
	return &MultiSink{sinks: sinks}
}

// Write 写入所有输出目标
func (ms *MultiSink) Write(p []byte) (n int, err error) {
	for _, sink := range ms.sinks {
		n, err = sink.Write(p)
		if err != nil {
			return n, err
		}
	}
	return len(p), nil
}

// Close 关闭所有输出
func (ms *MultiSink) Close() error {
	for _, sink := range ms.sinks {
		if err := sink.Close(); err != nil {
			return err
		}
	}
	return nil
}

// MemorySink 内存日志输出（用于测试）
type MemorySink struct {
	mu    sync.Mutex
	lines []string
}

// NewMemorySink 创建内存日志输出
func NewMemorySink() *MemorySink {
	return &MemorySink{lines: make([]string, 0)}
}

// Write 写入日志
func (ms *MemorySink) Write(p []byte) (n int, err error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.lines = append(ms.lines, string(p))
	return len(p), nil
}

// Close 关闭
func (ms *MemorySink) Close() error {
	return nil
}

// Lines 获取所有日志行
func (ms *MemorySink) Lines() []string {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	result := make([]string, len(ms.lines))
	copy(result, ms.lines)
	return result
}

// Clear 清空日志
func (ms *MemorySink) Clear() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.lines = ms.lines[:0]
}

// DiscardSink 丢弃日志输出
type DiscardSink struct{}

// Write 写入（丢弃）
func (d *DiscardSink) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// Close 关闭
func (d *DiscardSink) Close() error {
	return nil
}

// Ensure interfaces are satisfied
var _ io.Writer = (*FileSink)(nil)
var _ io.Writer = (*MultiSink)(nil)
var _ io.Writer = (*MemorySink)(nil)
var _ io.Writer = (*DiscardSink)(nil)
var _ Sink = (*FileSink)(nil)
var _ Sink = (*MultiSink)(nil)
var _ Sink = (*MemorySink)(nil)
var _ Sink = (*DiscardSink)(nil)

// Compile-time check
var _ = log.New(os.Stdout, "", 0)
