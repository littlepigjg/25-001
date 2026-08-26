// Package jsonutil 提供JSON读写工具
package jsonutil

import (
	"encoding/json"
	"io"
	"net/http"
)

// ReadBody 读取并解析请求体为指定类型
func ReadBody(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return &JSONError{Message: "request body is empty"}
	}
	defer r.Body.Close()
	return NewReader(r.Body).Decode(v)
}

// WriteJSON 写入JSON响应
func WriteJSON(w http.ResponseWriter, statusCode int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	return NewWriter(w).Encode(v)
}

// Reader JSON读取器
type Reader struct {
	decoder *json.Decoder
}

// NewReader 创建JSON读取器
func NewReader(r io.Reader) *Reader {
	return &Reader{decoder: json.NewDecoder(r)}
}

// Decode 解码JSON
func (reader *Reader) Decode(v interface{}) error {
	if err := reader.decoder.Decode(v); err != nil {
		if err == io.EOF {
			return &JSONError{Message: "empty JSON input"}
		}
		return &JSONError{Message: "invalid JSON: " + err.Error()}
	}
	return nil
}

// Writer JSON写入器
type Writer struct {
	encoder *json.Encoder
}

// NewWriter 创建JSON写入器
func NewWriter(w io.Writer) *Writer {
	return &Writer{encoder: json.NewEncoder(w)}
}

// Encode 编码为JSON
func (writer *Writer) Encode(v interface{}) error {
	return writer.encoder.Encode(v)
}

// JSONError JSON错误
type JSONError struct {
	Message string `json:"message"`
}

// Error 实现error接口
func (e *JSONError) Error() string {
	return e.Message
}

// Marshal 将任意值序列化为JSON
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal 将JSON反序列化到指定值
func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// MustMarshal 将任意值序列化为JSON，失败时返回nil
func MustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return data
}
