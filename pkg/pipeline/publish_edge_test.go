package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPublishStep_TreeStructure tests publishing with tree structure support
func TestPublishStep_TreeStructure(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-publish-tree-*")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Create test TSL with territory
	tsl := generateTSL("Test Service", "http://uri.etsi.org/TrstSvc/Svctype/CA/QC", []string{TestCertBase64})
	if tsl.StatusList.TslSchemeInformation == nil {
		tsl.StatusList.TslSchemeInformation = &etsi119612.TSLSchemeInformationType{}
	}
	tsl.StatusList.TslSchemeInformation.TslSchemeTerritory = "SE"

	// Set up context with tree structure
	ctx := &Context{}
	tree := NewTSLTree(tsl)
	ctx.EnsureTSLTrees().AddTSLTree(tree)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test publish with tree structure - note: tree:territory will create subdirectories
	_, err = PublishTSL(pl, ctx, testDir, "tree:territory")
	assert.NoError(t, err)

	// Verify at least one file was created (specific subdirectory structure depends on implementation)
	fileInfos, err := os.ReadDir(testDir)
	assert.NoError(t, err)
	assert.NotEmpty(t, fileInfos, "Expected files/directories to be created")
}

// TestPublishStep_SigningError tests error handling during signing
func TestPublishStep_SigningError(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-publish-sign-*")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	tsl := generateTSL("Test Service", "http://uri.etsi.org/TrstSvc/Svctype/CA/QC", []string{TestCertBase64})
	ctx := &Context{}
	ctx.EnsureTSLStack().TSLs.Push(tsl)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test with invalid certificate path - will fail during signing
	_, err = PublishTSL(pl, ctx, testDir, "/nonexistent/cert.pem", "/nonexistent/key.pem")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sign TSL")
}

// TestPublishStep_DirectoryCreation tests automatic directory creation
func TestPublishStep_DirectoryCreation(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "test-publish-create", "nested", "dirs")
	defer os.RemoveAll(filepath.Join(os.TempDir(), "test-publish-create"))

	// Ensure directory doesn't exist
	os.RemoveAll(filepath.Join(os.TempDir(), "test-publish-create"))

	tsl := generateTSL("Test Service", "http://uri.etsi.org/TrstSvc/Svctype/CA/QC", []string{TestCertBase64})
	ctx := &Context{}
	ctx.EnsureTSLStack().TSLs.Push(tsl)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test automatic directory creation
	_, err := PublishTSL(pl, ctx, testDir)
	assert.NoError(t, err)

	// Verify directory was created
	info, err := os.Stat(testDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestPublishStep_InvalidOutputDirectory tests validation of output directory
func TestPublishStep_InvalidOutputDirectory(t *testing.T) {
	tsl := generateTSL("Test Service", "http://uri.etsi.org/TrstSvc/Svctype/CA/QC", []string{TestCertBase64})
	ctx := &Context{}
	ctx.EnsureTSLStack().TSLs.Push(tsl)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test with various invalid directory paths
	testCases := []struct {
		name string
		path string
	}{
		{"null byte in path", "test\x00dir"},
		{"path traversal", "../../../etc/passwd"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := PublishTSL(pl, ctx, tc.path)
			assert.Error(t, err, "Should reject invalid directory path: %s", tc.path)
		})
	}
}
