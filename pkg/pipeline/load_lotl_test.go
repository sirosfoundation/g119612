package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadLoTL_FromJSONFile(t *testing.T) {
	dir := t.TempDir()

	lotl := &etsi119602.ListOfTrustedLists{
		Version: etsi119602.LoTEVersion,
		SchemeInformation: etsi119602.SchemeInformation{
			Territory:      "EU",
			SchemeOperator: etsi119602.NameSet{{Language: "en", Value: "Test"}},
			SchemeType:     etsi119602.LoTLTypeEU,
			IssueDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		PointersToOtherLoTEs: []etsi119602.LoTEPointer{
			{Location: "https://example.com/lote.json", SchemeType: etsi119602.LoTETypePIDProviders},
		},
	}
	data, err := json.Marshal(lotl)
	require.NoError(t, err)

	path := filepath.Join(dir, "lotl.json")
	require.NoError(t, os.WriteFile(path, data, 0644))

	ctx := NewContext()
	ctx, err = LoadLoTL(nil, ctx, path)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTLCount())
	loaded := ctx.GetLoTLs()
	require.Len(t, loaded, 1)
	assert.Equal(t, "EU", loaded[0].SchemeInformation.Territory)
	assert.Len(t, loaded[0].PointersToOtherLoTEs, 1)
}

func TestLoadLoTL_FromXMLFile(t *testing.T) {
	dir := t.TempDir()

	// Create a LoTL, encode to XML, then load it back
	lotl := &etsi119602.ListOfTrustedLists{
		Version: etsi119602.LoTEVersion,
		SchemeInformation: etsi119602.SchemeInformation{
			Territory:      "EU",
			SchemeOperator: etsi119602.NameSet{{Language: "en", Value: "XML Operator"}},
			SchemeType:     etsi119602.LoTLTypeEU,
			IssueDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		PointersToOtherLoTEs: []etsi119602.LoTEPointer{
			{Location: "https://example.com/lote.json"},
		},
	}
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
	assert.Equal(t, "EU", loaded[0].SchemeInformation.Territory)
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
		lotl := &etsi119602.ListOfTrustedLists{
			Version: etsi119602.LoTEVersion,
			SchemeInformation: etsi119602.SchemeInformation{
				Territory:      territory,
				SchemeOperator: etsi119602.NameSet{{Language: "en", Value: "Op"}},
				SchemeType:     etsi119602.LoTLTypeEU,
				IssueDate:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		}
		data, err := json.Marshal(lotl)
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
