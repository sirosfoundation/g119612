package pipeline

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSentinelErrors(t *testing.T) {
	// Verify sentinel errors are not nil
	assert.NotNil(t, ErrNoTSLs)
	assert.NotNil(t, ErrInvalidArguments)
	assert.NotNil(t, ErrEmptyPipeline)
	assert.NotNil(t, ErrFunctionNotFound)

	// Verify error messages
	assert.Contains(t, ErrNoTSLs.Error(), "no TSLs")
	assert.Contains(t, ErrInvalidArguments.Error(), "invalid")
	assert.Contains(t, ErrEmptyPipeline.Error(), "no steps")
	assert.Contains(t, ErrFunctionNotFound.Error(), "not found")
}

func TestTSLLoadError(t *testing.T) {
	t.Run("Basic error", func(t *testing.T) {
		baseErr := errors.New("connection timeout")
		err := NewTSLLoadError("https://example.com/tsl.xml", baseErr)

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "https://example.com/tsl.xml")
		assert.Contains(t, err.Error(), "connection timeout")
		assert.ErrorIs(t, err, baseErr)
	})

	t.Run("Error with reason", func(t *testing.T) {
		baseErr := errors.New("404 not found")
		err := NewTSLLoadErrorWithReason("https://example.com/tsl.xml", "HTTP error", baseErr)

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "https://example.com/tsl.xml")
		assert.Contains(t, err.Error(), "HTTP error")
		assert.Contains(t, err.Error(), "404 not found")
		assert.ErrorIs(t, err, baseErr)
	})

	t.Run("Unwrap", func(t *testing.T) {
		baseErr := errors.New("base error")
		err := NewTSLLoadError("https://example.com/tsl.xml", baseErr)

		unwrapped := errors.Unwrap(err)
		assert.Equal(t, baseErr, unwrapped)
	})

	t.Run("Error chain", func(t *testing.T) {
		baseErr := errors.New("network error")
		err := NewTSLLoadError("https://example.com/tsl.xml", baseErr)

		// Should be able to use errors.Is
		assert.True(t, errors.Is(err, baseErr))
	})
}

func TestXSLTTransformError(t *testing.T) {
	t.Run("Basic error", func(t *testing.T) {
		baseErr := errors.New("invalid XML")
		err := NewXSLTTransformError("tsl-to-html.xslt", 5, baseErr)

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "tsl-to-html.xslt")
		assert.Contains(t, err.Error(), "TSL 5")
		assert.Contains(t, err.Error(), "invalid XML")
		assert.ErrorIs(t, err, baseErr)
	})

	t.Run("Unwrap", func(t *testing.T) {
		baseErr := errors.New("xsltproc error")
		err := NewXSLTTransformError("style.xslt", 0, baseErr)

		unwrapped := errors.Unwrap(err)
		assert.Equal(t, baseErr, unwrapped)
	})

	t.Run("Error formatting", func(t *testing.T) {
		baseErr := errors.New("transformation failed")
		err := NewXSLTTransformError("/path/to/stylesheet.xslt", 10, baseErr)

		errMsg := err.Error()
		assert.Contains(t, errMsg, "XSLT transformation failed")
		assert.Contains(t, errMsg, "TSL 10")
		assert.Contains(t, errMsg, "/path/to/stylesheet.xslt")
	})
}

func TestValidationError(t *testing.T) {
	t.Run("Complete validation error", func(t *testing.T) {
		err := NewValidationError("timeout", "invalid", "must be a valid duration")

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "timeout")
		assert.Contains(t, err.Error(), "invalid")
		assert.Contains(t, err.Error(), "must be a valid duration")
	})

	t.Run("Field only", func(t *testing.T) {
		err := NewValidationError("url", "", "URL is required")

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "url")
		assert.Contains(t, err.Error(), "URL is required")
		assert.NotContains(t, err.Error(), "''") // Should not show empty value
	})

	t.Run("Message only", func(t *testing.T) {
		err := NewValidationError("", "", "general validation failed")

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "validation error")
		assert.Contains(t, err.Error(), "general validation failed")
	})
}

func TestPublishError(t *testing.T) {
	t.Run("Basic error", func(t *testing.T) {
		baseErr := errors.New("permission denied")
		err := NewPublishError("/output/path", 23, baseErr)

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "/output/path")
		assert.Contains(t, err.Error(), "23")
		assert.Contains(t, err.Error(), "permission denied")
		assert.ErrorIs(t, err, baseErr)
	})

	t.Run("Unwrap", func(t *testing.T) {
		baseErr := errors.New("disk full")
		err := NewPublishError("/mnt/output", 5, baseErr)

		unwrapped := errors.Unwrap(err)
		assert.Equal(t, baseErr, unwrapped)
	})

	t.Run("Multiple TSLs", func(t *testing.T) {
		baseErr := errors.New("write error")
		err := NewPublishError("/output", 100, baseErr)

		assert.Contains(t, err.Error(), "100 TSL(s)")
	})
}

func TestCertificateError(t *testing.T) {
	t.Run("With subject", func(t *testing.T) {
		baseErr := errors.New("expired")
		err := NewCertificateError("validation", "CN=Example CA", baseErr)

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "validation")
		assert.Contains(t, err.Error(), "CN=Example CA")
		assert.Contains(t, err.Error(), "expired")
		assert.ErrorIs(t, err, baseErr)
	})

	t.Run("Without subject", func(t *testing.T) {
		baseErr := errors.New("parse error")
		err := NewCertificateError("parse", "", baseErr)

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "parse")
		assert.Contains(t, err.Error(), "parse error")
		assert.NotContains(t, err.Error(), "for ") // Should not have "for" without subject
	})

	t.Run("Unwrap", func(t *testing.T) {
		baseErr := errors.New("invalid signature")
		err := NewCertificateError("verify", "CN=Test", baseErr)

		unwrapped := errors.Unwrap(err)
		assert.Equal(t, baseErr, unwrapped)
	})
}

func TestPipelineStepError(t *testing.T) {
	t.Run("Basic error", func(t *testing.T) {
		baseErr := errors.New("missing argument")
		err := NewPipelineStepError("load", 2, []string{"arg1", "arg2"}, baseErr)

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "step 2")
		assert.Contains(t, err.Error(), "load")
		assert.Contains(t, err.Error(), "missing argument")
		assert.ErrorIs(t, err, baseErr)
	})

	t.Run("No arguments", func(t *testing.T) {
		baseErr := errors.New("execution failed")
		err := NewPipelineStepError("transform", 0, []string{}, baseErr)

		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "step 0")
		assert.Contains(t, err.Error(), "transform")
	})

	t.Run("Unwrap", func(t *testing.T) {
		baseErr := errors.New("step failed")
		err := NewPipelineStepError("publish", 5, nil, baseErr)

		unwrapped := errors.Unwrap(err)
		assert.Equal(t, baseErr, unwrapped)
	})

	t.Run("Error chain with errors.Is", func(t *testing.T) {
		baseErr := ErrInvalidArguments
		err := NewPipelineStepError("select", 3, []string{"invalid"}, baseErr)

		assert.True(t, errors.Is(err, ErrInvalidArguments))
	})
}

// Test error wrapping and unwrapping chains
func TestErrorChaining(t *testing.T) {
	t.Run("Nested error unwrapping", func(t *testing.T) {
		// Create a chain: base -> TSLLoadError -> PipelineStepError
		baseErr := errors.New("network timeout")
		loadErr := NewTSLLoadError("https://example.com/tsl.xml", baseErr)
		stepErr := NewPipelineStepError("load", 0, []string{"https://example.com/tsl.xml"}, loadErr)

		// Should be able to unwrap to base error
		assert.True(t, errors.Is(stepErr, baseErr))
		assert.True(t, errors.Is(stepErr, loadErr))
	})

	t.Run("errors.As for type assertion", func(t *testing.T) {
		baseErr := errors.New("transform failed")
		xsltErr := NewXSLTTransformError("style.xslt", 1, baseErr)
		stepErr := NewPipelineStepError("transform", 3, nil, xsltErr)

		// Should be able to extract XSLTTransformError from the chain
		var extractedErr *XSLTTransformError
		require.True(t, errors.As(stepErr, &extractedErr))
		assert.Equal(t, "style.xslt", extractedErr.StylesheetPath)
		assert.Equal(t, 1, extractedErr.TSLIndex)
	})
}

// Benchmark error creation
func BenchmarkNewTSLLoadError(b *testing.B) {
	baseErr := errors.New("test error")
	for i := 0; i < b.N; i++ {
		_ = NewTSLLoadError("https://example.com/tsl.xml", baseErr)
	}
}

func BenchmarkNewXSLTTransformError(b *testing.B) {
	baseErr := errors.New("test error")
	for i := 0; i < b.N; i++ {
		_ = NewXSLTTransformError("stylesheet.xslt", 0, baseErr)
	}
}

func BenchmarkNewPipelineStepError(b *testing.B) {
	baseErr := errors.New("test error")
	args := []string{"arg1", "arg2"}
	for i := 0; i < b.N; i++ {
		_ = NewPipelineStepError("load", 0, args, baseErr)
	}
}

// Example usage for documentation
func ExampleNewTSLLoadError() {
	err := NewTSLLoadError("https://example.com/tsl.xml", errors.New("connection timeout"))
	fmt.Println(err)
	// Output will contain: failed to load TSL from https://example.com/tsl.xml: connection timeout
}

func ExampleNewXSLTTransformError() {
	err := NewXSLTTransformError("tsl-to-html.xslt", 5, errors.New("invalid XML"))
	fmt.Println(err)
	// Output will contain: XSLT transformation failed for TSL 5 using stylesheet tsl-to-html.xslt: invalid XML
}

func ExampleNewPipelineStepError() {
	baseErr := errors.New("missing required argument")
	err := NewPipelineStepError("load", 2, []string{"arg1"}, baseErr)
	fmt.Println(err)
	// Output will contain: step 2 (load) failed: missing required argument
}
