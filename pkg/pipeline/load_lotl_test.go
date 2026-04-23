package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestLoTL(territory string, pointers ...etsi119602.OtherLoTEPointer) *etsi119602.ListOfTrustedLists {
	return &etsi119602.ListOfTrustedLists{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeTerritory:       territory,
			SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "Test"}},
			LoTEType:              etsi119602.LoTLTypeEU,
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2027-01-01T00:00:00Z",
			PointersToOtherLoTE:   pointers,
		},
	}
}

func TestLoadLoTL_FromJSONFile(t *testing.T) {
	dir := t.TempDir()

	lotl := makeTestLoTL("EU", etsi119602.OtherLoTEPointer{
		LoTELocation: "https://example.com/lote.json",
		LoTEQualifiers: []etsi119602.LoTEQualifier{{
			LoTEType: etsi119602.LoTETypePIDProviders,
		}},
	})
	data, err := lotl.MarshalLoTLIndent()
	require.NoError(t, err)

	path := filepath.Join(dir, "lotl.json")
	require.NoError(t, os.WriteFile(path, data, 0644))

	ctx := NewContext()
	ctx, err = LoadLoTL(nil, ctx, path)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTLCount())
	loaded := ctx.GetLoTLs()
	require.Len(t, loaded, 1)
	assert.Equal(t, "EU", loaded[0].ListAndSchemeInformation.SchemeTerritory)
	assert.Len(t, loaded[0].ListAndSchemeInformation.PointersToOtherLoTE, 1)
}

func TestLoadLoTL_FromXMLFile(t *testing.T) {
	dir := t.TempDir()

	lotl := makeTestLoTL("EU", etsi119602.OtherLoTEPointer{
		LoTELocation: "https://example.com/lote.json",
	})
	xmlData, err := lotl.EncodeXML()
	require.NoError(t, err)

	xmlPath := filepath.Join(dir, "lotl.xml")
	require.NoError(t, os.WriteFile(xmlPath, xmlData, 0644))

	ctx := NewContext()
	ctx, err = LoadLoTL(nil, ctx, xmlPath)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTLCount())
	loaded := ctx.GetLoTLs()
	require.Len(t, loaded, 1)
	assert.Equal(t, "EU", loaded[0].ListAndSchemeInformation.SchemeTerritory)
}

func TestLoadLoTL_MissingArgs(t *testing.T) {
	ctx := NewContext()
	_, err := LoadLoTL(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 1 argument")
}

func TestLoadLoTL_NonexistentFile(t *testing.T) {
	ctx := NewContext()
	_, err := LoadLoTL(nil, ctx, "/nonexistent/lotl.json")
	assert.Error(t, err)
}

func TestLoadLoTL_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0644))

	ctx := NewContext()
	_, err := LoadLoTL(nil, ctx, path)
	assert.Error(t, err)
}

func TestLoadLoTL_MultipleLoads(t *testing.T) {
	dir := t.TempDir()

	for i, territory := range []string{"EU", "SE"} {
		lotl := makeTestLoTL(territory)
		data, err := lotl.MarshalLoTLIndent()
		require.NoError(t, err)
		path := filepath.Join(dir, fmt.Sprintf("lotl-%d.json", i))
		require.NoError(t, os.WriteFile(path, data, 0644))
	}

	ctx := NewContext()
	var err error
	ctx, err = LoadLoTL(nil, ctx, filepath.Join(dir, "lotl-0.json"))
	require.NoError(t, err)
	ctx, err = LoadLoTL(nil, ctx, filepath.Join(dir, "lotl-1.json"))
	require.NoError(t, err)

	assert.Equal(t, 2, ctx.GetLoTLCount())
}
