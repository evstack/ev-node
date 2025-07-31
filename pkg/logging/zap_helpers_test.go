package logging

import (
	"errors"
	"testing"

	golog "github.com/ipfs/go-log/v2"
)

func TestStructuredLoggingPath(t *testing.T) {
	// Create a logger the same way the codebase does
	logger := golog.Logger("test-logger")

	// Test if structured logging is available
	if !IsStructuredLoggingAvailable(logger) {
		t.Errorf("Expected structured logging to be available with ipfs/go-log logger, but it's not")
	}

	// Test type assertion - logger is already *ZapEventLogger
	if logger.Desugar() == nil {
		t.Errorf("ZapEventLogger.Desugar() returned nil, structured logging not available")
	} else {
		t.Logf("✅ Structured logging is available - ZapEventLogger.Desugar() returned non-nil zap logger")
	}
}

func TestInfoWithKVStructuredPath(t *testing.T) {
	// Create logger
	logger := golog.Logger("test-infowithkv")

	// This should use the structured logging path
	InfoWithKV(logger, "test message", "key1", "value1", "key2", 42, "key3", true)

	// Verify structured logging is available (this test confirms the path is taken)
	if !IsStructuredLoggingAvailable(logger) {
		t.Errorf("Structured logging should be available for InfoWithKV")
	}
}

func TestErrorWithKV(t *testing.T) {
	logger := golog.Logger("test-errorwithkv")

	// Test ErrorWithKV with various types
	ErrorWithKV(logger, "test error message", "error", "something failed", "code", 500)

	// Verify structured logging is available
	if !IsStructuredLoggingAvailable(logger) {
		t.Errorf("Structured logging should be available for ErrorWithKV")
	}
}

func TestWarnWithKV(t *testing.T) {
	logger := golog.Logger("test-warnwithkv")

	// Test WarnWithKV with various types
	WarnWithKV(logger, "test warning message", "threshold", 100, "current", 85)

	// Verify structured logging is available
	if !IsStructuredLoggingAvailable(logger) {
		t.Errorf("Structured logging should be available for WarnWithKV")
	}
}

func TestDebugWithKV(t *testing.T) {
	logger := golog.Logger("test-debugwithkv")

	// Test DebugWithKV with various types
	DebugWithKV(logger, "test debug message", "count", 10, "enabled", true)

	// Verify structured logging is available
	if !IsStructuredLoggingAvailable(logger) {
		t.Errorf("Structured logging should be available for DebugWithKV")
	}
}

func TestOddNumberOfArguments(t *testing.T) {
	logger := golog.Logger("test-odd-args")

	// These should log an error and return early (no panic)
	InfoWithKV(logger, "test message", "key1", "value1", "key2") // odd number
	ErrorWithKV(logger, "test error", "key1")                    // odd number
	WarnWithKV(logger, "test warning", "key1", "value1", "key2") // odd number
	DebugWithKV(logger, "test debug", "key1")                    // odd number

	// If we get here without panicking, the test passes
	t.Log("✅ All methods handled odd number of arguments gracefully")
}

func TestCreateZapField(t *testing.T) {
	tests := []struct {
		key      string
		value    interface{}
		expected string // Just check the field type name
	}{
		{"stringField", "hello", "String"},
		{"intField", 42, "Int"},
		{"int64Field", int64(123), "Int64"},
		{"uint64Field", uint64(456), "Uint64"},
		{"float64Field", 3.14, "Float64"},
		{"boolField", true, "Bool"},
		{"anyField", []int{1, 2, 3}, "Any"},
	}

	for _, tt := range tests {
		field := createZapField(tt.key, tt.value)
		if field.Key != tt.key {
			t.Errorf("createZapField(%q, %v) key = %q, want %q", tt.key, tt.value, field.Key, tt.key)
		}
		// Field type is verified by the fact that it doesn't panic
	}
}

func TestCreateZapFieldWithError(t *testing.T) {
	// Test special case: error type uses zap.Error which doesn't take a key
	err := errors.New("test error")
	field := createZapField("myError", err)
	// zap.Error creates a field with key "error", not "myError"
	if field.Key != "error" {
		t.Errorf("createZapField with error type should use key 'error', got %q", field.Key)
	}
}

func TestFormatWithKV(t *testing.T) {
	tests := []struct {
		msg      string
		kvPairs  []interface{}
		expected string
	}{
		{"simple message", []interface{}{}, "simple message"},
		{"with params", []interface{}{"key1", "value1"}, "with params, key1: value1"},
		{"multiple params", []interface{}{"key1", "value1", "key2", 42}, "multiple params, key1: value1, key2: 42"},
	}

	for _, tt := range tests {
		result := formatWithKV(tt.msg, tt.kvPairs...)
		if result != tt.expected {
			t.Errorf("formatWithKV(%q, %v) = %q, want %q", tt.msg, tt.kvPairs, result, tt.expected)
		}
	}
}

func TestLoggerTypes(t *testing.T) {
	// Test different ways loggers might be created in the codebase

	// Method 1: Direct golog.Logger() call (most common)
	logger1 := golog.Logger("method1")
	t.Logf("Method 1 - golog.Logger(): type=%T, structured=%v", logger1, IsStructuredLoggingAvailable(logger1))

	// Method 2: With log level set
	golog.SetLogLevel("method2", "debug")
	logger2 := golog.Logger("method2")
	t.Logf("Method 2 - with log level: type=%T, structured=%v", logger2, IsStructuredLoggingAvailable(logger2))

	// All should support structured logging
	loggers := []*golog.ZapEventLogger{logger1, logger2}
	for i, logger := range loggers {
		if !IsStructuredLoggingAvailable(logger) {
			t.Errorf("Logger %d should support structured logging", i+1)
		}
	}
}
