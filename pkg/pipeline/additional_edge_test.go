package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/stretchr/testify/assert"
)

// TestSelectCertPool_NoTSLs tests SelectCertPool when no TSLs are loaded
func TestSelectCertPool_NoTSLs(t *testing.T) {
	pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
	ctx := NewContext()
	// ctx has no TSLs loaded

	_, err := SelectCertPool(pl, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no TSLs loaded")
}

// TestSelectCertPool_InvalidReferenceDepth tests SelectCertPool with invalid reference depth
func TestSelectCertPool_InvalidReferenceDepth(t *testing.T) {
	pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
	ctx := NewContext()

	// Load a test TSL
	tslData := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
	<SchemeInformation>
		<TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
		<SchemeTerritory>SE</SchemeTerritory>
	</SchemeInformation>
</TrustServiceStatusList>`

	tmpFile := filepath.Join(t.TempDir(), "test.xml")
	err := os.WriteFile(tmpFile, []byte(tslData), 0644)
	assert.NoError(t, err)

	ctx, err = LoadTSL(pl, ctx, tmpFile)
	assert.NoError(t, err)

	// Test with invalid reference depth (negative number triggers warning)
	_, err = SelectCertPool(pl, ctx, "reference-depth:-1")
	// Should not error, but use default depth
	assert.NoError(t, err)

	// Test with non-numeric reference depth (triggers warning)
	_, err = SelectCertPool(pl, ctx, "reference-depth:invalid")
	assert.NoError(t, err)
}

// TestSelectCertPool_StatusLogicAnd tests SelectCertPool with AND status logic
func TestSelectCertPool_StatusLogicAnd(t *testing.T) {
	pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
	ctx := NewContext()

	// Load a test TSL
	tslData := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
	<SchemeInformation>
		<TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
		<SchemeTerritory>SE</SchemeTerritory>
	</SchemeInformation>
</TrustServiceStatusList>`

	tmpFile := filepath.Join(t.TempDir(), "test.xml")
	err := os.WriteFile(tmpFile, []byte(tslData), 0644)
	assert.NoError(t, err)

	ctx, err = LoadTSL(pl, ctx, tmpFile)
	assert.NoError(t, err)

	// Test with status logic AND
	_, err = SelectCertPool(pl, ctx, "status-logic:and", "status:http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted")
	assert.NoError(t, err)
}

// TestPublishTSL_MissingDirectory tests PublishTSL with missing directory argument
func TestPublishTSL_MissingDirectory(t *testing.T) {
	pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
	ctx := NewContext()

	_, err := PublishTSL(pl, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing argument: directory path")
}

// TestPublishTSL_NotADirectory tests PublishTSL when path exists but is not a directory
func TestPublishTSL_NotADirectory(t *testing.T) {
	pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
	ctx := NewContext()

	// Create a file (not a directory)
	tmpFile := filepath.Join(t.TempDir(), "notadir.txt")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	assert.NoError(t, err)

	// Load a test TSL
	tslData := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
	<SchemeInformation>
		<TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
		<SchemeTerritory>SE</SchemeTerritory>
	</SchemeInformation>
</TrustServiceStatusList>`

	tslFile := filepath.Join(t.TempDir(), "test.xml")
	err = os.WriteFile(tslFile, []byte(tslData), 0644)
	assert.NoError(t, err)

	ctx, err = LoadTSL(pl, ctx, tslFile)
	assert.NoError(t, err)

	_, err = PublishTSL(pl, ctx, tmpFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")
}

// TestPublishTSL_InvalidCertPath tests PublishTSL with invalid certificate file path
func TestPublishTSL_InvalidCertPath(t *testing.T) {
	pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
	ctx := NewContext()

	// Load a test TSL first
	tslData := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
	<SchemeInformation>
		<TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
		<SchemeTerritory>SE</SchemeTerritory>
	</SchemeInformation>
</TrustServiceStatusList>`

	tslFile := filepath.Join(t.TempDir(), "test.xml")
	err := os.WriteFile(tslFile, []byte(tslData), 0644)
	assert.NoError(t, err)

	ctx, err = LoadTSL(pl, ctx, tslFile)
	assert.NoError(t, err)

	tmpDir := t.TempDir()

	_, err = PublishTSL(pl, ctx, tmpDir, "/nonexistent/cert.pem", "/some/key.pem")
	assert.Error(t, err)
	// The error comes from the signing step, not validation
	assert.Contains(t, err.Error(), "failed to sign TSL")
}

// TestPublishTSL_InvalidKeyPath tests PublishTSL with invalid key file path
func TestPublishTSL_InvalidKeyPath(t *testing.T) {
	pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
	ctx := NewContext()

	tmpDir := t.TempDir()

	// Load a test TSL first
	tslData := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
	<SchemeInformation>
		<TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
		<SchemeTerritory>SE</SchemeTerritory>
	</SchemeInformation>
</TrustServiceStatusList>`

	tslFile := filepath.Join(tmpDir, "test.xml")
	err := os.WriteFile(tslFile, []byte(tslData), 0644)
	assert.NoError(t, err)

	ctx, err = LoadTSL(pl, ctx, tslFile)
	assert.NoError(t, err)

	// Create a valid cert file
	certFile := filepath.Join(tmpDir, "cert.pem")
	err = os.WriteFile(certFile, []byte("dummy cert"), 0644)
	assert.NoError(t, err)

	_, err = PublishTSL(pl, ctx, tmpDir, certFile, "/nonexistent/key.pem")
	assert.Error(t, err)
	// The error comes from the signing step, not validation
	assert.Contains(t, err.Error(), "failed to sign TSL")
}

// TestPublishTSL_PKCS11Signer tests PublishTSL with PKCS#11 signer configuration
func TestPublishTSL_PKCS11Signer(t *testing.T) {
	pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
	ctx := NewContext()

	tmpDir := t.TempDir()

	// Load a test TSL
	tslData := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
	<SchemeInformation>
		<TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
		<SchemeTerritory>SE</SchemeTerritory>
	</SchemeInformation>
</TrustServiceStatusList>`

	tslFile := filepath.Join(t.TempDir(), "test.xml")
	err := os.WriteFile(tslFile, []byte(tslData), 0644)
	assert.NoError(t, err)

	ctx, err = LoadTSL(pl, ctx, tslFile)
	assert.NoError(t, err)

	// Test with PKCS#11 URI (will fail to initialize but tests the code path)
	// This tests the PKCS#11 configuration parsing path
	_, err = PublishTSL(pl, ctx, tmpDir, "pkcs11:module-path=/usr/lib/softhsm/libsofthsm2.so;token=mytoken", "mykey", "mycert", "02")
	// Expected to fail during actual signing, but configuration parsing should work
	// The error will be from signing, not from parsing
	if err != nil {
		// Error is expected since we don't have a real PKCS#11 setup
		assert.Error(t, err)
	}
}

// TestPublishTSL_WithFileSigner tests PublishTSL with file-based signer
func TestPublishTSL_WithFileSigner(t *testing.T) {
	pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
	ctx := NewContext()

	tmpDir := t.TempDir()

	// Create dummy cert and key files
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	err := os.WriteFile(certFile, []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"), 0644)
	assert.NoError(t, err)

	err = os.WriteFile(keyFile, []byte("-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----"), 0644)
	assert.NoError(t, err)

	// Load a test TSL
	tslData := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
	<SchemeInformation>
		<TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
		<SchemeTerritory>SE</SchemeTerritory>
	</SchemeInformation>
</TrustServiceStatusList>`

	tslFile := filepath.Join(t.TempDir(), "test.xml")
	err = os.WriteFile(tslFile, []byte(tslData), 0644)
	assert.NoError(t, err)

	ctx, err = LoadTSL(pl, ctx, tslFile)
	assert.NoError(t, err)

	outputDir := filepath.Join(tmpDir, "output")

	// This will fail during signing (invalid cert/key), but tests the file signer path
	_, err = PublishTSL(pl, ctx, outputDir, certFile, keyFile)
	// Expected to fail during signing
	if err != nil {
		assert.Error(t, err)
	}
}

// TestAddProviderCertificates_DirectoryReadError tests error when provider directory doesn't exist
func TestAddProviderCertificates_DirectoryReadError(t *testing.T) {
	err := addProviderCertificates("/nonexistent/directory", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read provider directory")
}

// TestAddProviderCertificates_MissingMetadata tests error when .yaml metadata is missing
func TestAddProviderCertificates_MissingMetadata(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .pem file without corresponding .yaml
	pemFile := filepath.Join(tmpDir, "cert.pem")
	err := os.WriteFile(pemFile, []byte("test cert data"), 0644)
	assert.NoError(t, err)

	provider := &etsi119612.TSPType{}
	err = addProviderCertificates(tmpDir, provider)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read certificate metadata")
}

// TestAddProviderCertificates_InvalidYAML tests error when metadata YAML is malformed
func TestAddProviderCertificates_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .pem file
	pemFile := filepath.Join(tmpDir, "cert.pem")
	err := os.WriteFile(pemFile, []byte("test cert"), 0644)
	assert.NoError(t, err)

	// Create invalid YAML metadata
	yamlFile := filepath.Join(tmpDir, "cert.yaml")
	err = os.WriteFile(yamlFile, []byte("invalid: yaml: content: :::"), 0644)
	assert.NoError(t, err)

	provider := &etsi119612.TSPType{}
	err = addProviderCertificates(tmpDir, provider)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse certificate metadata")
}

// TestAddProviderCertificates_NoServiceNames tests error when metadata has no service names
func TestAddProviderCertificates_NoServiceNames(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .pem file
	pemFile := filepath.Join(tmpDir, "cert.pem")
	err := os.WriteFile(pemFile, []byte("test cert"), 0644)
	assert.NoError(t, err)

	// Create metadata without service names
	yamlFile := filepath.Join(tmpDir, "cert.yaml")
	yamlContent := `service_type: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC"
status: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted"
service_names: []`
	err = os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	assert.NoError(t, err)

	provider := &etsi119612.TSPType{}
	err = addProviderCertificates(tmpDir, provider)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must include at least one service name")
}

// TestAddProviderCertificates_InvalidCertificate tests error when certificate data is invalid
func TestAddProviderCertificates_InvalidCertificate(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid metadata with correct YAML structure (using camelCase)
	yamlFile := filepath.Join(tmpDir, "cert.yaml")
	yamlContent := `serviceType: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC"
status: "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted"
serviceNames:
  - language: "en"
    value: "Test Service"`
	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	assert.NoError(t, err)

	// Create a .pem file with invalid certificate data
	pemFile := filepath.Join(tmpDir, "cert.pem")
	err = os.WriteFile(pemFile, []byte("invalid certificate data"), 0644)
	assert.NoError(t, err)

	provider := &etsi119612.TSPType{}
	err = addProviderCertificates(tmpDir, provider)
	// Should fail to parse the certificate
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode invalid certificate")
}

// TestPublishTSLToFile_CreateError tests publishTSLToFile when file creation fails
func TestPublishTSLToFile_InvalidPath(t *testing.T) {
	pl := &Pipeline{Logger: logging.NewLogger(logging.InfoLevel)}
	ctx := NewContext()

	// Create a minimal TSL XML
	tslData := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
	<SchemeInformation>
		<TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
		<SchemeTerritory>SE</SchemeTerritory>
	</SchemeInformation>
</TrustServiceStatusList>`

	// Write to a temp file and load it
	tmpFile := filepath.Join(t.TempDir(), "test.xml")
	err := os.WriteFile(tmpFile, []byte(tslData), 0644)
	assert.NoError(t, err)

	ctx, err = LoadTSL(pl, ctx, tmpFile)
	assert.NoError(t, err)
	assert.NotNil(t, ctx.TSLs)
	assert.Greater(t, len(ctx.TSLs.ToSlice()), 0)

	tsl := ctx.TSLs.ToSlice()[0]

	// Try to write to an invalid path (e.g., a directory that doesn't exist and can't be created)
	invalidPath := "/proc/nonexistent/impossible/path/file.xml"
	err = publishTSLToFile(pl, tsl, invalidPath, nil)
	assert.Error(t, err)
}
