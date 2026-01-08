package pipeline

import (
	"os"
	"path/filepath"
	"testing"

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
