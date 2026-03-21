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

// validSchemeInfo returns a minimal valid SchemeInformation for testing.
func validSchemeInfo(territory string) etsi119602.SchemeInformation {
	return etsi119602.SchemeInformation{
		Territory:      territory,
		SchemeOperator: etsi119602.NameSet{{Language: "en", Value: "Test Operator"}},
		SchemeType:     "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric",
		IssueDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestPublishLoTE_Unsigned(t *testing.T) {
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

	ctx, err := PublishLoTE(nil, ctx, outputDir)
	require.NoError(t, err)

	// Should have created output directory and lote-SE.json
	data, err := os.ReadFile(filepath.Join(outputDir, "lote-SE.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"territory": "SE"`)
	assert.Contains(t, string(data), `"https://example.com"`)
}

func TestPublishLoTE_NoTerritory(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	si := validSchemeInfo("")
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: si,
	})

	ctx, err := PublishLoTE(nil, ctx, outputDir)
	require.NoError(t, err)

	// Should use index-based name
	_, err = os.ReadFile(filepath.Join(outputDir, "lote-0.json"))
	require.NoError(t, err)
}

func TestPublishLoTE_Empty(t *testing.T) {
	dir := t.TempDir()
	ctx := NewContext()

	// No LoTEs, should succeed with warning
	ctx, err := PublishLoTE(nil, ctx, dir)
	require.NoError(t, err)
}

func TestPublishLoTE_MissingArgs(t *testing.T) {
	ctx := NewContext()
	_, err := PublishLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 argument")
}

func TestPublishLoTE_MultipleLoTEs(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: validSchemeInfo("SE"),
	})
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: validSchemeInfo("NO"),
	})

	ctx, err := PublishLoTE(nil, ctx, outputDir)
	require.NoError(t, err)

	_, err = os.ReadFile(filepath.Join(outputDir, "lote-SE.json"))
	require.NoError(t, err)
	_, err = os.ReadFile(filepath.Join(outputDir, "lote-NO.json"))
	require.NoError(t, err)
}

func TestPublishLoTE_ValidationFailure(t *testing.T) {
	dir := t.TempDir()
	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{Version: "1.0"}) // missing required fields
	_, err := PublishLoTE(nil, ctx, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed validation")
}

func TestPublishLoTE_DistributionPointFilename(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	si := validSchemeInfo("SE")
	si.DistributionPoints = []string{"https://example.com/trusted-list.json"}
	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: si,
	})

	ctx, err := PublishLoTE(nil, ctx, outputDir)
	require.NoError(t, err)

	// Should use distribution point filename
	_, err = os.ReadFile(filepath.Join(outputDir, "trusted-list.json"))
	require.NoError(t, err)
}

func TestPublishLoTE_TerritoryCollision(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: validSchemeInfo("SE"),
	})
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: validSchemeInfo("SE"),
	})

	ctx, err := PublishLoTE(nil, ctx, outputDir)
	require.NoError(t, err)

	// First uses lote-SE.json, second should get a unique name
	_, err = os.ReadFile(filepath.Join(outputDir, "lote-SE.json"))
	require.NoError(t, err)
	_, err = os.ReadFile(filepath.Join(outputDir, "lote-SE-1.json"))
	require.NoError(t, err)
}
