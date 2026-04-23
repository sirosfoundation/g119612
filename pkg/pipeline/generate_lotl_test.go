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
	assert.Equal(t, 1, lotl.ListAndSchemeInformation.LoTEVersionIdentifier)
	assert.Equal(t, "EU", lotl.ListAndSchemeInformation.SchemeTerritory)
	assert.Equal(t, etsi119602.LoTLTypeEU, lotl.ListAndSchemeInformation.LoTEType)
	assert.Equal(t, "European Commission", lotl.ListAndSchemeInformation.SchemeOperatorName.Get("en", ""))
	assert.NotEmpty(t, lotl.ListAndSchemeInformation.ListIssueDateTime)

	require.Len(t, lotl.ListAndSchemeInformation.PointersToOtherLoTE, 2)
	assert.Equal(t, "https://example.com/lote-pid.json", lotl.ListAndSchemeInformation.PointersToOtherLoTE[0].LoTELocation)
	assert.Equal(t, etsi119602.LoTETypePIDProviders, lotl.ListAndSchemeInformation.PointersToOtherLoTE[0].LoTEQualifiers[0].LoTEType)
	assert.Equal(t, "https://example.com/lote-wallet.json", lotl.ListAndSchemeInformation.PointersToOtherLoTE[1].LoTELocation)
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
	assert.Empty(t, lotls[0].ListAndSchemeInformation.PointersToOtherLoTE)
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
	assert.Equal(t, "SE", lotes[0].ListAndSchemeInformation.SchemeTerritory)
}
