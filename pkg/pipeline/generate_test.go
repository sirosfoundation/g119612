package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTSL_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() (string, error)
		expectError string
	}{
		{
			name: "missing root directory",
			setupFunc: func() (string, error) {
				return "/nonexistent/directory", nil
			},
			expectError: "failed to read providers directory",
		},
		{
			name: "missing scheme.yaml",
			setupFunc: func() (string, error) {
				dir, err := os.MkdirTemp("", "tsl-test-*")
				if err != nil {
					return "", err
				}
				if err := os.MkdirAll(filepath.Join(dir, "providers"), 0755); err != nil {
					return "", err
				}
				return dir, nil
			},
			expectError: "failed to read scheme metadata",
		},
		{
			name: "invalid scheme.yaml",
			setupFunc: func() (string, error) {
				dir, err := os.MkdirTemp("", "tsl-test-*")
				if err != nil {
					return "", err
				}
				if err := os.MkdirAll(filepath.Join(dir, "providers"), 0755); err != nil {
					return "", err
				}
				err = os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte("invalid_yaml: ["), 0644)
				if err != nil {
					return "", err
				}
				return dir, nil
			},
			expectError: "failed to parse scheme metadata",
		},
		{
			name: "empty scheme.yaml",
			setupFunc: func() (string, error) {
				dir, err := os.MkdirTemp("", "tsl-test-*")
				if err != nil {
					return "", err
				}
				if err := os.MkdirAll(filepath.Join(dir, "providers"), 0755); err != nil {
					return "", err
				}
				err = os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte("# Empty file"), 0644)
				if err != nil {
					return "", err
				}
				return dir, nil
			},
			expectError: "scheme metadata must include at least one operator name",
		},
		{
			name: "invalid provider metadata",
			setupFunc: func() (string, error) {
				dir, err := os.MkdirTemp("", "tsl-test-*")
				if err != nil {
					return "", err
				}
				if err := os.MkdirAll(filepath.Join(dir, "providers", "test_provider"), 0755); err != nil {
					return "", err
				}
				// Write valid scheme.yaml
				schemeYAML := "operatorNames:\n  - language: en\n    value: \"Test Operator\"\ntype: \"http://test.example.com/tsl-type\""
				if err := os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte(schemeYAML), 0644); err != nil {
					return "", err
				}
				// Write invalid provider.yaml
				err = os.WriteFile(filepath.Join(dir, "providers", "test_provider", "provider.yaml"), []byte("invalid_yaml: ["), 0644)
				if err != nil {
					return "", err
				}
				return dir, nil
			},
			expectError: "failed to parse provider metadata",
		},
		{
			name: "malformed certificate data",
			setupFunc: func() (string, error) {
				dir, err := os.MkdirTemp("", "tsl-test-*")
				if err != nil {
					return "", err
				}
				if err := os.MkdirAll(filepath.Join(dir, "providers", "test_provider"), 0755); err != nil {
					return "", err
				}

				// Write valid scheme.yaml
				schemeYAML := "operatorNames:\n  - language: en\n    value: \"Test Operator\"\ntype: \"http://test.example.com/tsl-type\""
				if err := os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte(schemeYAML), 0644); err != nil {
					return "", err
				}

				// Write provider.yaml with valid metadata
				providerYAML := `names:
  - language: en
    value: "Test Provider"
`
				if err := os.WriteFile(filepath.Join(dir, "providers", "test_provider", "provider.yaml"), []byte(providerYAML), 0644); err != nil {
					return "", err
				}

				// Write invalid certificate and metadata files
				certContent := []byte("INVALID_CERTIFICATE_DATA") // Not PEM formatted
				if err := os.WriteFile(filepath.Join(dir, "providers", "test_provider", "cert1.pem"), certContent, 0644); err != nil {
					return "", err
				}

				certMetadata := `serviceNames:
  - language: en
    value: "Test Service"
serviceType: "http://test.example.com/service-type"
status: "http://test.example.com/status/valid"
`
				if err := os.WriteFile(filepath.Join(dir, "providers", "test_provider", "cert1.yaml"), []byte(certMetadata), 0644); err != nil {
					return "", err
				}

				return dir, nil
			},
			expectError: "failed to decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, setupErr := tt.setupFunc()
			if setupErr != nil {
				t.Fatalf("Setup failed: %v", setupErr)
			}
			defer os.RemoveAll(testDir)

			// Run the GenerateTSL step
			ctx := NewContext()
			var err error

			_, err = GenerateTSL(nil, ctx, testDir)
			if err == nil {
				t.Errorf("Expected error for case: %s", tt.name)
			} else {
				assert.Contains(t, err.Error(), tt.expectError, "Expected error containing '%s', got '%s' for case: %s", tt.expectError, err.Error(), tt.name)
			}
		})
	}
}

// TestGenerateTSL_Success tests successful TSL generation with proper attributes
func TestGenerateTSL_Success(t *testing.T) {
	// Create a valid test directory structure
	dir, err := os.MkdirTemp("", "tsl-success-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	// Create providers directory
	err = os.MkdirAll(filepath.Join(dir, "providers", "test_provider"), 0755)
	assert.NoError(t, err)

	// Write valid scheme.yaml
	schemeYAML := `operatorNames:
  - language: en
    value: "Test Operator"
  - language: sv
    value: "Test Operatör"
type: "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric"
sequenceNumber: 1
`
	err = os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte(schemeYAML), 0644)
	assert.NoError(t, err)

	// Write valid provider.yaml
	providerYAML := `names:
  - language: en
    value: "Test Provider Inc"
`
	err = os.WriteFile(filepath.Join(dir, "providers", "test_provider", "provider.yaml"), []byte(providerYAML), 0644)
	assert.NoError(t, err)

	// Run the GenerateTSL step
	ctx := NewContext()
	ctx, err = GenerateTSL(nil, ctx, dir)
	assert.NoError(t, err)

	// Verify the TSL was created
	assert.NotNil(t, ctx.TSLs)
	assert.False(t, ctx.TSLs.IsEmpty())

	// Get the generated TSL
	tsl, ok := ctx.TSLs.Peek()
	assert.True(t, ok, "Should be able to peek TSL from stack")
	assert.NotNil(t, tsl)

	// Verify TSLTag and Id attributes are set
	assert.Equal(t, "http://uri.etsi.org/19612/TSLTag", tsl.StatusList.TSLTagAttr, "TSLTag should be set")
	assert.Equal(t, "TSL-001", tsl.StatusList.IdAttr, "Id should default to TSL-001 for sequenceNumber 1")

	// Verify scheme information
	assert.NotNil(t, tsl.StatusList.TslSchemeInformation)
	assert.Equal(t, "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric", tsl.StatusList.TslSchemeInformation.TslTSLType)
	assert.Equal(t, 1, tsl.StatusList.TslSchemeInformation.TSLVersionIdentifier)
}

// TestGenerateTSL_CustomId tests that a custom id from scheme.yaml is used
func TestGenerateTSL_CustomId(t *testing.T) {
	dir, err := os.MkdirTemp("", "tsl-custom-id-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	err = os.MkdirAll(filepath.Join(dir, "providers", "test_provider"), 0755)
	assert.NoError(t, err)

	schemeYAML := `operatorNames:
  - language: en
    value: "Test Operator"
type: "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric"
sequenceNumber: 42
id: "MY-CUSTOM-TSL"
`
	err = os.WriteFile(filepath.Join(dir, "scheme.yaml"), []byte(schemeYAML), 0644)
	assert.NoError(t, err)

	providerYAML := `names:
  - language: en
    value: "Test Provider"
`
	err = os.WriteFile(filepath.Join(dir, "providers", "test_provider", "provider.yaml"), []byte(providerYAML), 0644)
	assert.NoError(t, err)

	ctx := NewContext()
	ctx, err = GenerateTSL(nil, ctx, dir)
	assert.NoError(t, err)

	tsl, ok := ctx.TSLs.Peek()
	assert.True(t, ok)
	assert.Equal(t, "MY-CUSTOM-TSL", tsl.StatusList.IdAttr, "Custom id from scheme.yaml should be used")
}

// TestPublishTSL_XMLNamespacesAndAttributes verifies the generated XML has correct namespaces and attributes
func TestPublishTSL_XMLNamespacesAndAttributes(t *testing.T) {
	// Create a temporary output directory
	outputDir, err := os.MkdirTemp("", "tsl-publish-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create a TSL with TSLTag and Id attributes
	tsl := generateTSL("Test Service", "http://uri.etsi.org/TrstSvc/Svctype/CA/QC", []string{TestCertBase64})
	tsl.StatusList.TSLTagAttr = "http://uri.etsi.org/19612/TSLTag"
	tsl.StatusList.IdAttr = "TSL-TEST-001"

	// Set up context
	ctx := &Context{}
	ctx.EnsureTSLStack().TSLs.Push(tsl)

	// Run publish
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}
	_, err = PublishTSL(pl, ctx, outputDir)
	assert.NoError(t, err)

	// Read the generated XML
	content, err := os.ReadFile(filepath.Join(outputDir, "tsl-0.xml"))
	assert.NoError(t, err)
	xmlStr := string(content)

	// Verify XML declaration
	assert.Contains(t, xmlStr, `<?xml version="1.0" encoding="UTF-8"?>`, "Should have XML declaration")

	// Verify namespaces
	assert.Contains(t, xmlStr, `xmlns="http://uri.etsi.org/02231/v2#"`, "Should have default ETSI TSL namespace")
	assert.Contains(t, xmlStr, `xmlns:ns2="http://www.w3.org/2000/09/xmldsig#"`, "Should have XML-DSIG namespace")
	assert.Contains(t, xmlStr, `xmlns:ns6="http://uri.etsi.org/01903/v1.4.1#"`, "Should have XAdES namespace")

	// Verify TSLTag attribute
	assert.Contains(t, xmlStr, `TSLTag="http://uri.etsi.org/19612/TSLTag"`, "Should have TSLTag attribute")

	// Verify Id attribute
	assert.Contains(t, xmlStr, `Id="TSL-TEST-001"`, "Should have Id attribute")

	// Verify root element structure
	assert.Contains(t, xmlStr, "<TrustServiceStatusList", "Should have TrustServiceStatusList root element")
	assert.Contains(t, xmlStr, "</TrustServiceStatusList>", "Should have closing TrustServiceStatusList tag")

	// Verify it's well-formed (basic check)
	assert.Contains(t, xmlStr, "<SchemeInformation>", "Should have SchemeInformation element")
	assert.Contains(t, xmlStr, "<TrustServiceProviderList>", "Should have TrustServiceProviderList element")
}
