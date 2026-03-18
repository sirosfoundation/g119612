package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishLoTE_Unsigned(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version: "1.0",
		SchemeInformation: etsi119602.SchemeInformation{
			Territory: "SE",
		},
		TrustedEntities: []etsi119602.TrustedEntity{
			{EntityID: "https://example.com"},
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
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version: "1.0",
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
		Version: "1.0",
		SchemeInformation: etsi119602.SchemeInformation{Territory: "SE"},
	})
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		Version: "1.0",
		SchemeInformation: etsi119602.SchemeInformation{Territory: "NO"},
	})

	ctx, err := PublishLoTE(nil, ctx, outputDir)
	require.NoError(t, err)

	_, err = os.ReadFile(filepath.Join(outputDir, "lote-SE.json"))
	require.NoError(t, err)
	_, err = os.ReadFile(filepath.Join(outputDir, "lote-NO.json"))
	require.NoError(t, err)
}
