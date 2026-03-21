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
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
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

	_, err := FetchLoTE(srv.URL+"/bad", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Content-Type")
}
