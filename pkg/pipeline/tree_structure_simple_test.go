package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestSimpleTreeForPublishing(t *testing.T) {
	// Create a TSL tree for testing
	rootTSL := &etsi119612.TSL{
		Source: "root.xml",
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "ROOT",
				TslDistributionPoints: &etsi119612.NonEmptyURIListType{
					URI: []string{"https://example.com/root.xml"},
				},
			},
		},
	}

	// Create tree
	tree := NewTSLTree(rootTSL)

	// Create a temporary output directory
	tempDir, err := os.MkdirTemp("", "simple-tree-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Logf("Created temporary directory: %s", tempDir)

	// Create a pipeline instance
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Try to process the tree directly
	err = processTreeForPublishing(pl, nil, tree, tempDir, 0, "territory", nil)
	assert.NoError(t, err)

	// Check if the ROOT directory was created
	rootDir := filepath.Join(tempDir, "ROOT")
	info, err := os.Stat(rootDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())

	// Check for index file
	indexPath := filepath.Join(rootDir, "index.txt")
	_, err = os.Stat(indexPath)
	assert.NoError(t, err)

	// Read and verify the index content
	content, err := os.ReadFile(indexPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "TSL Tree Structure")
	assert.Contains(t, string(content), "ROOT")

	// List all files in the directory structure
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		t.Logf("File: %s (isDir: %v)", path, info.IsDir())
		return nil
	})
	assert.NoError(t, err)
}
