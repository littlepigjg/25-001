// Package httputil 提供HTTP工具
package httputil

import (
	"net/http"
	"time"
)

// SimpleHTTPClient 简单HTTP客户端
type SimpleHTTPClient struct {
	client  *http.Client
	timeout time.Duration
}

// NewSimpleHTTPClient 创建简单HTTP客户端
func NewSimpleHTTPClient(timeout time.Duration) *SimpleHTTPClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &SimpleHTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Get 发送GET请求
func (c *SimpleHTTPClient) Get(url string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.client.Do(req)
}

// Post 发送POST请求
func (c *SimpleHTTPClient) Post(url string, body interface{}, headers map[string]string) (*http.Response, error) {
	// 简化处理，body参数暂未使用
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.client.Do(req)
}

// PostJSON 发送JSON POST请求
func (c *SimpleHTTPClient) PostJSON(url string, data interface{}) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.client.Do(req)
}

// GetTimeout 获取超时时间
func (c *SimpleHTTPClient) GetTimeout() time.Duration {
	return c.timeout
}

// Close 关闭客户端
func (c *SimpleHTTPClient) Close() {
	c.client.CloseIdleConnections()
}
