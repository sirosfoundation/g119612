package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadLoTE_FromFile(t *testing.T) {
	dir := t.TempDir()
	lote := &etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeTerritory:       "SE",
			SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "Test"}},
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2027-01-01T00:00:00Z",
		},
		TrustedEntitiesList: []etsi119602.TrustedEntity{
			{
				TrustedEntityInformation: etsi119602.TrustedEntityInformation{
					TEName: etsi119602.NameSet{{Lang: "en", Value: "https://test.example.com"}},
				},
				TrustedEntityServices: []etsi119602.TrustedEntityService{{
					ServiceInformation: etsi119602.ServiceInformation{
						ServiceName: etsi119602.NameSet{{Lang: "en", Value: "Svc"}},
					},
				}},
			},
		},
	}
	data, err := lote.MarshalIndent()
	require.NoError(t, err)

	path := filepath.Join(dir, "test-lote.json")
	require.NoError(t, os.WriteFile(path, data, 0644))

	ctx := NewContext()
	ctx, err = LoadLoTE(nil, ctx, path)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTECount())
	loaded := ctx.GetLoTEs()
	require.Len(t, loaded, 1)
	assert.Equal(t, "SE", loaded[0].ListAndSchemeInformation.SchemeTerritory)
	assert.Len(t, loaded[0].TrustedEntitiesList, 1)
}

func TestLoadLoTE_MissingArgs(t *testing.T) {
	ctx := NewContext()
	_, err := LoadLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 1 argument")
}

func TestLoadLoTE_NonexistentFile(t *testing.T) {
	ctx := NewContext()
	_, err := LoadLoTE(nil, ctx, "/nonexistent/lote.json")
	assert.Error(t, err)
}

func TestLoadLoTE_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0644))

	ctx := NewContext()
	_, err := LoadLoTE(nil, ctx, path)
	assert.Error(t, err)
}

func TestLoadLoTE_XMLFile(t *testing.T) {
	dir := t.TempDir()

	// Create a valid LoTE, encode to XML
	lote := &etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeTerritory:       "SE",
			SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "XML Test"}},
			LoTEType:              "http://example.com/type",
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2027-01-01T00:00:00Z",
		},
		TrustedEntitiesList: []etsi119602.TrustedEntity{
			{
				TrustedEntityInformation: etsi119602.TrustedEntityInformation{
					TEName: etsi119602.NameSet{{Lang: "en", Value: "Entity"}},
				},
				TrustedEntityServices: []etsi119602.TrustedEntityService{{
					ServiceInformation: etsi119602.ServiceInformation{
						ServiceName:   etsi119602.NameSet{{Lang: "en", Value: "Svc"}},
						ServiceStatus: etsi119602.StatusGranted,
					},
				}},
			},
		},
	}
	xmlData, err := lote.EncodeXML()
	require.NoError(t, err)

	xmlPath := filepath.Join(dir, "test-lote.xml")
	require.NoError(t, os.WriteFile(xmlPath, xmlData, 0644))

	ctx := NewContext()
	ctx, err = LoadLoTE(nil, ctx, xmlPath)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTECount())
	loaded := ctx.GetLoTEs()
	require.Len(t, loaded, 1)
	assert.Equal(t, "SE", loaded[0].ListAndSchemeInformation.SchemeTerritory)
	assert.Equal(t, "XML Test", loaded[0].ListAndSchemeInformation.SchemeOperatorName.Get("en", ""))
}

func TestLoadLoTE_IDUnionXML(t *testing.T) {
	// Load the idunion test fixture (if available)
	xmlPath := "../etsi119602/testdata/idunion_lote.xml"
	if _, err := os.Stat(xmlPath); os.IsNotExist(err) {
		t.Skip("idunion test fixture not available")
	}

	ctx := NewContext()
	ctx, err := LoadLoTE(nil, ctx, xmlPath)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTECount())
	loaded := ctx.GetLoTEs()
	require.Len(t, loaded, 1)
	assert.GreaterOrEqual(t, len(loaded[0].TrustedEntitiesList), 1)
}

func TestLoadLoTE_AutoClassifiesLoTL(t *testing.T) {
	dir := t.TempDir()

	// Create a document with LoTL scheme type — should go to LoTLs stack
	lote := &etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeTerritory:       "EU",
			SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "EC"}},
			LoTEType:              etsi119602.LoTLTypeEU,
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2027-01-01T00:00:00Z",
			PointersToOtherLoTE: []etsi119602.OtherLoTEPointer{
				{LoTELocation: "https://example.com/lote.json"},
			},
		},
	}
	data, err := lote.MarshalIndent()
	require.NoError(t, err)

	path := filepath.Join(dir, "lotl.json")
	require.NoError(t, os.WriteFile(path, data, 0644))

	ctx := NewContext()
	ctx, err = LoadLoTE(nil, ctx, path)
	require.NoError(t, err)

	// Should be classified as LoTL, not LoTE
	assert.Equal(t, 0, ctx.GetLoTECount())
	assert.Equal(t, 1, ctx.GetLoTLCount())
	lotls := ctx.GetLoTLs()
	require.Len(t, lotls, 1)
	assert.Equal(t, "EU", lotls[0].ListAndSchemeInformation.SchemeTerritory)
	assert.Len(t, lotls[0].ListAndSchemeInformation.PointersToOtherLoTE, 1)
}
