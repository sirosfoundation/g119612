package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestPublishStep(t *testing.T) {
	// Create a temporary directory for testing
	testDir, err := os.MkdirTemp("", "test-publish-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a test TSL with a distribution point
	tsl1 := generateTSL("Test Service 1", "http://uri.etsi.org/TrstSvc/Svctype/CA/QC", []string{TestCertBase64})
	tsl1.StatusList.TslSchemeInformation.TslDistributionPoints = &etsi119612.NonEmptyURIListType{
		URI: []string{"https://example.com/test-tsl.xml"},
	}

	// Create another test TSL without a distribution point
	tsl2 := generateTSL("Test Service 2", "http://uri.etsi.org/TrstSvc/Svctype/CA/QC", []string{TestCertBase64})

	// Set up the context with the test TSLs
	ctx := &Context{}
	ctx.EnsureTSLStack().TSLs.Push(tsl2) // Note: LIFO order, tsl2 will be processed first
	ctx.EnsureTSLStack().TSLs.Push(tsl1)

	// Test the publish step
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}
	_, err = PublishTSL(pl, ctx, testDir)
	assert.NoError(t, err)

	// Check that the files were created
	fileInfos, err := os.ReadDir(testDir)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(fileInfos), "Expected two files to be created")

	// List the files in the directory for debugging
	fileInfos, err = os.ReadDir(testDir)
	assert.NoError(t, err)
	t.Logf("Files in directory: %d", len(fileInfos))
	for _, fi := range fileInfos {
		t.Logf("  - %s", fi.Name())
	}

	// Check that the file with the specific name exists (from distribution point)
	expectedFile1 := filepath.Join(testDir, "test-tsl.xml")
	_, err = os.Stat(expectedFile1)
	assert.NoError(t, err, "Expected file test-tsl.xml to exist")

	// Check that the default named file exists
	expectedFile2 := filepath.Join(testDir, "tsl-0.xml") // Changed from tsl-1.xml to tsl-0.xml
	_, err = os.Stat(expectedFile2)
	assert.NoError(t, err, "Expected file tsl-0.xml to exist")

	// Verify that the files have content
	content1, err := os.ReadFile(expectedFile1)
	assert.NoError(t, err)
	assert.NotEmpty(t, content1, "File content should not be empty")
	assert.Contains(t, string(content1), "<TrustServiceStatusList>", "File should contain XML structure")

	content2, err := os.ReadFile(expectedFile2)
	assert.NoError(t, err)
	assert.NotEmpty(t, content2, "File content should not be empty")
	assert.Contains(t, string(content2), "<TrustServiceStatusList>", "File should contain XML structure")
}

func TestPublishStep_Errors(t *testing.T) {
	// Test case 1: Missing directory argument
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}
	ctx := &Context{}
	_, err := PublishTSL(pl, ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing argument: directory path")

	// Test case 2: Invalid directory path (file exists with the same name)
	tmpfile, err := os.CreateTemp("", "not-a-directory-*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	_, err = PublishTSL(pl, ctx, tmpfile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")

	// Test case 3: No TSLs to publish
	tmpdir, err := os.MkdirTemp("", "empty-tsl-dir-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	_, err = PublishTSL(pl, ctx, tmpdir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no TSLs to publish")

	// Test case 4: Nil TSL in the stack
	ctx.EnsureTSLStack().TSLs.Push(nil)
	_, err = PublishTSL(pl, ctx, tmpdir)
	assert.NoError(t, err, "Should handle nil TSLs gracefully")

	// No files should be created in the directory
	fileInfos, err := os.ReadDir(tmpdir)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(fileInfos), "No files should be created for nil TSL")
}
