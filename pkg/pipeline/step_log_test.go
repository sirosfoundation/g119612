package pipeline

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEcho(t *testing.T) {
	t.Run("Returns context unchanged", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()
		ctx.Data["test"] = "value"

		result, err := Echo(pl, ctx)

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
		assert.Equal(t, "value", result.Data["test"])
	})

	t.Run("Works with arguments", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		result, err := Echo(pl, ctx, "arg1", "arg2", "arg3")

		require.NoError(t, err)
		assert.Equal(t, ctx, result)
	})

	t.Run("Works with no arguments", func(t *testing.T) {
		pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
		ctx := NewContext()

		result, err := Echo(pl, ctx)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}
