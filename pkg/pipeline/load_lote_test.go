package pipeline

import (
	"encoding/json"
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
		Version: "1.0",
		SchemeInformation: etsi119602.SchemeInformation{
			Territory: "SE",
		},
		TrustedEntities: []etsi119602.TrustedEntity{
			{EntityID: "https://test.example.com"},
		},
	}
	data, err := json.Marshal(lote)
	require.NoError(t, err)

	path := filepath.Join(dir, "test-lote.json")
	require.NoError(t, os.WriteFile(path, data, 0644))

	ctx := NewContext()
	ctx, err = LoadLoTE(nil, ctx, path)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTECount())
	loaded := ctx.GetLoTEs()
	require.Len(t, loaded, 1)
	assert.Equal(t, "SE", loaded[0].SchemeInformation.Territory)
	assert.Len(t, loaded[0].TrustedEntities, 1)
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
