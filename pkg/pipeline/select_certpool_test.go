package pipeline

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
)

func TestSelectCertPoolWithReferences(t *testing.T) {
	pl := createTestPipeline(nil)
	ctx := NewContext()

	// Create test TSLs using the generateTSL helper function
	mainTSL := generateTSL("Main Service", "http://example.org/MainService", []string{})
	mainTSL.Source = "Main TSL"

	// Create referenced TSL
	refTSL := generateTSL("Referenced Service", "http://example.org/ReferencedService", []string{})
	refTSL.Source = "Referenced TSL"

	// Add referenced TSL to the main TSL
	mainTSL.Referenced = []*etsi119612.TSL{refTSL}

	// Set up the TSL stack in the context
	ctx.EnsureTSLStack()
	ctx.TSLs.Push(mainTSL)

	// Test without including references
	ctx, err := SelectCertPool(pl, ctx)
	assert.NoError(t, err)
	assert.NotNil(t, ctx.CertPool)

	// Test with including references using reference-depth
	ctx = NewContext()
	ctx.EnsureTSLStack()
	ctx.TSLs.Push(mainTSL)
	ctx, err = SelectCertPool(pl, ctx, "reference-depth:1")
	assert.NoError(t, err)
	assert.NotNil(t, ctx.CertPool)

	// Test with legacy include-referenced parameter
	ctx = NewContext()
	ctx.EnsureTSLStack()
	ctx.TSLs.Push(mainTSL)
	ctx, err = SelectCertPool(pl, ctx, "include-referenced")
	assert.NoError(t, err)
	assert.NotNil(t, ctx.CertPool)
}
