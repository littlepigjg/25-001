package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"logalert/internal/config"
	"logalert/internal/handler"
	"logalert/internal/store"
	"logalert/internal/service"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	logStore := store.NewMemoryLogStore(100000)
	cfg := config.DefaultConfig()
	log := logger.New()
	svc := service.NewLogService(logStore, cfg, log)
	h := handler.NewLogHandler(svc, log)

	// Send malformed JSON: object type into string field causes UnmarshalTypeError
	body := `{"level": {}, "source": "test-app", "message": "hello world"}`
	req := httptest.NewRequest("POST", "/api/logs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateLog(w, req)

	resp := w.Result()

	if resp.StatusCode == http.StatusBadRequest {
		t.Log("GREEN（绿灯，缺陷已修复）")
		return
	}

	if resp.StatusCode == http.StatusCreated {
		t.Log("RED（红灯，缺陷未修复）")
		t.FailNow()
	}

	t.Logf("UNEXPECTED STATUS: %d", resp.StatusCode)
	t.FailNow()
}
