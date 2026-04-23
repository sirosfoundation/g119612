package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validSchemeInfo returns a minimal valid ListAndSchemeInformation for testing.
func validSchemeInfo(territory string) etsi119602.ListAndSchemeInformation {
	return etsi119602.ListAndSchemeInformation{
		LoTEVersionIdentifier: 1,
		SchemeTerritory:       territory,
		SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "Test Operator"}},
		LoTEType:              "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric",
		ListIssueDateTime:     "2026-01-01T00:00:00Z",
		NextUpdate:            "2027-01-01T00:00:00Z",
	}
}

func testEntity() etsi119602.TrustedEntity {
	return etsi119602.TrustedEntity{
		TrustedEntityInformation: etsi119602.TrustedEntityInformation{
			TEName: etsi119602.NameSet{{Lang: "en", Value: "https://example.com"}},
		},
		TrustedEntityServices: []etsi119602.TrustedEntityService{{
			ServiceInformation: etsi119602.ServiceInformation{
				ServiceName:   etsi119602.NameSet{{Lang: "en", Value: "Svc"}},
				ServiceStatus: etsi119602.StatusGranted,
			},
		}},
	}
}

func TestPublishLoTE_Unsigned(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	lote := &etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: validSchemeInfo("SE"),
		TrustedEntitiesList:     []etsi119602.TrustedEntity{testEntity()},
	}
	ctx.AddLoTE(lote)

	ctx, err := PublishLoTE(nil, ctx, outputDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(outputDir, "lote-SE.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"SchemeTerritory"`)
}

func TestPublishLoTE_NoTerritory(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	si := validSchemeInfo("")
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: si,
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
		ListAndSchemeInformation: validSchemeInfo("SE"),
	})
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: validSchemeInfo("NO"),
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
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{}) // missing required fields
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
		ListAndSchemeInformation: si,
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
		ListAndSchemeInformation: validSchemeInfo("SE"),
	})
	ctx.AddLoTE(&etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: validSchemeInfo("SE"),
	})

	ctx, err := PublishLoTE(nil, ctx, outputDir)
	require.NoError(t, err)

	// First uses lote-SE.json, second should get a unique name
	_, err = os.ReadFile(filepath.Join(outputDir, "lote-SE.json"))
	require.NoError(t, err)
	_, err = os.ReadFile(filepath.Join(outputDir, "lote-SE-1.json"))
	require.NoError(t, err)
}
