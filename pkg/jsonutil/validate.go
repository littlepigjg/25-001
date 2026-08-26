// Package jsonutil 提供JSON读写工具
package jsonutil

import (
	"encoding/json"
	"fmt"
)

// Validate 验证JSON数据
func Validate(data []byte) error {
	if len(data) == 0 {
		return &JSONError{Message: "empty JSON data"}
	}

	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return &JSONError{Message: "invalid JSON: " + err.Error()}
	}
	return nil
}

// ValidateAndParse 验证并解析JSON
func ValidateAndParse(data []byte, v interface{}) error {
	if err := Validate(data); err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// Pretty 格式化JSON
func Pretty(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// MustPretty 格式化JSON（失败返回空字符串）
func MustPretty(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("{\n  \"error\": \"%v\"\n}", err)
	}
	return string(data)
}

// Merge 合并两个JSON对象
func Merge(a, b []byte) ([]byte, error) {
	var objA, objB map[string]interface{}
	if err := json.Unmarshal(a, &objA); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &objB); err != nil {
		return nil, err
	}

	for k, v := range objB {
		objA[k] = v
	}
	return json.Marshal(objA)
}

// Keys 获取JSON对象的所有键
func Keys(data []byte) ([]string, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	return keys, nil
}

// HasKey 检查JSON是否包含指定键
func HasKey(data []byte, key string) (bool, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return false, err
	}
	_, exists := obj[key]
	return exists, nil
}
