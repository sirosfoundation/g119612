package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateLoTE_Basic(t *testing.T) {
	dir := t.TempDir()

	// Create scheme.yaml
	schemeYAML := `operatorNames:
  - language: en
    value: "Test Operator"
schemeType: "http://example.com/lote"
territory: SE
sequenceNumber: 1
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte(schemeYAML), 0644))

	// Create entities directory with one entity
	entityDir := filepath.Join(dir, "entities", "entity1")
	require.NoError(t, os.MkdirAll(entityDir, 0755))

	entityYAML := `names:
  - language: en
    value: "Test Issuer"
entityId: "https://issuer.example.com"
entityType: "credential-issuer"
status: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted"
`
	require.NoError(t, os.WriteFile(filepath.Join(entityDir, "entity.yaml"), []byte(entityYAML), 0644))

	// Create JWK file
	jwkJSON := `{"kty": "EC", "crv": "P-256", "x": "test", "y": "test"}`
	require.NoError(t, os.WriteFile(filepath.Join(entityDir, "key1.jwk"), []byte(jwkJSON), 0644))

	ctx := NewContext()
	ctx, err := GenerateLoTE(nil, ctx, dir)
	require.NoError(t, err)

	assert.Equal(t, 1, ctx.GetLoTECount())
	lotes := ctx.GetLoTEs()
	require.Len(t, lotes, 1)

	lote := lotes[0]
	assert.Equal(t, "1.0", lote.Version)
	assert.Equal(t, "SE", lote.SchemeInformation.Territory)
	assert.Len(t, lote.TrustedEntities, 1)
	assert.Equal(t, "https://issuer.example.com", lote.TrustedEntities[0].EntityID)
	assert.Len(t, lote.TrustedEntities[0].DigitalIdentities, 1)
	assert.Equal(t, "jwk", lote.TrustedEntities[0].DigitalIdentities[0].Type)
}

func TestGenerateLoTE_EmptyEntities(t *testing.T) {
	dir := t.TempDir()

	schemeYAML := `operatorNames:
  - language: en
    value: "Test Operator"
schemeType: "http://example.com/lote"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte(schemeYAML), 0644))

	// No entities directory — should still succeed with empty list
	ctx := NewContext()
	ctx, err := GenerateLoTE(nil, ctx, dir)
	require.NoError(t, err)
	assert.Equal(t, 1, ctx.GetLoTECount())
	assert.Empty(t, ctx.GetLoTEs()[0].TrustedEntities)
}

func TestGenerateLoTE_MissingScheme(t *testing.T) {
	dir := t.TempDir()
	ctx := NewContext()
	_, err := GenerateLoTE(nil, ctx, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scheme.yaml")
}

func TestGenerateLoTE_InvalidScheme(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte("not: valid: yaml: ["), 0644))
	ctx := NewContext()
	_, err := GenerateLoTE(nil, ctx, dir)
	assert.Error(t, err)
}

func TestGenerateLoTE_MissingArgs(t *testing.T) {
	ctx := NewContext()
	_, err := GenerateLoTE(nil, ctx)
	assert.Error(t, err)
}

func TestGenerateLoTE_EmptySchemaFields(t *testing.T) {
	dir := t.TempDir()
	// Missing operatorNames
	schemeYAML := `schemeType: "http://example.com/lote"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte(schemeYAML), 0644))
	ctx := NewContext()
	_, err := GenerateLoTE(nil, ctx, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operatorName")
}

func TestGenerateLoTE_EntityWithServices(t *testing.T) {
	dir := t.TempDir()

	schemeYAML := `operatorNames:
  - language: en
    value: "Ops"
schemeType: "http://example.com/lote"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte(schemeYAML), 0644))

	entityDir := filepath.Join(dir, "entities", "e1")
	require.NoError(t, os.MkdirAll(entityDir, 0755))

	entityYAML := `names:
  - language: en
    value: "Multi-service Entity"
entityId: "https://multi.example.com"
status: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted"
services:
  - serviceNames:
      - language: en
        value: "Issuing Service"
    serviceType: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC"
    status: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted"
  - serviceNames:
      - language: en
        value: "Signing Service"
    serviceType: "http://uri.etsi.org/TrstSvc/Svctype/TSA/QTST"
    status: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted"
`
	require.NoError(t, os.WriteFile(filepath.Join(entityDir, "entity.yaml"), []byte(entityYAML), 0644))

	ctx := NewContext()
	ctx, err := GenerateLoTE(nil, ctx, dir)
	require.NoError(t, err)

	lote := ctx.GetLoTEs()[0]
	require.Len(t, lote.TrustedEntities, 1)
	assert.Len(t, lote.TrustedEntities[0].Services, 2)
}
