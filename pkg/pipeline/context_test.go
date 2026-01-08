package pipeline

import (
	"crypto/x509"
	"testing"

	etsi119612 "github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/sirosfoundation/g119612/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithLogger(t *testing.T) {
	t.Run("Replace logger", func(t *testing.T) {
		// Create initial pipeline with default logger
		pl := &Pipeline{
			Pipes:  []Pipe{{MethodName: "test", MethodArguments: []string{}}},
			Logger: logging.NewLogger(logging.InfoLevel),
		}

		// Create new logger with different level
		debugLogger := logging.NewLogger(logging.DebugLevel)

		// Replace logger
		newPl := pl.WithLogger(debugLogger)

		// Verify new pipeline has new logger
		assert.NotNil(t, newPl)
		assert.Equal(t, debugLogger, newPl.Logger)

		// Verify pipes are preserved
		assert.Equal(t, pl.Pipes, newPl.Pipes)

		// Verify original pipeline unchanged
		assert.NotEqual(t, pl.Logger, newPl.Logger)
	})

	t.Run("Nil logger falls back to default", func(t *testing.T) {
		pl := &Pipeline{
			Pipes: []Pipe{{MethodName: "test", MethodArguments: []string{}}},
		}

		newPl := pl.WithLogger(nil)

		assert.NotNil(t, newPl)
		assert.NotNil(t, newPl.Logger)
	})

	t.Run("Preserves pipes", func(t *testing.T) {
		pipes := []Pipe{
			{MethodName: "load", MethodArguments: []string{"url"}},
			{MethodName: "transform", MethodArguments: []string{"xslt"}},
		}

		pl := &Pipeline{
			Pipes:  pipes,
			Logger: logging.NewLogger(logging.InfoLevel),
		}

		newLogger := logging.NewLogger(logging.DebugLevel)
		newPl := pl.WithLogger(newLogger)

		require.NotNil(t, newPl)
		assert.Equal(t, len(pipes), len(newPl.Pipes))
		assert.Equal(t, "load", newPl.Pipes[0].MethodName)
		assert.Equal(t, "transform", newPl.Pipes[1].MethodName)
	})
}

func TestAddTSL_EdgeCases(t *testing.T) {
	t.Run("Add nil TSL", func(t *testing.T) {
		ctx := NewContext()
		ctx.AddTSL(nil)

		// AddTSL returns early for nil, so nothing is added
		assert.Equal(t, 0, ctx.TSLs.Size())
		assert.Equal(t, 0, ctx.TSLTrees.Size())
	})

	t.Run("Add multiple TSLs", func(t *testing.T) {
		ctx := NewContext()

		// Add several TSLs
		// Note: AddTSL creates a tree and traverses it, which can add multiple TSLs to the stack
		for i := 0; i < 10; i++ {
			ctx.AddTSL(&etsi119612.TSL{})
		}

		// Each TSL gets added twice - once directly and once via tree traversal
		assert.Equal(t, 20, ctx.TSLs.Size())
		assert.Equal(t, 10, ctx.TSLTrees.Size())
	})

	t.Run("Add TSL with references", func(t *testing.T) {
		ctx := NewContext()

		// Create a TSL with referenced TSLs (children)
		childTSL1 := &etsi119612.TSL{Source: "child1.xml"}
		childTSL2 := &etsi119612.TSL{Source: "child2.xml"}
		rootTSL := &etsi119612.TSL{
			Source:     "root.xml",
			Referenced: []*etsi119612.TSL{childTSL1, childTSL2},
		}

		ctx.AddTSL(rootTSL)

		// Should have 1 tree
		assert.Equal(t, 1, ctx.TSLTrees.Size())

		// The legacy stack should have all TSLs (root + children)
		// AddTSLTree adds them via Traverse, plus AddTSL adds the root again
		// So we get: children from tree + root from tree + root directly = 4 total
		assert.Equal(t, 4, ctx.TSLs.Size())
	})

	t.Run("Method chaining works", func(t *testing.T) {
		ctx := NewContext()

		// AddTSL returns the context for chaining
		result := ctx.AddTSL(&etsi119612.TSL{Source: "test.xml"})

		assert.Equal(t, ctx, result)
		assert.Equal(t, 1, ctx.TSLTrees.Size())
	})

	t.Run("TSLs stack is always initialized", func(t *testing.T) {
		ctx := NewContext()

		// NewContext already initializes the stack
		assert.NotNil(t, ctx.TSLs)

		// Add a TSL
		ctx.AddTSL(&etsi119612.TSL{Source: "first.xml"})

		// Stack should still be valid and have entries
		assert.NotNil(t, ctx.TSLs)
		assert.Equal(t, 2, ctx.TSLs.Size()) // Added twice (via tree and directly)
	})

	t.Run("Handles context with nil TSLs stack", func(t *testing.T) {
		// Create a context manually (not via NewContext)
		ctx := &Context{
			TSLTrees: utils.NewStack[*TSLTree](),
			TSLs:     nil, // Explicitly set to nil
			Data:     make(map[string]any),
		}

		// AddTSLTree will initialize it
		ctx.AddTSL(&etsi119612.TSL{Source: "test.xml"})

		// Now it should be initialized
		assert.NotNil(t, ctx.TSLs)
		assert.Greater(t, ctx.TSLs.Size(), 0)
	})
}

func TestContext_Copy_DeepCopy(t *testing.T) {
	t.Run("Modifications don't affect original", func(t *testing.T) {
		original := NewContext()
		original.Data["key1"] = "value1"
		original.AddTSL(&etsi119612.TSL{})

		// Original has 1 data entry and 2 TSLs (AddTSL adds via both tree and direct push)
		assert.Equal(t, 1, len(original.Data))
		originalTSLCount := original.TSLs.Size()

		// Make a copy
		copied := original.Copy()

		// Modify the copy
		copied.Data["key2"] = "value2"
		copied.AddTSL(&etsi119612.TSL{})

		// Verify original unchanged
		assert.Equal(t, 1, len(original.Data))
		assert.Equal(t, originalTSLCount, original.TSLs.Size())

		// Verify copy has modifications
		assert.Equal(t, 2, len(copied.Data))
		assert.Greater(t, copied.TSLs.Size(), originalTSLCount)
	})

	t.Run("Copy with CertPool", func(t *testing.T) {
		original := NewContext()
		original.CertPool = x509.NewCertPool()

		copied := original.Copy()

		assert.NotNil(t, copied.CertPool)
		// CertPool is recreated (not the same reference)
		assert.NotSame(t, original.CertPool, copied.CertPool)
	})
}
