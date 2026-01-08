package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLoggerInterface(t *testing.T) {
	// Test that all implementations satisfy the interface
	var _ Logger = &LogrusAdapter{}
}

func TestLogrusAdapter(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a logrus logger with the buffer as output
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		DisableColors:    true,
	})

	// Create our adapter
	logger := NewLogrusAdapter(logrusLogger)

	// Test different log levels
	tests := []struct {
		level    LogLevel
		logFunc  func(string, ...Field)
		expected string
	}{
		{DebugLevel, logger.Debug, "level=debug"},
		{InfoLevel, logger.Info, "level=info"},
		{WarnLevel, logger.Warn, "level=warning"},
		{ErrorLevel, logger.Error, "level=error"},
	}

	for _, test := range tests {
		// Reset buffer
		buf.Reset()

		// Set the log level
		logger.SetLevel(test.level)

		// Log a message with fields
		test.logFunc("test message", F("key1", "value1"), F("key2", 123))

		// Check the output
		output := buf.String()
		if !strings.Contains(output, test.expected) {
			t.Errorf("Expected log to contain '%s', got: %s", test.expected, output)
		}
		if !strings.Contains(output, "test message") {
			t.Errorf("Expected log to contain 'test message', got: %s", output)
		}
		if !strings.Contains(output, "key1=value1") {
			t.Errorf("Expected log to contain 'key1=value1', got: %s", output)
		}
		if !strings.Contains(output, "key2=123") {
			t.Errorf("Expected log to contain 'key2=123', got: %s", output)
		}
	}
}

func TestJSONLogrusAdapter(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a logrus logger with the buffer as output
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetFormatter(&logrus.JSONFormatter{})

	// Create our adapter
	logger := NewLogrusAdapter(logrusLogger)

	// Log a message with fields
	logger.Info("test message", F("key1", "value1"), F("key2", 123))

	// Verify JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	// Check fields
	if msg, ok := logEntry["msg"].(string); !ok || msg != "test message" {
		t.Errorf("Expected 'msg' field to be 'test message', got: %v", logEntry["msg"])
	}
	if lvl, ok := logEntry["level"].(string); !ok || lvl != "info" {
		t.Errorf("Expected 'level' field to be 'info', got: %v", logEntry["level"])
	}
	if val, ok := logEntry["key1"].(string); !ok || val != "value1" {
		t.Errorf("Expected 'key1' field to be 'value1', got: %v", logEntry["key1"])
	}
	if val, ok := logEntry["key2"].(float64); !ok || val != 123 {
		t.Errorf("Expected 'key2' field to be 123, got: %v", logEntry["key2"])
	}
}

func TestWithContext(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create a logrus logger with the buffer as output
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetFormatter(&logrus.JSONFormatter{})

	// Create our adapter
	logger := NewLogrusAdapter(logrusLogger)

	// Create a context with a value
	type contextKey string
	var requestIDKey contextKey = "request_id"
	ctx := context.WithValue(context.Background(), requestIDKey, "req-123")

	// Create a logger with the context
	ctxLogger := logger.WithContext(ctx)

	// Log a message with the request ID manually (since logrus doesn't auto-extract from context)
	ctxLogger.Info("test with context", F("request_id", ctx.Value(requestIDKey)))

	// Verify the request ID is in the output
	output := buf.String()
	if !strings.Contains(output, "req-123") {
		t.Errorf("Expected log to contain request ID 'req-123', got: %s", output)
	}
}

func TestFactoryMethods(t *testing.T) {
	// Test DefaultLogger
	logger := DefaultLogger()
	if logger == nil {
		t.Fatal("DefaultLogger() returned nil")
	}

	// Test NewLogger with different levels
	for _, level := range []LogLevel{DebugLevel, InfoLevel, WarnLevel, ErrorLevel, FatalLevel} {
		logger := NewLogger(level)
		if logger == nil {
			t.Fatalf("NewLogger(%d) returned nil", level)
		}
		if got := logger.GetLevel(); got != level {
			t.Errorf("Expected level %d, got %d", level, got)
		}
	}

	// Test JSONLogger
	jsonLogger := JSONLogger(DebugLevel)
	if jsonLogger == nil {
		t.Fatal("JSONLogger() returned nil")
	}
	if got := jsonLogger.GetLevel(); got != DebugLevel {
		t.Errorf("Expected level %d, got %d", DebugLevel, got)
	}
}
