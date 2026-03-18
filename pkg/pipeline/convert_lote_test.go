package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertTSLToLoTE_Basic(t *testing.T) {
	ctx := NewContext()
	ctx.EnsureTSLStack()

	tsl := generateTSL("Test Service", "http://uri.etsi.org/TrstSvc/Svctype/CA/QC", []string{})
	ctx.TSLs.Push(tsl)

	ctx, err := ConvertTSLToLoTE(nil, ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTECount())
	lotes := ctx.GetLoTEs()
	require.Len(t, lotes, 1)

	// The converted LoTE should have the operator name from the TSL
	assert.NotEmpty(t, lotes[0].SchemeInformation.SchemeOperator)
}

func TestConvertTSLToLoTE_Empty(t *testing.T) {
	ctx := NewContext()
	_, err := ConvertTSLToLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no TSLs in context")
}

func TestConvertTSLToLoTE_Multiple(t *testing.T) {
	ctx := NewContext()
	ctx.EnsureTSLStack()

	tsl1 := generateTSL("Service A", "http://svctype/a", []string{})
	tsl2 := generateTSL("Service B", "http://svctype/b", []string{})
	ctx.TSLs.Push(tsl1)
	ctx.TSLs.Push(tsl2)

	ctx, err := ConvertTSLToLoTE(nil, ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, ctx.GetLoTECount())
}

func TestConvertTSLToLoTE_PreservesOriginalTSLs(t *testing.T) {
	ctx := NewContext()
	ctx.EnsureTSLStack()

	tsl := generateTSL("Test Service", "http://svctype/test", []string{})
	ctx.TSLs.Push(tsl)

	ctx, err := ConvertTSLToLoTE(nil, ctx)
	require.NoError(t, err)

	// Original TSLs should still be on the stack
	assert.Equal(t, 1, ctx.TSLs.Size())
	assert.Equal(t, 1, ctx.GetLoTECount())
}
