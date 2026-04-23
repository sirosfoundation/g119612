package pipeline

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// --- parseCertificateFile coverage ---

func TestParseCertificateFile_DER(t *testing.T) {
	cert := generateTestCertForPipeline(t, "DER Cert")
	result, err := parseCertificateFile(cert.Raw)
	require.NoError(t, err)
	assert.Equal(t, "DER Cert", result.Subject.CommonName)
}

func TestParseCertificateFile_PEM(t *testing.T) {
	cert := generateTestCertForPipeline(t, "PEM Cert")
	pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	result, err := parseCertificateFile(pemData)
	require.NoError(t, err)
	assert.Equal(t, "PEM Cert", result.Subject.CommonName)
}

func TestParseCertificateFile_InvalidData(t *testing.T) {
	_, err := parseCertificateFile([]byte("not a certificate"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not valid DER or PEM")
}

func TestParseCertificateFile_WrongPEMType(t *testing.T) {
	pemData := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte{0x01}})
	_, err := parseCertificateFile(pemData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected CERTIFICATE")
}

// --- loadLoTEEntity coverage (JWK, DID, CRT files) ---

func TestLoadLoTEEntity_WithPEM(t *testing.T) {
	dir := t.TempDir()
	writeEntityYAML(t, dir, "Test Entity", "urn:test:entity", etsi119602.StatusGranted)
	cert := generateTestCertForPipeline(t, "Entity CA")
	pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ca.pem"), pemData, 0600))

	entity, err := loadLoTEEntity(dir)
	require.NoError(t, err)
	assert.Equal(t, "Test Entity", entity.TrustedEntityInformation.TEName[0].Value)
	require.Len(t, entity.TrustedEntityServices, 1)
	sdi := entity.TrustedEntityServices[0].ServiceInformation.ServiceDigitalIdentity
	require.Len(t, sdi.X509Certificates, 1)
}

func TestLoadLoTEEntity_WithCRT(t *testing.T) {
	dir := t.TempDir()
	writeEntityYAML(t, dir, "CRT Entity", "urn:test:crt", etsi119602.StatusGranted)
	cert := generateTestCertForPipeline(t, "CRT CA")
	// Write raw DER as .crt
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ca.crt"), cert.Raw, 0600))

	entity, err := loadLoTEEntity(dir)
	require.NoError(t, err)
	require.Len(t, entity.TrustedEntityServices, 1)
	sdi := entity.TrustedEntityServices[0].ServiceInformation.ServiceDigitalIdentity
	require.Len(t, sdi.X509Certificates, 1)
}

func TestLoadLoTEEntity_WithJWK(t *testing.T) {
	dir := t.TempDir()
	writeEntityYAML(t, dir, "JWK Entity", "urn:test:jwk", etsi119602.StatusGranted)
	jwk := map[string]any{
		"kty": "EC",
		"crv": "P-256",
		"x":   "f83OJ3D2xF1Bg8vub9tLe1gHMzV76e8Tus9uPHvRVEU",
		"y":   "x_FEzRu9m36HLN_tue659LNpXW6pCyStikYjKIWI5a0",
	}
	jwkData, err := json.Marshal(jwk)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "key.jwk"), jwkData, 0600))

	entity, err := loadLoTEEntity(dir)
	require.NoError(t, err)
	require.Len(t, entity.TrustedEntityServices, 1)
	sdi := entity.TrustedEntityServices[0].ServiceInformation.ServiceDigitalIdentity
	require.Len(t, sdi.PublicKeyValues, 1)
}

func TestLoadLoTEEntity_WithDID(t *testing.T) {
	dir := t.TempDir()
	writeEntityYAML(t, dir, "DID Entity", "urn:test:did", etsi119602.StatusGranted)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "identity.did"), []byte("did:web:example.com"), 0600))

	entity, err := loadLoTEEntity(dir)
	require.NoError(t, err)
	require.Len(t, entity.TrustedEntityServices, 1)
	sdi := entity.TrustedEntityServices[0].ServiceInformation.ServiceDigitalIdentity
	require.Len(t, sdi.OtherIds, 1)
	assert.Equal(t, "did:web:example.com", sdi.OtherIds[0])
}

func TestLoadLoTEEntity_InvalidDID(t *testing.T) {
	dir := t.TempDir()
	writeEntityYAML(t, dir, "Bad DID", "urn:test:bad-did", etsi119602.StatusGranted)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.did"), []byte("not-a-did"), 0600))

	_, err := loadLoTEEntity(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must start with 'did:'")
}

func TestLoadLoTEEntity_MissingEntityYAML(t *testing.T) {
	dir := t.TempDir()
	_, err := loadLoTEEntity(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read entity.yaml")
}

func TestLoadLoTEEntity_InvalidEntityYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "entity.yaml"), []byte("not: [valid: yaml: !!"), 0600))
	_, err := loadLoTEEntity(dir)
	assert.Error(t, err)
}

func TestLoadLoTEEntity_NoNames(t *testing.T) {
	dir := t.TempDir()
	meta := LoTEEntityMetadata{EntityID: "urn:test:noname", Status: etsi119602.StatusGranted}
	data, err := yaml.Marshal(meta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "entity.yaml"), data, 0600))

	_, err = loadLoTEEntity(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have at least one name")
}

func TestLoadLoTEEntity_DefaultStatus(t *testing.T) {
	dir := t.TempDir()
	// Write entity.yaml without status
	content := "names:\n  - language: en\n    value: Default Status\nentityId: urn:test:default\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "entity.yaml"), []byte(content), 0600))

	entity, err := loadLoTEEntity(dir)
	require.NoError(t, err)
	assert.Equal(t, etsi119602.StatusGranted, entity.TrustedEntityServices[0].ServiceInformation.ServiceStatus)
}

func TestLoadLoTEEntity_WithServices(t *testing.T) {
	dir := t.TempDir()
	content := `names:
  - language: en
    value: Service Entity
entityId: urn:test:services
status: http://status/granted
services:
  - serviceNames:
      - language: en
        value: My Service
    serviceType: http://type/test
    status: http://status/active
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "entity.yaml"), []byte(content), 0600))

	entity, err := loadLoTEEntity(dir)
	require.NoError(t, err)
	require.Len(t, entity.TrustedEntityServices, 1)
	assert.Equal(t, "http://type/test", entity.TrustedEntityServices[0].ServiceInformation.ServiceTypeIdentifier)
}

// --- GenerateLoTE coverage ---

func TestGenerateLoTE_NoArgs(t *testing.T) {
	ctx := NewContext()
	_, err := GenerateLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 1 argument")
}

func TestGenerateLoTE_MissingSchemeYAML(t *testing.T) {
	dir := t.TempDir()
	ctx := NewContext()
	_, err := GenerateLoTE(nil, ctx, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scheme.yaml")
}

func TestGenerateLoTE_NoEntitiesDir(t *testing.T) {
	dir := t.TempDir()
	writeSchemeYAML(t, dir, "Test Operator", "http://test/type", "SE")

	ctx := NewContext()
	result, err := GenerateLoTE(nil, ctx, dir)
	require.NoError(t, err)
	assert.Equal(t, 1, result.LoTEs.Size())
	lote, ok := result.LoTEs.Peek()
	require.True(t, ok)
	assert.Empty(t, lote.TrustedEntitiesList)
}

func TestGenerateLoTE_WithEntity(t *testing.T) {
	dir := t.TempDir()
	writeSchemeYAML(t, dir, "Test Operator", "http://test/type", "SE")

	entitiesDir := filepath.Join(dir, "entities", "entity1")
	require.NoError(t, os.MkdirAll(entitiesDir, 0750))
	writeEntityYAML(t, entitiesDir, "Test Entity", "urn:test:e1", etsi119602.StatusGranted)

	cert := generateTestCertForPipeline(t, "Entity Cert")
	pemData := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	require.NoError(t, os.WriteFile(filepath.Join(entitiesDir, "cert.pem"), pemData, 0600))

	ctx := NewContext()
	result, err := GenerateLoTE(nil, ctx, dir)
	require.NoError(t, err)
	require.Equal(t, 1, result.LoTEs.Size())
	lote, ok := result.LoTEs.Peek()
	require.True(t, ok)
	require.Len(t, lote.TrustedEntitiesList, 1)
	assert.Equal(t, "Test Entity", lote.TrustedEntitiesList[0].TrustedEntityInformation.TEName[0].Value)
}

// --- LoadLoTE coverage ---

func TestLoadLoTE_NoArgs(t *testing.T) {
	ctx := NewContext()
	_, err := LoadLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires 1 argument")
}

func TestLoadLoTE_FileSuccess(t *testing.T) {
	dir := t.TempDir()
	lote := &etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeTerritory:       "SE",
			SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "Test"}},
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2027-01-01T00:00:00Z",
		},
	}
	data, err := lote.MarshalIndent()
	require.NoError(t, err)
	path := filepath.Join(dir, "test.json")
	require.NoError(t, os.WriteFile(path, data, 0600))

	ctx := NewContext()
	result, err := LoadLoTE(nil, ctx, path)
	require.NoError(t, err)
	require.Equal(t, 1, result.LoTEs.Size())
	peek, ok := result.LoTEs.Peek()
	require.True(t, ok)
	assert.Equal(t, "SE", peek.ListAndSchemeInformation.SchemeTerritory)
}

func TestLoadLoTE_InvalidFile(t *testing.T) {
	ctx := NewContext()
	_, err := LoadLoTE(nil, ctx, "/nonexistent/path.json")
	assert.Error(t, err)
}

func TestLoadLoTE_InvalidCertPath(t *testing.T) {
	dir := t.TempDir()

	// Use invalid JSON so the initial load fails and it tries JWS verification
	path := filepath.Join(dir, "test.jws")
	require.NoError(t, os.WriteFile(path, []byte("not-a-valid-jws"), 0600))

	ctx := NewContext()
	_, err := LoadLoTE(nil, ctx, path, "/nonexistent/cert.pem")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create JWS verifier")
}

// --- PublishLoTE coverage ---

func TestPublishLoTE_NoArgs(t *testing.T) {
	ctx := NewContext()
	_, err := PublishLoTE(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 argument")
}

func TestPublishLoTE_UnsignedSuccess(t *testing.T) {
	dir := t.TempDir()
	ctx := NewContext()
	ctx.EnsureLoTEs()
	lote := &etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeTerritory:       "SE",
			SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "Operator"}},
			LoTEType:              "http://type/test",
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2027-01-01T00:00:00Z",
		},
	}
	ctx.LoTEs.Push(lote)

	result, err := PublishLoTE(nil, ctx, dir)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify file was written
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, files)
}

func TestPublishLoTE_EmptyLoTEs(t *testing.T) {
	dir := t.TempDir()
	ctx := NewContext()
	result, err := PublishLoTE(nil, ctx, dir)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPublishLoTE_WithFileSigner(t *testing.T) {
	dir := t.TempDir()
	// Generate cert and key for signing
	certPath, keyPath := generateTestCertAndKeyForPipeline(t, dir)

	outputDir := filepath.Join(dir, "output")

	ctx := NewContext()
	ctx.EnsureLoTEs()
	lote := &etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			SchemeTerritory:       "FI",
			SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "Finnish Op"}},
			LoTEType:              "http://type/fi",
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2027-01-01T00:00:00Z",
		},
	}
	ctx.LoTEs.Push(lote)

	result, err := PublishLoTE(nil, ctx, outputDir, certPath, keyPath)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should have both .json and .json.jws files
	files, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	fileNames := make([]string, len(files))
	for i, f := range files {
		fileNames[i] = f.Name()
	}
	assert.Contains(t, fileNames, "lote-FI.json")
	assert.Contains(t, fileNames, "lote-FI.json.jws")
}

// --- IncrementLoTESequence coverage ---

func TestIncrementLoTESequence_NoLoTEs(t *testing.T) {
	ctx := NewContext()
	_, err := IncrementLoTESequence(nil, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no LoTEs")
}

func TestIncrementLoTESequence_Success(t *testing.T) {
	ctx := NewContext()
	ctx.EnsureLoTEs()
	lote := &etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: etsi119602.ListAndSchemeInformation{
			LoTEVersionIdentifier: 1,
			LoTESequenceNumber:    5,
			SchemeOperatorName:    etsi119602.NameSet{{Lang: "en", Value: "Test"}},
			ListIssueDateTime:     "2026-01-01T00:00:00Z",
			NextUpdate:            "2027-01-01T00:00:00Z",
		},
	}
	ctx.LoTEs.Push(lote)

	_, err := IncrementLoTESequence(nil, ctx)
	require.NoError(t, err)
	assert.Equal(t, 6, lote.ListAndSchemeInformation.LoTESequenceNumber)
}

// --- createLoTESigner coverage ---

func TestCreateLoTESigner_NoArgs(t *testing.T) {
	signer, err := createLoTESigner(nil)
	assert.NoError(t, err)
	assert.Nil(t, signer)
}

func TestCreateLoTESigner_FileBased(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCertAndKeyForPipeline(t, dir)

	signer, err := createLoTESigner([]string{certPath, keyPath})
	require.NoError(t, err)
	assert.NotNil(t, signer)
}

func TestCreateLoTESigner_InvalidPKCS11URI(t *testing.T) {
	_, err := createLoTESigner([]string{"pkcs11:invalid"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PKCS#11 URI")
}

func TestCreateLoTESigner_InsufficientArgs(t *testing.T) {
	_, err := createLoTESigner([]string{"/only/cert.pem"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires cert and key")
}

// --- helpers ---

func writeSchemeYAML(t *testing.T, dir, operatorName, schemeType, territory string) {
	t.Helper()
	scheme := LoTESchemeMetadata{
		OperatorNames: []MultiLangName{{Language: "en", Value: operatorName}},
		SchemeType:    schemeType,
		Territory:     territory,
	}
	data, err := yaml.Marshal(scheme)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scheme.yaml"), data, 0600))
}

func writeEntityYAML(t *testing.T, dir, name, entityID, status string) {
	t.Helper()
	meta := LoTEEntityMetadata{
		Names:    []MultiLangName{{Language: "en", Value: name}},
		EntityID: entityID,
		Status:   status,
		Address: &Address{
			Postal: struct {
				StreetAddress   string `yaml:"streetAddress"`
				Locality        string `yaml:"locality"`
				StateOrProvince string `yaml:"stateOrProvince,omitempty"`
				PostalCode      string `yaml:"postalCode,omitempty"`
				CountryName     string `yaml:"countryName"`
			}{
				StreetAddress: "Test Street 1",
				Locality:      "Test City",
				CountryName:   "SE",
			},
			Electronic: []string{"mailto:test@example.com"},
		},
		InformationURI: []MultiLangName{{Language: "en", Value: "https://example.com"}},
	}
	data, err := yaml.Marshal(meta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "entity.yaml"), data, 0600))
}

func generateTestCertForPipeline(t *testing.T, cn string) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

func generateTestCertAndKeyForPipeline(t *testing.T, dir string) (certPath, keyPath string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-signer"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	certPath = filepath.Join(dir, "cert.pem")
	certFile, err := os.Create(certPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	certFile.Close()

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	keyPath = filepath.Join(dir, "key.pem")
	keyFile, err := os.Create(keyPath)
	require.NoError(t, err)
	require.NoError(t, pem.Encode(keyFile, &pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
	keyFile.Close()

	return certPath, keyPath
}
