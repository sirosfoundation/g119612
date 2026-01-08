package pipeline

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLog(t *testing.T) {
	ctx := NewContext()
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test with no arguments
	resultCtx, err := Log(pl, ctx)
	assert.NoError(t, err)
	assert.Equal(t, ctx, resultCtx)

	// Test with a message argument
	resultCtx, err = Log(pl, ctx, "Test log message")
	assert.NoError(t, err)
	assert.Equal(t, ctx, resultCtx)

	// The Log function should not modify the context
	assert.Equal(t, ctx, resultCtx)
}

func TestLog_EdgeCases(t *testing.T) {
	t.Run("Message with fields", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		result, err := Log(pl, ctx, "Test message", "key1=value1", "key2=value2")

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})

	t.Run("Debug level message", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.DebugLevel)}
		ctx := NewContext()

		result, err := Log(pl, ctx, "level=debug Debug message", "field=value")

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})

	t.Run("Info level message", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		result, err := Log(pl, ctx, "level=info Info message")

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})

	t.Run("Warn level message", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		result, err := Log(pl, ctx, "level=warn Warning message")

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})

	t.Run("Warning level message", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		result, err := Log(pl, ctx, "level=warning Warning message")

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})

	t.Run("Error level message", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		result, err := Log(pl, ctx, "level=error Error message")

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})

	t.Run("Level prefix with no message", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		result, err := Log(pl, ctx, "level=info")

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})

	t.Run("Unknown level defaults to info", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		result, err := Log(pl, ctx, "level=unknown Message with unknown level")

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})

	t.Run("Field without equals sign is ignored", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		result, err := Log(pl, ctx, "Message", "valid=value", "invalidfield")

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})

	t.Run("Empty field value", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		result, err := Log(pl, ctx, "Message", "key=")

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})

	t.Run("Case insensitive level", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.DebugLevel)}
		ctx := NewContext()

		// Test uppercase
		result, err := Log(pl, ctx, "level=DEBUG Debug message")
		require.NoError(t, err)
		assert.Equal(t, ctx, result)

		// Test mixed case
		result, err = Log(pl, ctx, "level=WaRn Warning message")
		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})
}
