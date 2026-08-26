// Package model 定义系统核心数据模型
package model

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
)

// GenerateID 生成唯一标识符
func GenerateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// fallback to timestamp-based ID
		return time.Now().Format("20060102150405.000000")
	}
	return hex.EncodeToString(b)
}

// ExtractKeywords 从消息中提取关键词
func ExtractKeywords(message string) []string {
	if message == "" {
		return nil
	}
	// 简单的关键词提取：基于空格和标点分割
	words := splitWords(message)
	keywords := make([]string, 0, len(words))
	seen := make(map[string]struct{})
	for _, w := range words {
		w = normalizeKeyword(w)
		if w == "" {
			continue
		}
		if len(w) < 2 {
			continue
		}
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		keywords = append(keywords, w)
	}
	return keywords
}

// splitWords 将消息分割为单词
func splitWords(message string) []string {
	// 使用非字母数字字符作为分隔符
	replacer := strings.NewReplacer(
		"，", " ",
		"。", " ",
		"！", " ",
		"？", " ",
		"；", " ",
		"：", " ",
		"、", " ",
		"（", " ",
		"）", " ",
		"【", " ",
		"】", " ",
		",", " ",
		".", " ",
		"!", " ",
		"?", " ",
		";", " ",
		":", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
		"{", " ",
		"}", " ",
		"\t", " ",
		"\n", " ",
		"\r", " ",
		"|", " ",
		"/", " ",
		"\\", " ",
		"=", " ",
		"+", " ",
		"-", " ",
		"*", " ",
		"&", " ",
		"^", " ",
		"%", " ",
		"$", " ",
		"#", " ",
		"@", " ",
		"~", " ",
		"`", " ",
		"<", " ",
		">", " ",
	)
	normalized := replacer.Replace(message)
	fields := strings.Fields(normalized)
	return fields
}

// normalizeKeyword 规范化关键词（转小写、去空格）
func normalizeKeyword(kw string) string {
	return strings.ToLower(strings.TrimSpace(kw))
}

// ContainsAllKeywords 检查消息是否包含所有指定关键词
func ContainsAllKeywords(message string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	msg := strings.ToLower(message)
	for _, kw := range keywords {
		kw = normalizeKeyword(kw)
		if !strings.Contains(msg, kw) {
			return false
		}
	}
	return true
}

// ContainsAnyKeyword 检查消息是否包含任意指定关键词
func ContainsAnyKeyword(message string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	msg := strings.ToLower(message)
	for _, kw := range keywords {
		kw = normalizeKeyword(kw)
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}
