package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateLoTL_Basic(t *testing.T) {
	dir := t.TempDir()

	lotlYAML := `operatorNames:
  - language: en
    value: "European Commission"
schemeType: "http://uri.etsi.org/19602/LoTLType/EUListOfTrustedLists"
territory: EU
sequenceNumber: 1
pointers:
  - location: "https://example.com/lote-pid.json"
    schemeTerritory: EU
    schemeType: "http://uri.etsi.org/19602/LoTEType/EUPIDProvidersList"
  - location: "https://example.com/lote-wallet.json"
    schemeTerritory: EU
    schemeType: "http://uri.etsi.org/19602/LoTEType/EUWalletProvidersList"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "lotl.yaml"), []byte(lotlYAML), 0644))

	ctx := NewContext()
	ctx, err := GenerateLoTL(nil, ctx, dir)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTLCount())
	lotls := ctx.GetLoTLs()
	require.Len(t, lotls, 1)

	lotl := lotls[0]
	assert.Equal(t, etsi119602.LoTEVersion, lotl.Version)
	assert.Equal(t, "EU", lotl.SchemeInformation.Territory)
	assert.Equal(t, etsi119602.LoTLTypeEU, lotl.SchemeInformation.SchemeType)
	assert.Equal(t, "European Commission", lotl.SchemeInformation.SchemeOperator.Get("en", ""))
	assert.False(t, lotl.SchemeInformation.IssueDate.IsZero())

	require.Len(t, lotl.PointersToOtherLoTEs, 2)
	assert.Equal(t, "https://example.com/lote-pid.json", lotl.PointersToOtherLoTEs[0].Location)
	assert.Equal(t, etsi119602.LoTETypePIDProviders, lotl.PointersToOtherLoTEs[0].SchemeType)
	assert.Equal(t, "https://example.com/lote-wallet.json", lotl.PointersToOtherLoTEs[1].Location)
}

func TestGenerateLoTL_MissingArgs(t *testing.T) {
	ctx := NewContext()
	_, err := GenerateLoTL(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 1 argument")
}

func TestGenerateLoTL_MissingYAML(t *testing.T) {
	dir := t.TempDir()
	ctx := NewContext()
	_, err := GenerateLoTL(nil, ctx, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lotl.yaml")
}

func TestGenerateLoTL_MissingOperatorNames(t *testing.T) {
	dir := t.TempDir()

	lotlYAML := `schemeType: "http://example.com/type"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "lotl.yaml"), []byte(lotlYAML), 0644))

	ctx := NewContext()
	_, err := GenerateLoTL(nil, ctx, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operatorName")
}

func TestGenerateLoTL_MissingSchemeType(t *testing.T) {
	dir := t.TempDir()

	lotlYAML := `operatorNames:
  - language: en
    value: "Test"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "lotl.yaml"), []byte(lotlYAML), 0644))

	ctx := NewContext()
	_, err := GenerateLoTL(nil, ctx, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schemeType")
}

func TestGenerateLoTL_NoPointers(t *testing.T) {
	dir := t.TempDir()

	lotlYAML := `operatorNames:
  - language: en
    value: "Test Operator"
schemeType: "http://example.com/type"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "lotl.yaml"), []byte(lotlYAML), 0644))

	ctx := NewContext()
	ctx, err := GenerateLoTL(nil, ctx, dir)
	require.NoError(t, err)

	lotls := ctx.GetLoTLs()
	require.Len(t, lotls, 1)
	assert.Empty(t, lotls[0].PointersToOtherLoTEs)
}

func TestGenerateLoTL_AutoDetectsSchemeYAML(t *testing.T) {
	// Using generate-lotl on a directory with scheme.yaml should produce a LoTE
	dir := t.TempDir()

	schemeYAML := `operatorNames:
  - language: en
    value: "Test Operator"
schemeType: "http://example.com/lote-type"
territory: SE
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte(schemeYAML), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "entities"), 0750))

	ctx := NewContext()
	ctx, err := GenerateLoTL(nil, ctx, dir)
	require.NoError(t, err)

	// Should have produced a LoTE, not a LoTL
	assert.Equal(t, 0, ctx.GetLoTLCount())
	lotes := ctx.LoTEs.ToSlice()
	require.Len(t, lotes, 1)
	assert.Equal(t, "SE", lotes[0].SchemeInformation.Territory)
}
