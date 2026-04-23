package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validLoTL() *etsi119602.ListOfTrustedLists {
	return &etsi119602.ListOfTrustedLists{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeTerritory:       "EU",
			SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "Test Operator"}},
			LoTEType:              etsi119602.LoTLTypeEU,
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2027-01-01T00:00:00Z",
			PointersToOtherLoTE: []etsi119602.OtherLoTEPointer{
				{
					LoTELocation: "https://example.com/lote.json",
					LoTEQualifiers: []etsi119602.LoTEQualifier{
						{SchemeTerritory: "EU", LoTEType: etsi119602.LoTETypePIDProviders},
					},
				},
			},
		},
	}
}

func TestPublishLoTL_Unsigned(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	ctx.AddLoTL(validLoTL())

	ctx, err := PublishLoTL(nil, ctx, outputDir)
	require.NoError(t, err)

	// Default: JSON only
	jsonData, err := os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists-EU.json"))
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"SchemeTerritory": "EU"`)
	_, err = os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists-EU.xml"))
	assert.True(t, os.IsNotExist(err), "XML should not be produced by default")
}

func TestPublishLoTL_WithXML(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	ctx.AddLoTL(validLoTL())

	ctx, err := PublishLoTL(nil, ctx, outputDir, "xml")
	require.NoError(t, err)

	// Both JSON and XML should exist
	jsonData, err := os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists-EU.json"))
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"SchemeTerritory": "EU"`)

	xmlData, err := os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists-EU.xml"))
	require.NoError(t, err)
	assert.Contains(t, string(xmlData), "ListAndSchemeInformation")
}

func TestPublishLoTL_JSONOnlyFlag(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	ctx.AddLoTL(validLoTL())

	// json-only is effectively the default (no XML), but should still work
	ctx, err := PublishLoTL(nil, ctx, outputDir, "json-only")
	require.NoError(t, err)

	// JSON should exist
	_, err = os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists-EU.json"))
	require.NoError(t, err)

	// XML should NOT exist
	_, err = os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists-EU.xml"))
	assert.True(t, os.IsNotExist(err), "XML file should not exist with json-only flag")
}

func TestPublishLoTL_XMLOnly(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	ctx.AddLoTL(validLoTL())

	ctx, err := PublishLoTL(nil, ctx, outputDir, "xml-only")
	require.NoError(t, err)

	// XML should exist
	xmlData, err := os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists-EU.xml"))
	require.NoError(t, err)
	assert.Contains(t, string(xmlData), "ListAndSchemeInformation")

	// JSON should NOT exist
	_, err = os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists-EU.json"))
	assert.True(t, os.IsNotExist(err), "JSON file should not exist with xml-only flag")
}

func TestPublishLoTL_Empty(t *testing.T) {
	dir := t.TempDir()
	ctx := NewContext()

	// No LoTLs, should succeed with warning
	ctx, err := PublishLoTL(nil, ctx, dir)
	require.NoError(t, err)
}

func TestPublishLoTL_MissingArgs(t *testing.T) {
	ctx := NewContext()
	_, err := PublishLoTL(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 argument")
}

func TestPublishLoTL_FilenameNoTerritory(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	lotl := validLoTL()
	lotl.ListAndSchemeInformation.SchemeTerritory = ""

	ctx := NewContext()
	ctx.AddLoTL(lotl)

	ctx, err := PublishLoTL(nil, ctx, outputDir)
	require.NoError(t, err)

	// Should use default name without territory
	_, err = os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists.json"))
	require.NoError(t, err)
}

func TestPublishLoTE_XMLFlag(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: validSchemeInfo("SE"),
		TrustedEntitiesList:      []etsi119602.TrustedEntity{testEntity()},
	})

	ctx, err := PublishLoTE(nil, ctx, outputDir, "xml")
	require.NoError(t, err)

	// Both JSON and XML should exist
	_, err = os.ReadFile(filepath.Join(outputDir, "lote-SE.json"))
	require.NoError(t, err)

	xmlData, err := os.ReadFile(filepath.Join(outputDir, "lote-SE.xml"))
	require.NoError(t, err)
	assert.Contains(t, string(xmlData), "ListAndSchemeInformation")
}

func TestPublishLoTE_XMLOnly(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: validSchemeInfo("SE"),
		TrustedEntitiesList:      []etsi119602.TrustedEntity{testEntity()},
	})

	ctx, err := PublishLoTE(nil, ctx, outputDir, "xml-only")
	require.NoError(t, err)

	// XML should exist
	_, err = os.ReadFile(filepath.Join(outputDir, "lote-SE.xml"))
	require.NoError(t, err)

	// JSON should NOT exist
	_, err = os.ReadFile(filepath.Join(outputDir, "lote-SE.json"))
	assert.True(t, os.IsNotExist(err), "JSON file should not exist with xml-only flag")
}
