package resilience

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestDoWithRetry_Success(t *testing.T) {
	calls := 0
	err := DoWithRetry(context.Background(), Config{MaxAttempts: 3, RetryBaseDelay: time.Millisecond}, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestDoWithRetry_TransientThenSuccess(t *testing.T) {
	calls := 0
	err := DoWithRetry(context.Background(), Config{MaxAttempts: 3, RetryBaseDelay: time.Millisecond}, func() error {
		calls++
		if calls < 3 {
			return &TransportError{Err: fmt.Errorf("connection refused")}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDoWithRetry_5xxRetry(t *testing.T) {
	calls := 0
	err := DoWithRetry(context.Background(), Config{MaxAttempts: 3, RetryBaseDelay: time.Millisecond}, func() error {
		calls++
		if calls < 3 {
			return &HTTPStatusError{StatusCode: http.StatusServiceUnavailable, URL: "https://example.com"}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success after 5xx retries, got: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDoWithRetry_4xxNoRetry(t *testing.T) {
	calls := 0
	err := DoWithRetry(context.Background(), Config{MaxAttempts: 3, RetryBaseDelay: time.Millisecond}, func() error {
		calls++
		return &HTTPStatusError{StatusCode: http.StatusNotFound, URL: "https://example.com"}
	})
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry on 404), got %d", calls)
	}
}

func TestDoWithRetry_AllAttemptsExhausted(t *testing.T) {
	calls := 0
	err := DoWithRetry(context.Background(), Config{MaxAttempts: 2, RetryBaseDelay: time.Millisecond}, func() error {
		calls++
		return &TransportError{Err: fmt.Errorf("timeout")}
	})
	if err == nil {
		t.Fatal("expected error when all attempts exhausted")
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestDoWithRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	calls := 0
	err := DoWithRetry(ctx, Config{MaxAttempts: 10, RetryBaseDelay: 50 * time.Millisecond}, func() error {
		calls++
		return &TransportError{Err: fmt.Errorf("timeout")}
	})
	if err == nil {
		t.Fatal("expected error on context cancellation")
	}
	// Should have been cut short by context
	if calls >= 10 {
		t.Errorf("expected fewer than 10 calls due to context cancellation, got %d", calls)
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"transport", &TransportError{Err: fmt.Errorf("dns")}, true},
		{"500", &HTTPStatusError{StatusCode: 500, URL: "x"}, true},
		{"503", &HTTPStatusError{StatusCode: 503, URL: "x"}, true},
		{"404", &HTTPStatusError{StatusCode: 404, URL: "x"}, false},
		{"400", &HTTPStatusError{StatusCode: 400, URL: "x"}, false},
		{"generic", fmt.Errorf("other"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{}
	c := cfg.withDefaults()
	if c.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts=3, got %d", c.MaxAttempts)
	}
	if c.RetryBaseDelay != 500*time.Millisecond {
		t.Errorf("expected RetryBaseDelay=500ms, got %v", c.RetryBaseDelay)
	}
	if c.MaxBodyBytes != 10<<20 {
		t.Errorf("expected MaxBodyBytes=10MB, got %d", c.MaxBodyBytes)
	}
}
