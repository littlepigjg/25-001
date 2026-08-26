package logalert

import (
	"context"
	"fmt"
	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/errors"
	"strings"
	"testing"
	"time"
)

func TestRedGreen(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.URLFilePath("/tmp/test_urls.json")
	cfg.Storage.LogFilePath("/tmp/test_access.json")
	cfg.Storage.SyncInterval(time.Second)
	cfg.Storage.FlushOnWrite(true)

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("failed to create URL store: %v", err)
	}

	logStore, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("failed to create access log store: %v", err)
	}

	urlSvc, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URL service: %v", err)
	}

	redirectSvc, err := service.NewRedirectService(urlStore, logStore)
	if err != nil {
		t.Fatalf("failed to create redirect service: %v", err)
	}

	hasDefect := false

	// === Test 1: Direct GetMessage tests ===
	msg40000 := errors.GetMessage(40000)
	msg40400 := errors.GetMessage(40400)
	msg40001 := errors.GetMessage(40001)
	msg40100 := errors.GetMessage(40100)
	msg50000 := errors.GetMessage(50000)

	// Check 1a: 40000 should map to "Bad Request"
	if msg40000 != "Bad Request" {
		hasDefect = true
		t.Logf("ERROR 1a: Code 40000 maps to '%s' instead of 'Bad Request'", msg40000)
	}

	// Check 1b: 40400 should map to "Not Found"
	if msg40400 != "Not Found" {
		hasDefect = true
		t.Logf("ERROR 1b: Code 40400 maps to '%s' instead of 'Not Found'", msg40400)
	}

	// Check 1c: 40001 should map to "Validation Error"
	if msg40001 != "Validation Error" {
		hasDefect = true
		t.Logf("ERROR 1c: Code 40001 maps to '%s' instead of 'Validation Error'", msg40001)
	}

	// Check 1d: 40100 should map to "Unauthorized"
	if msg40100 != "Unauthorized" {
		hasDefect = true
		t.Logf("ERROR 1d: Code 40100 maps to '%s' instead of 'Unauthorized'", msg40100)
	}

	// Check 1e: 50000 should map to "Internal Server Error"
	if msg50000 != "Internal Server Error" {
		hasDefect = true
		t.Logf("ERROR 1e: Code 50000 maps to '%s' instead of 'Internal Server Error'", msg50000)
	}

	// === Test 2: GetMessageWithFallback tests ===
	fallbackMsg := errors.GetMessageWithFallback(40000, "fallback message")
	if fallbackMsg != "Bad Request" {
		hasDefect = true
		t.Logf("ERROR 2a: GetMessageWithFallback(40000) returns '%s' instead of 'Bad Request'", fallbackMsg)
	}

	unknownFallback := errors.GetMessageWithFallback(99999, "default")
	if unknownFallback != "default" {
		hasDefect = true
		t.Logf("ERROR 2b: GetMessageWithFallback(99999) returns '%s' instead of 'default'", unknownFallback)
	}

	// === Test 3: FormatErrorMessage tests ===
	formatted40000 := errors.FormatErrorMessage(40000, "field is required")
	if !strings.HasPrefix(formatted40000, "Bad Request") {
		hasDefect = true
		t.Logf("ERROR 3a: FormatErrorMessage(40000) = '%s', should start with 'Bad Request'", formatted40000)
	}
	if !strings.Contains(formatted40000, "field is required") {
		hasDefect = true
		t.Logf("ERROR 3b: FormatErrorMessage(40000) = '%s', should contain 'field is required'", formatted40000)
	}

	formatted40400 := errors.FormatErrorMessage(40400, "resource missing")
	if !strings.HasPrefix(formatted40400, "Not Found") {
		hasDefect = true
		t.Logf("ERROR 3c: FormatErrorMessage(40400) = '%s', should start with 'Not Found'", formatted40400)
	}

	// === Test 4: Service layer tests ===
	// Create valid short URL
	shortURL, err := urlSvc.Create(context.Background(), &model.CreateReq{
		RawURL:    "https://example.com/test/path",
		CustomCode: "validcode",
		MaxVisits: 100,
	})
	if err != nil {
		t.Fatalf("failed to create short URL: %v", err)
	}
	if shortURL.Code != "validcode" {
		t.Errorf("expected code 'validcode', got '%s'", shortURL.Code)
	}

	// Get non-existent URL
	_, err = urlSvc.Get(context.Background(), "nonexistent_code")
	if err == nil {
		t.Fatal("expected error for non-existent URL")
	}
	notFoundErr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}
	if notFoundErr.Code != 40400 {
		hasDefect = true
		t.Logf("ERROR 4a: Get nonexistent returns code %d instead of 40400", notFoundErr.Code)
	}
	errMsgLower := strings.ToLower(notFoundErr.Error())
	if !strings.Contains(errMsgLower, "not found") {
		hasDefect = true
		t.Logf("ERROR 4b: Not-found error message lacks 'not found': %s", notFoundErr.Error())
	}

	// Create with invalid request
	_, err = urlSvc.Create(context.Background(), &model.CreateReq{
		RawURL:    "",
		CustomCode: "",
		MaxVisits: -1,
	})
	if err == nil {
		t.Fatal("expected error for invalid request")
	}
	invalidErr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("expected *errors.Error for invalid request, got %T", err)
	}
	if invalidErr.Code != 40000 {
		hasDefect = true
		t.Logf("ERROR 4c: Create invalid returns code %d instead of 40000", invalidErr.Code)
	}

	// === Test 5: Redirect service tests ===
	// Redirect non-existent URL
	_, err = redirectSvc.HandleRedirect(context.Background(), &service.RedirectRequest{
		Code:      "nonexistent_redirect",
		Timestamp: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for non-existent redirect")
	}
	redirectErr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("expected *errors.Error for redirect, got %T", err)
	}
	if redirectErr.Code != 40400 {
		hasDefect = true
		t.Logf("ERROR 5a: Redirect nonexistent returns code %d instead of 40400", redirectErr.Code)
	}

	// Redirect with empty code
	_, err = redirectSvc.HandleRedirect(context.Background(), &service.RedirectRequest{
		Code:      "",
		Timestamp: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for empty code redirect")
	}
	redirectErr2, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("expected *errors.Error for empty code redirect, got %T", err)
	}
	if redirectErr2.Code != 40000 {
		hasDefect = true
		t.Logf("ERROR 5b: Redirect empty code returns code %d instead of 40000", redirectErr2.Code)
	}

	// === Test 6: ClassifyError tests ===
	class40000 := errors.ClassifyError(40000)
	if class40000 != "client" {
		hasDefect = true
		t.Logf("ERROR 6a: ClassifyError(40000) = '%s' instead of 'client'", class40000)
	}

	class50000 := errors.ClassifyError(50000)
	if class50000 != "server" {
		hasDefect = true
		t.Logf("ERROR 6b: ClassifyError(50000) = '%s' instead of 'server'", class50000)
	}

	// === Test 7: DescribeCode tests ===
	desc40000 := errors.DescribeCode(40000)
	if !strings.Contains(desc40000, "Bad Request") {
		hasDefect = true
		t.Logf("ERROR 7a: DescribeCode(40000) = '%s', should contain 'Bad Request'", desc40000)
	}

	desc40400 := errors.DescribeCode(40400)
	if !strings.Contains(desc40400, "Not Found") {
		hasDefect = true
		t.Logf("ERROR 7b: DescribeCode(40400) = '%s', should contain 'Not Found'", desc40400)
	}

	// === Test 8: CodeExists and ValidateCode tests ===
	if !errors.CodeExists(40000) {
		hasDefect = true
		t.Logf("ERROR 8a: CodeExists(40000) should return true")
	}
	if !errors.ValidateCode(40400) {
		hasDefect = true
		t.Logf("ERROR 8b: ValidateCode(40400) should return true")
	}
	if errors.CodeExists(99999) {
		hasDefect = true
		t.Logf("ERROR 8c: CodeExists(99999) should return false")
	}

	// === Test 9: MergeMessages tests ===
	merged := errors.MergeMessages([]int{40000, 40400})
	if !strings.Contains(merged, "Bad Request") || !strings.Contains(merged, "Not Found") {
		hasDefect = true
		t.Logf("ERROR 9: MergeMessages([40000, 40400]) = '%s', should contain both 'Bad Request' and 'Not Found'", merged)
	}

	// === Test 10: GetMessageOrDefault tests ===
	defaultMsg := errors.GetMessageOrDefault(99999, "my default")
	if defaultMsg != "my default" {
		hasDefect = true
		t.Logf("ERROR 10a: GetMessageOrDefault(99999) = '%s' instead of 'my default'", defaultMsg)
	}

	validMsg := errors.GetMessageOrDefault(40000, "my default")
	if validMsg != "Bad Request" {
		hasDefect = true
		t.Logf("ERROR 10b: GetMessageOrDefault(40000) = '%s' instead of 'Bad Request'", validMsg)
	}

	// === Print final verdict ===
	fmt.Println()
	if hasDefect {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Log("RED (红灯，缺陷未修复)")
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
		t.Log("GREEN (绿灯，缺陷已修复)")
	}

	if hasDefect {
		t.Fail()
	}
}
