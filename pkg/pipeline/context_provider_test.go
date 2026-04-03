package pipeline

import (
	"crypto/x509"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
)

func TestGetCertPool(t *testing.T) {
	ctx := NewContext()
	assert.Nil(t, ctx.GetCertPool())

	ctx.InitCertPool()
	assert.NotNil(t, ctx.GetCertPool())
	assert.IsType(t, &x509.CertPool{}, ctx.GetCertPool())
}

func TestGetTSLs(t *testing.T) {
	ctx := NewContext()
	// GetTSLs on fresh context with empty stack
	tsls := ctx.GetTSLs()
	assert.Empty(t, tsls)

	// Add a TSL and verify
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TSLTagAttr: "test",
		},
	}
	ctx.TSLs.Push(tsl)
	tsls = ctx.GetTSLs()
	assert.Len(t, tsls, 1)
	assert.Equal(t, "test", tsls[0].StatusList.TSLTagAttr)
}

func TestGetTSLs_NilStack(t *testing.T) {
	ctx := &Context{}
	assert.Nil(t, ctx.GetTSLs())
}

func TestGetTSLCount(t *testing.T) {
	ctx := NewContext()
	assert.Equal(t, 0, ctx.GetTSLCount())

	ctx.TSLs.Push(&etsi119612.TSL{})
	assert.Equal(t, 1, ctx.GetTSLCount())

	ctx.TSLs.Push(&etsi119612.TSL{})
	assert.Equal(t, 2, ctx.GetTSLCount())
}

func TestGetTSLCount_NilStack(t *testing.T) {
	ctx := &Context{}
	assert.Equal(t, 0, ctx.GetTSLCount())
}

func TestGetLoTEs(t *testing.T) {
	ctx := NewContext()
	assert.Empty(t, ctx.GetLoTEs())

	lote := &etsi119602.ListOfTrustedEntities{
		Version: "1.0",
	}
	ctx.AddLoTE(lote)
	lotes := ctx.GetLoTEs()
	assert.Len(t, lotes, 1)
	assert.Equal(t, "1.0", lotes[0].Version)
}

func TestGetLoTEs_NilStack(t *testing.T) {
	ctx := &Context{}
	assert.Nil(t, ctx.GetLoTEs())
}

func TestGetLoTECount(t *testing.T) {
	ctx := NewContext()
	assert.Equal(t, 0, ctx.GetLoTECount())

	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{})
	assert.Equal(t, 1, ctx.GetLoTECount())
}

func TestGetLoTECount_NilStack(t *testing.T) {
	ctx := &Context{}
	assert.Equal(t, 0, ctx.GetLoTECount())
}

func TestEnsureLoTEs_Idempotent(t *testing.T) {
	ctx := &Context{}
	assert.Nil(t, ctx.LoTEs)

	ctx.EnsureLoTEs()
	assert.NotNil(t, ctx.LoTEs)

	stack := ctx.LoTEs
	ctx.EnsureLoTEs()
	assert.Same(t, stack, ctx.LoTEs) // Same pointer, not reallocated
}
