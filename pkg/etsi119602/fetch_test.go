package etsi119602

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchLoTE_File(t *testing.T) {
	lote := testLoTE()
	dir := t.TempDir()
	path := filepath.Join(dir, "lote.json")

	data, err := lote.MarshalIndent()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))

	fetched, err := FetchLoTE(path, nil)
	require.NoError(t, err)
	assert.Equal(t, "SE", fetched.ListAndSchemeInformation.SchemeTerritory)
}

func TestFetchLoTE_FileURI(t *testing.T) {
	lote := testLoTE()
	dir := t.TempDir()
	path := filepath.Join(dir, "lote.json")

	data, err := lote.MarshalIndent()
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))

	fetched, err := FetchLoTE("file://"+path, nil)
	require.NoError(t, err)
	assert.Equal(t, "SE", fetched.ListAndSchemeInformation.SchemeTerritory)
}

func TestFetchLoTE_HTTP(t *testing.T) {
	lote := testLoTE()
	data, err := lote.MarshalIndent()
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer srv.Close()

	fetched, err := FetchLoTE(srv.URL+"/lote.json", nil)
	require.NoError(t, err)
	assert.Equal(t, "SE", fetched.ListAndSchemeInformation.SchemeTerritory)
}

func TestFetchLoTE_XMLFile(t *testing.T) {
	lote := testLoTE()
	dir := t.TempDir()
	path := filepath.Join(dir, "lote.xml")

	require.NoError(t, lote.EncodeXMLToFile(path))

	fetched, err := FetchLoTE(path, nil)
	require.NoError(t, err)
	assert.Equal(t, "SE", fetched.ListAndSchemeInformation.SchemeTerritory)
}

func TestFetchLoTE_XMLAutoDetect(t *testing.T) {
	lote := testLoTE()
	dir := t.TempDir()

	// Write XML content with .dat extension (no XML extension)
	xmlData, err := lote.EncodeXML()
	require.NoError(t, err)
	path := filepath.Join(dir, "lote.dat")
	require.NoError(t, os.WriteFile(path, xmlData, 0644))

	fetched, err := FetchLoTE(path, nil)
	require.NoError(t, err)
	assert.Equal(t, "SE", fetched.ListAndSchemeInformation.SchemeTerritory)
}

func TestFetchLoTL_File(t *testing.T) {
	lotl := &ListOfTrustedLists{
		ListAndSchemeInformation: ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			LoTESequenceNumber:    1,
			LoTEType:              LoTLTypeEU,
			SchemeOperatorName:    NameSet{{Lang: "en", Value: "EU"}},
			SchemeTerritory:       "EU",
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2026-07-01T00:00:00Z",
			PointersToOtherLoTE: []OtherLoTEPointer{
				{LoTELocation: "https://example.com/lote.json"},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "lotl.json")
	data, err := json.MarshalIndent(LoTEDocument{LoTE: *lotl}, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))

	fetched, err := FetchLoTL(path, nil)
	require.NoError(t, err)
	assert.Equal(t, "EU", fetched.ListAndSchemeInformation.SchemeTerritory)
	assert.True(t, fetched.IsLoTL())
}

func TestFetchLoTE_XMLContentType(t *testing.T) {
	lote := testLoTE()
	xmlData, err := lote.EncodeXML()
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write(xmlData)
	}))
	defer srv.Close()

	fetched, err := FetchLoTE(srv.URL+"/lote.json", nil)
	require.NoError(t, err)
	assert.Equal(t, "SE", fetched.ListAndSchemeInformation.SchemeTerritory)
}

func TestFetchRaw(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0644))

	data, err := FetchRaw(path, nil)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestIsXMLContent(t *testing.T) {
	assert.True(t, IsXMLContent([]byte("<?xml version=\"1.0\"?><root/>")))
	assert.True(t, IsXMLContent([]byte("<root/>")))
	assert.True(t, IsXMLContent([]byte("  <?xml version=\"1.0\"?>")))
	assert.False(t, IsXMLContent([]byte("{\"LoTE\": {}}")))
	assert.False(t, IsXMLContent([]byte("")))
}
