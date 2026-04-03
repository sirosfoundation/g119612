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
	assert.Contains(t, string(content1), "<TrustServiceStatusList", "File should contain XML structure")

	content2, err := os.ReadFile(expectedFile2)
	assert.NoError(t, err)
	assert.NotEmpty(t, content2, "File content should not be empty")
	assert.Contains(t, string(content2), "<TrustServiceStatusList", "File should contain XML structure")
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

// TestPublishTSLToFile tests the publishTSLToFile function directly
func TestPublishTSLToFile(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-publish-file-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	tsl.StatusList.TSLTagAttr = "http://uri.etsi.org/19612/TSLTag"
	tsl.StatusList.IdAttr = "TEST-001"

	filePath := filepath.Join(outputDir, "test.xml")
	err = publishTSLToFile(pl, tsl, filePath, nil)
	assert.NoError(t, err)

	// Verify file was created
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), `xmlns="http://uri.etsi.org/02231/v2#"`)
	assert.Contains(t, string(content), `TSLTag="http://uri.etsi.org/19612/TSLTag"`)
	assert.Contains(t, string(content), `Id="TEST-001"`)
}

// TestPublishTSLToFile_NilTSL tests publishing a nil TSL
func TestPublishTSLToFile_NilTSL(t *testing.T) {
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	err := publishTSLToFile(pl, nil, "/tmp/test.xml", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot publish nil TSL")
}

// TestProcessTreeForPublishing_NilTree tests processTreeForPublishing with nil tree
func TestProcessTreeForPublishing_NilTree(t *testing.T) {
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}
	ctx := &Context{}

	// Nil tree should return nil error
	err := processTreeForPublishing(pl, ctx, nil, "/tmp", 0, "territory", nil)
	assert.NoError(t, err)

	// Tree with nil root should return nil error
	tree := &TSLTree{Root: nil}
	err = processTreeForPublishing(pl, ctx, tree, "/tmp", 0, "territory", nil)
	assert.NoError(t, err)
}

// TestPublishStep_LegacyStackWithDistributionPoints tests legacy stack publishing
func TestPublishStep_LegacyStackWithDistributionPoints(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-legacy-dp-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create TSL with distribution point
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	tsl.StatusList.TslSchemeInformation.TslDistributionPoints = &etsi119612.NonEmptyURIListType{
		URI: []string{"https://example.com/my-custom-tsl.xml"},
	}

	ctx := &Context{}
	ctx.EnsureTSLStack().TSLs.Push(tsl)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	_, err = PublishTSL(pl, ctx, outputDir)
	assert.NoError(t, err)

	// Verify file was created with distribution point name
	expectedFile := filepath.Join(outputDir, "my-custom-tsl.xml")
	_, err = os.Stat(expectedFile)
	assert.NoError(t, err, "File should be created with distribution point name")
}

// TestPublishStep_TreeWithIndexFormat tests tree publishing with index format
func TestPublishStep_TreeWithIndexFormat(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-tree-index-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create a TSL
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	tsl.StatusList.TslSchemeInformation.TslSchemeTerritory = "SE"

	// Create context with tree
	ctx := &Context{}
	tree := FromSlice([]*etsi119612.TSL{tsl})
	ctx.EnsureTSLTrees()
	ctx.TSLTrees.Push(tree)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test with index-based tree format
	_, err = PublishTSL(pl, ctx, outputDir, "tree:index")
	assert.NoError(t, err)

	// Verify tree-0 directory was created
	treeDir := filepath.Join(outputDir, "tree-0")
	info, err := os.Stat(treeDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestPublishStep_TreeWithoutTerritory tests tree publishing without territory
func TestPublishStep_TreeWithoutTerritory(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-tree-no-territory-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create a TSL without territory
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	tsl.StatusList.TslSchemeInformation.TslSchemeTerritory = ""

	// Create context with tree
	ctx := &Context{}
	tree := FromSlice([]*etsi119612.TSL{tsl})
	ctx.EnsureTSLTrees()
	ctx.TSLTrees.Push(tree)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test with territory format but no territory - should fall back to index
	_, err = PublishTSL(pl, ctx, outputDir, "tree:territory")
	assert.NoError(t, err)

	// Verify tree-0 directory was created (fallback)
	treeDir := filepath.Join(outputDir, "tree-0")
	info, err := os.Stat(treeDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestPublishStep_TreeFlatMode tests tree publishing without tree format (flat mode)
func TestPublishStep_TreeFlatMode(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-tree-flat-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create a TSL
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	tsl.StatusList.TslSchemeInformation.TslSchemeTerritory = "SE"

	// Create context with tree (not legacy stack)
	ctx := &Context{}
	tree := FromSlice([]*etsi119612.TSL{tsl})
	ctx.EnsureTSLTrees()
	ctx.TSLTrees.Push(tree)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Publish without tree format - should use flat mode
	_, err = PublishTSL(pl, ctx, outputDir)
	assert.NoError(t, err)

	// Verify files are in the root directory, not in subdirectories
	files, err := os.ReadDir(outputDir)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1)

	// Verify no SE subdirectory was created
	_, err = os.Stat(filepath.Join(outputDir, "SE"))
	assert.True(t, os.IsNotExist(err), "Should not create territory subdirectory in flat mode")
}

// TestPublishStep_MultipleTSLsInLegacyStack tests publishing multiple TSLs from legacy stack
func TestPublishStep_MultipleTSLsInLegacyStack(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-multi-tsl-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create multiple TSLs
	tsl1 := generateTSL("Service 1", "http://test.com/type", []string{TestCertBase64})
	tsl2 := generateTSL("Service 2", "http://test.com/type", []string{TestCertBase64})
	tsl3 := generateTSL("Service 3", "http://test.com/type", []string{TestCertBase64})

	ctx := &Context{}
	ctx.EnsureTSLStack().TSLs.Push(tsl1)
	ctx.EnsureTSLStack().TSLs.Push(tsl2)
	ctx.EnsureTSLStack().TSLs.Push(tsl3)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	_, err = PublishTSL(pl, ctx, outputDir)
	assert.NoError(t, err)

	// Verify 3 files were created
	files, err := os.ReadDir(outputDir)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(files), "Should create 3 files for 3 TSLs")
}

// TestPublishStep_EmptyDistributionPointURI tests handling of empty distribution point URI
func TestPublishStep_EmptyDistributionPointURI(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-empty-dp-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create TSL with empty distribution point URI
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	tsl.StatusList.TslSchemeInformation.TslDistributionPoints = &etsi119612.NonEmptyURIListType{
		URI: []string{""},
	}

	ctx := &Context{}
	ctx.EnsureTSLStack().TSLs.Push(tsl)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	_, err = PublishTSL(pl, ctx, outputDir)
	assert.NoError(t, err)

	// Should fall back to default naming
	files, err := os.ReadDir(outputDir)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, "tsl-0.xml", files[0].Name())
}

// TestPublishStep_InvalidCertificatePath tests validation of certificate path
func TestPublishStep_InvalidCertificatePath(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-invalid-cert-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	ctx := &Context{}
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	ctx.EnsureTSLStack().TSLs.Push(tsl)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test with path traversal in certificate path
	_, err = PublishTSL(pl, ctx, outputDir, "../../../etc/passwd", "/tmp/key.pem")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid certificate path")
}

// TestPublishStep_InvalidKeyPath tests validation of key path
func TestPublishStep_InvalidKeyPath(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-invalid-key-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	ctx := &Context{}
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	ctx.EnsureTSLStack().TSLs.Push(tsl)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test with path traversal in key path
	_, err = PublishTSL(pl, ctx, outputDir, "/tmp/cert.pem", "../../../etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key path")
}

// TestPublishStep_PKCS11Signer tests PKCS11 signer configuration
func TestPublishStep_PKCS11Signer(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-pkcs11-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	ctx := &Context{}
	ctx.Data = make(map[string]interface{})
	ctx.Data["test"] = "pkcs11"
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	ctx.EnsureTSLStack().TSLs.Push(tsl)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test with PKCS11 configuration - will fail because no HSM available
	// but exercises the PKCS11 config parsing code path
	_, err = PublishTSL(pl, ctx, outputDir, "pkcs11:module=/usr/lib/softhsm/libsofthsm2.so;token=test", "key-label", "cert-label", "01")
	// This will fail because no HSM is available, but it exercises the code path
	assert.Error(t, err)
}

// TestPublishStep_PKCS11SignerMinimalArgs tests PKCS11 signer with minimal arguments
func TestPublishStep_PKCS11SignerMinimalArgs(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-pkcs11-minimal-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	ctx := &Context{}
	ctx.Data = make(map[string]interface{})
	ctx.Data["test"] = "pkcs11"
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	ctx.EnsureTSLStack().TSLs.Push(tsl)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test PKCS11 with only module path - exercises default label handling
	_, err = PublishTSL(pl, ctx, outputDir, "pkcs11:module=/usr/lib/softhsm/libsofthsm2.so;token=test")
	assert.Error(t, err) // Will fail due to no HSM, but exercises code path
}

// TestPublishStep_TreeWithNonTreeArg tests tree publishing with non-tree second argument
func TestPublishStep_TreeWithNonTreeArg(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-non-tree-arg-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create a TSL
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	tsl.StatusList.TslSchemeInformation.TslSchemeTerritory = "SE"

	// Create context with tree (not legacy stack)
	ctx := &Context{}
	tree := FromSlice([]*etsi119612.TSL{tsl})
	ctx.EnsureTSLTrees()
	ctx.TSLTrees.Push(tree)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Second argument is not a tree format - should use flat mode
	_, err = PublishTSL(pl, ctx, outputDir, "some-other-arg")
	assert.NoError(t, err)

	// Files should be in root directory
	files, err := os.ReadDir(outputDir)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1)
}

// TestPublishStep_TreeEmptyFormat tests tree publishing with empty tree format
func TestPublishStep_TreeEmptyFormat(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-empty-format-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create a TSL
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	tsl.StatusList.TslSchemeInformation.TslSchemeTerritory = "SE"

	// Create context with tree
	ctx := &Context{}
	tree := FromSlice([]*etsi119612.TSL{tsl})
	ctx.EnsureTSLTrees()
	ctx.TSLTrees.Push(tree)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Empty format after tree: should default to territory
	_, err = PublishTSL(pl, ctx, outputDir, "tree:")
	assert.NoError(t, err)

	// SE directory should be created (default is territory)
	info, err := os.Stat(filepath.Join(outputDir, "SE"))
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TestPublishStep_TreeInvalidFormat tests tree publishing with invalid tree format
func TestPublishStep_TreeInvalidFormat(t *testing.T) {
	outputDir, err := os.MkdirTemp("", "test-invalid-format-*")
	assert.NoError(t, err)
	defer os.RemoveAll(outputDir)

	// Create a TSL
	tsl := generateTSL("Test Service", "http://test.com/type", []string{TestCertBase64})
	tsl.StatusList.TslSchemeInformation.TslSchemeTerritory = "SE"

	// Create context with tree
	ctx := &Context{}
	tree := FromSlice([]*etsi119612.TSL{tsl})
	ctx.EnsureTSLTrees()
	ctx.TSLTrees.Push(tree)

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Invalid format should default to territory
	_, err = PublishTSL(pl, ctx, outputDir, "tree:invalid-format")
	assert.NoError(t, err)

	// SE directory should be created (default is territory)
	info, err := os.Stat(filepath.Join(outputDir, "SE"))
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}
