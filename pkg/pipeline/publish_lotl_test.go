package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validLoTL() *etsi119602.ListOfTrustedLists {
	return &etsi119602.ListOfTrustedLists{
		Version: etsi119602.LoTEVersion,
		SchemeInformation: etsi119602.SchemeInformation{
			Territory:      "EU",
			SchemeOperator: etsi119602.NameSet{{Language: "en", Value: "Test Operator"}},
			SchemeType:     etsi119602.LoTLTypeEU,
			IssueDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		PointersToOtherLoTEs: []etsi119602.LoTEPointer{
			{
				Location:        "https://example.com/lote.json",
				SchemeTerritory: "EU",
				SchemeType:      etsi119602.LoTETypePIDProviders,
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

	// Should have created JSON file
	jsonData, err := os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists-EU.json"))
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"territory": "EU"`)

	// Should have created XML file
	xmlData, err := os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists-EU.xml"))
	require.NoError(t, err)
	assert.Contains(t, string(xmlData), "ListAndSchemeInformation")
	assert.Contains(t, string(xmlData), "Test Operator")
}

func TestPublishLoTL_JSONOnly(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	ctx.AddLoTL(validLoTL())

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
	_, err = os.ReadFile(filepath.Join(outputDir, "list_of_trusted_lists-EU.xml"))
	require.NoError(t, err)

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
	lotl.SchemeInformation.Territory = ""

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
		Version:           "1.0",
		SchemeInformation: validSchemeInfo("SE"),
		TrustedEntities: []etsi119602.TrustedEntity{
			{EntityID: "https://example.com", EntityStatus: etsi119602.StatusGranted},
		},
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
		Version:           "1.0",
		SchemeInformation: validSchemeInfo("SE"),
		TrustedEntities: []etsi119602.TrustedEntity{
			{EntityID: "https://example.com", EntityStatus: etsi119602.StatusGranted},
		},
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
