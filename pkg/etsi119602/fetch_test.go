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

func TestFetchLoTE_PlainFilePath(t *testing.T) {
	dir := t.TempDir()
	lote := &ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: SchemeInformation{Territory: "SE"},
	}
	data, err := json.Marshal(lote)
	require.NoError(t, err)

	path := filepath.Join(dir, "test.json")
	require.NoError(t, os.WriteFile(path, data, 0644))

	result, err := FetchLoTE(path, nil)
	require.NoError(t, err)
	assert.Equal(t, "SE", result.SchemeInformation.Territory)
}

func TestFetchLoTE_FilePrefix(t *testing.T) {
	dir := t.TempDir()
	lote := &ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: SchemeInformation{Territory: "NO"},
	}
	data, err := json.Marshal(lote)
	require.NoError(t, err)

	path := filepath.Join(dir, "test.json")
	require.NoError(t, os.WriteFile(path, data, 0644))

	result, err := FetchLoTE("file://"+path, nil)
	require.NoError(t, err)
	assert.Equal(t, "NO", result.SchemeInformation.Territory)
}

func TestFetchLoTE_NonexistentFile(t *testing.T) {
	_, err := FetchLoTE("/nonexistent/path.json", nil)
	assert.Error(t, err)
}

func TestFetchLoTE_HTTPSuccess(t *testing.T) {
	lote := &ListOfTrustedEntities{
		Version:           "1.0",
		SchemeInformation: SchemeInformation{Territory: "FI"},
		TrustedEntities: []TrustedEntity{
			{EntityID: "https://test.fi", EntityStatus: StatusGranted},
		},
	}
	data, err := json.Marshal(lote)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer srv.Close()

	result, err := FetchLoTE(srv.URL+"/lote.json", nil)
	require.NoError(t, err)
	assert.Equal(t, "FI", result.SchemeInformation.Territory)
	assert.Len(t, result.TrustedEntities, 1)
}

func TestFetchLoTE_HTTPWithOptions(t *testing.T) {
	lote := &ListOfTrustedEntities{Version: "1.0"}
	data, err := json.Marshal(lote)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "TestAgent/1.0", r.Header.Get("User-Agent"))
		w.Write(data)
	}))
	defer srv.Close()

	opts := &FetchOptions{
		UserAgent: "TestAgent/1.0",
	}
	result, err := FetchLoTE(srv.URL+"/lote.json", opts)
	require.NoError(t, err)
	assert.Equal(t, "1.0", result.Version)
}

func TestFetchLoTE_HTTPCustomClient(t *testing.T) {
	lote := &ListOfTrustedEntities{Version: "1.0"}
	data, err := json.Marshal(lote)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	opts := &FetchOptions{
		HTTPClient: srv.Client(),
	}
	result, err := FetchLoTE(srv.URL+"/lote.json", opts)
	require.NoError(t, err)
	assert.Equal(t, "1.0", result.Version)
}

func TestFetchLoTE_HTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := FetchLoTE(srv.URL+"/missing.json", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestFetchLoTE_HTTPInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := FetchLoTE(srv.URL+"/bad.json", nil)
	assert.Error(t, err)
}

func TestFetchLoTE_HTTPUnreachable(t *testing.T) {
	_, err := FetchLoTE("http://127.0.0.1:1/unreachable", nil)
	assert.Error(t, err)
}

func TestFetchLoTE_ContentTypeHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<html>not json</html>"))
	}))
	defer srv.Close()

	_, err := FetchLoTE(srv.URL+"/bad", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Content-Type")
}

func TestFetchLoTE_ContentTypeJOSE(t *testing.T) {
	lote := &ListOfTrustedEntities{Version: "1.0"}
	data, err := json.Marshal(lote)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/jose")
		w.Write(data)
	}))
	defer srv.Close()

	result, err := FetchLoTE(srv.URL+"/lote.jose", nil)
	require.NoError(t, err)
	assert.Equal(t, "1.0", result.Version)
}

func TestFetchLoTE_ContentTypeOctetStream(t *testing.T) {
	lote := &ListOfTrustedEntities{Version: "1.0"}
	data, err := json.Marshal(lote)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)
	}))
	defer srv.Close()

	result, err := FetchLoTE(srv.URL+"/lote", nil)
	require.NoError(t, err)
	assert.Equal(t, "1.0", result.Version)
}

func TestFetchLoTE_ContentTypeXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte("<xml/>"))
	}))
	defer srv.Close()

	// XML content type is now accepted; <xml/> parses as an empty LoTE
	result, err := FetchLoTE(srv.URL+"/lote", nil)
	require.NoError(t, err)
	// Result should be an empty LoTE since <xml/> has no matching fields
	assert.Equal(t, "", result.SchemeInformation.Territory)
}

func TestFetchLoTE_XMLFile(t *testing.T) {
	dir := t.TempDir()

	// Write a minimal LoTE XML file
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ListOfTrustedEntitiesType LOTETag="http://uri.etsi.org/019602/LOTETag">
  <lote:ListAndSchemeInformation>
    <LoTEVersionIdentifier>1</LoTEVersionIdentifier>
    <lote:LoTEType>http://example.com/type</lote:LoTEType>
    <lote:SchemeOperatorName>
      <Name xml:lang="en">Test</Name>
    </lote:SchemeOperatorName>
    <ListIssueDateTime>2026-01-01T00:00:00Z</ListIssueDateTime>
  </lote:ListAndSchemeInformation>
</ListOfTrustedEntitiesType>`

	xmlPath := filepath.Join(dir, "test-lote.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(xmlData), 0644))

	result, err := FetchLoTE(xmlPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "Test", result.SchemeInformation.SchemeOperator.Get("en", ""))
}

func TestFetchLoTE_XMLFileWithFilePrefix(t *testing.T) {
	dir := t.TempDir()

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ListOfTrustedEntitiesType LOTETag="http://uri.etsi.org/019602/LOTETag">
  <lote:ListAndSchemeInformation>
    <LoTEVersionIdentifier>1</LoTEVersionIdentifier>
    <lote:LoTEType>http://example.com/type</lote:LoTEType>
    <lote:SchemeOperatorName>
      <Name xml:lang="en">Operator</Name>
    </lote:SchemeOperatorName>
    <ListIssueDateTime>2026-01-01T00:00:00Z</ListIssueDateTime>
  </lote:ListAndSchemeInformation>
</ListOfTrustedEntitiesType>`

	xmlPath := filepath.Join(dir, "test.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(xmlData), 0644))

	result, err := FetchLoTE("file://"+xmlPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "Operator", result.SchemeInformation.SchemeOperator.Get("en", ""))
}

func TestFetchLoTE_HTTPAutoDetectXML(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ListOfTrustedEntitiesType LOTETag="http://uri.etsi.org/019602/LOTETag">
  <lote:ListAndSchemeInformation>
    <LoTEVersionIdentifier>1</LoTEVersionIdentifier>
    <lote:LoTEType>http://example.com/type</lote:LoTEType>
    <lote:SchemeOperatorName>
      <Name xml:lang="en">HTTPOp</Name>
    </lote:SchemeOperatorName>
    <ListIssueDateTime>2026-01-01T00:00:00Z</ListIssueDateTime>
  </lote:ListAndSchemeInformation>
</ListOfTrustedEntitiesType>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(xmlData))
	}))
	defer srv.Close()

	result, err := FetchLoTE(srv.URL+"/lote.xml", nil)
	require.NoError(t, err)
	assert.Equal(t, "HTTPOp", result.SchemeInformation.SchemeOperator.Get("en", ""))
}

func TestFetchLoTL_FromJSONFile(t *testing.T) {
	dir := t.TempDir()

	lotl := &ListOfTrustedLists{
		Version: LoTEVersion,
		SchemeInformation: SchemeInformation{
			Territory:  "EU",
			SchemeType: LoTLTypeEU,
		},
		PointersToOtherLoTEs: []LoTEPointer{
			{Location: "https://example.com/lote.json"},
		},
	}
	data, err := json.Marshal(lotl)
	require.NoError(t, err)

	path := filepath.Join(dir, "lotl.json")
	require.NoError(t, os.WriteFile(path, data, 0644))

	result, err := FetchLoTL(path, nil)
	require.NoError(t, err)
	assert.Equal(t, "EU", result.SchemeInformation.Territory)
	assert.Len(t, result.PointersToOtherLoTEs, 1)
}

func TestFetchLoTEXML_FromFile(t *testing.T) {
	dir := t.TempDir()

	xmlData := `<?xml version="1.0" encoding="UTF-8"?>
<ListOfTrustedEntitiesType LOTETag="http://uri.etsi.org/019602/LOTETag">
  <lote:ListAndSchemeInformation>
    <LoTEVersionIdentifier>1</LoTEVersionIdentifier>
    <lote:LoTEType>http://example.com/type</lote:LoTEType>
    <lote:SchemeOperatorName>
      <Name xml:lang="en">ExplicitXML</Name>
    </lote:SchemeOperatorName>
    <ListIssueDateTime>2026-01-01T00:00:00Z</ListIssueDateTime>
  </lote:ListAndSchemeInformation>
</ListOfTrustedEntitiesType>`

	xmlPath := filepath.Join(dir, "lote.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(xmlData), 0644))

	result, err := FetchLoTEXML(xmlPath, nil)
	require.NoError(t, err)
	assert.Equal(t, "ExplicitXML", result.SchemeInformation.SchemeOperator.Get("en", ""))
}

func TestFetchRaw_FromFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("test content")
	path := filepath.Join(dir, "raw.txt")
	require.NoError(t, os.WriteFile(path, content, 0644))

	result, err := FetchRaw(path, nil)
	require.NoError(t, err)
	assert.Equal(t, content, result)
}

func TestIsXMLLocation(t *testing.T) {
	assert.True(t, isXMLLocation("https://example.com/lote.xml"))
	assert.True(t, isXMLLocation("https://example.com/lote.XML"))
	assert.True(t, isXMLLocation("/path/to/file.xml"))
	assert.True(t, isXMLLocation("https://example.com/lote.xml?v=1"))
	assert.False(t, isXMLLocation("https://example.com/lote.json"))
	assert.False(t, isXMLLocation("/path/to/file.json"))
	assert.False(t, isXMLLocation("https://example.com/api/lote"))
}

func TestIsXMLContent(t *testing.T) {
	assert.True(t, isXMLContent([]byte(`<?xml version="1.0"?><root/>`)))
	assert.True(t, isXMLContent([]byte(`  <?xml version="1.0"?>`)))
	assert.True(t, isXMLContent([]byte(`<root/>`)))
	assert.False(t, isXMLContent([]byte(`{"version": "1.0"}`)))
	assert.False(t, isXMLContent([]byte(`not xml`)))
}
