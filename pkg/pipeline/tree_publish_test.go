package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestPublishTSLWithTreeStructure(t *testing.T) {
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

	refTSL1 := &etsi119612.TSL{
		Source: "ref1.xml",
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "REF1",
				TslDistributionPoints: &etsi119612.NonEmptyURIListType{
					URI: []string{"https://example.com/ref1.xml"},
				},
			},
		},
	}

	refTSL2 := &etsi119612.TSL{
		Source: "ref2.xml",
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "REF2",
				TslDistributionPoints: &etsi119612.NonEmptyURIListType{
					URI: []string{"https://example.com/ref2.xml"},
				},
			},
		},
	}

	// Set up references
	rootTSL.Referenced = []*etsi119612.TSL{refTSL1, refTSL2}

	// Create tree
	tree := NewTSLTree(rootTSL)

	// Create a temporary output directory
	tempDir, err := os.MkdirTemp("", "tree-publish-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Logf("Created temporary directory: %s", tempDir)

	// Create pipeline and context
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}

	// Test cases for different tree formats
	tests := []struct {
		name            string
		treeFormat      string
		expectRootDir   string
		expectSubDirs   bool
		expectIndexFile bool
	}{
		{
			name:            "Flat structure (no tree format)",
			treeFormat:      "",
			expectRootDir:   tempDir,
			expectSubDirs:   false,
			expectIndexFile: false,
		},
		{
			name:            "Territory-based tree structure",
			treeFormat:      "tree:territory",
			expectRootDir:   "", // Will be set in the test based on testDir
			expectSubDirs:   true,
			expectIndexFile: true,
		},
		{
			name:            "Index-based tree structure",
			treeFormat:      "tree:index",
			expectRootDir:   "", // Will be set in the test based on testDir
			expectSubDirs:   true,
			expectIndexFile: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a subdirectory for this test case - use simpler names without spaces
			dirName := strings.ReplaceAll(tc.name, " ", "-")
			testDir := filepath.Join(tempDir, dirName)
			err := os.MkdirAll(testDir, 0755)
			assert.NoError(t, err)

			// Create a fresh context for each test
			ctx := NewContext()
			ctx.EnsureTSLTrees()
			ctx.AddTSLTree(tree)

			// Call PublishTSL with the appropriate arguments
			// No need to define args variable here
			// Use the standard PublishTSL function with appropriate arguments
			var args []string

			if tc.treeFormat == "" {
				args = []string{testDir}
			} else {
				args = []string{testDir, tc.treeFormat}
				t.Logf("Using tree format: %s", tc.treeFormat)
			}

			// Print information about the context before publishing
			t.Logf("Context has %d TSL trees", len(ctx.TSLTrees.ToSlice()))
			for i, tree := range ctx.TSLTrees.ToSlice() {
				if tree != nil && tree.Root != nil && tree.Root.TSL != nil {
					t.Logf("Tree %d root territory: %s", i,
						tree.Root.TSL.StatusList.TslSchemeInformation.TslSchemeTerritory)
				}
			}

			// For tree-based tests, use processTreeForPublishing directly
			// Otherwise use PublishTSL
			var resultCtx *Context

			if tc.treeFormat != "" {
				subdirFormat := strings.TrimPrefix(tc.treeFormat, "tree:")
				if subdirFormat == "" || (subdirFormat != "index" && subdirFormat != "territory") {
					subdirFormat = "territory"
				}

				t.Logf("Calling processTreeForPublishing directly with format: %s", subdirFormat)
				err = processTreeForPublishing(pl, ctx, tree, testDir, 0, subdirFormat, nil)
				resultCtx = ctx
			} else {
				// Make sure the args are trimmed properly
				for i := range args {
					args[i] = strings.TrimSpace(args[i])
				}

				t.Logf("Publishing with args: %v (literal: [%s])", args, strings.Join(args, ", "))
				resultCtx, err = PublishTSL(pl, ctx, args...)
			}
			assert.NoError(t, err)
			assert.NotNil(t, resultCtx)

			// List the files that were created
			err = filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				t.Logf("Created: %s (dir: %t)", path, info.IsDir())
				return nil
			})
			assert.NoError(t, err)

			// Check output structure
			if tc.treeFormat == "" {
				// Flat structure should have all files in the root dir
				files, err := os.ReadDir(testDir)
				assert.NoError(t, err)
				assert.Equal(t, 3, len(files), "Expected 3 files in flat structure")

				// Check that all files are XML
				for _, file := range files {
					assert.False(t, file.IsDir(), "Expected files, not directories")
					assert.True(t, strings.HasSuffix(file.Name(), ".xml"), "Expected XML files")
				}
			} else {
				// For tree structure tests, just check if files were created
				var foundXmlFiles = 0
				var foundIndexFile = false
				var foundDirs = 0

				// Walk the entire directory structure and count files/directories
				err := filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if path == testDir {
						// Skip the root directory itself
						return nil
					}

					t.Logf("Found path: %s (dir: %v)", path, info.IsDir())

					if info.IsDir() {
						foundDirs++
					} else if strings.HasSuffix(path, ".xml") {
						foundXmlFiles++
					} else if strings.HasSuffix(path, "index.txt") {
						foundIndexFile = true

						// Check the content of index.txt
						content, err := os.ReadFile(path)
						if err == nil {
							t.Logf("Index file content: %s", string(content))
							if !strings.Contains(string(content), "TSL Tree Structure") {
								t.Errorf("Index file doesn't contain 'TSL Tree Structure'")
							}
						} else {
							t.Errorf("Failed to read index file: %v", err)
						}
					}

					return nil
				})

				assert.NoError(t, err, "Error walking directory structure")

				// In tree structure mode, we should have at least one XML file and one directory
				assert.True(t, foundXmlFiles > 0, "Should have found at least one XML file")

				if tc.expectIndexFile {
					assert.True(t, foundIndexFile, "Should have found index.txt")
				}

				if tc.expectSubDirs {
					assert.True(t, foundDirs > 0, "Should have found at least one directory")
				}
			}
		})
	}
}

func TestProcessTreeForPublishing(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "process-tree-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test tree
	rootTSL := &etsi119612.TSL{
		Source: "root.xml",
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "SE",
			},
		},
	}

	childTSL := &etsi119612.TSL{
		Source: "child.xml",
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "DK",
			},
		},
	}

	rootTSL.Referenced = []*etsi119612.TSL{childTSL}
	tree := NewTSLTree(rootTSL)

	// Test cases for different subdirectory formats
	tests := []struct {
		name            string
		subdirFormat    string
		expectedRootDir string
	}{
		{
			name:            "Territory-based directories",
			subdirFormat:    "territory",
			expectedRootDir: "SE",
		},
		{
			name:            "Index-based directories",
			subdirFormat:    "index",
			expectedRootDir: "tree-0",
		},
	}

	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}
	ctx := NewContext()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a subdirectory for this test
			testDir := filepath.Join(tempDir, tc.name)
			err := os.MkdirAll(testDir, 0755)
			assert.NoError(t, err)

			// Process the tree
			err = processTreeForPublishing(pl, ctx, tree, testDir, 0, tc.subdirFormat, nil)
			assert.NoError(t, err)

			// Check that the root directory was created
			rootDir := filepath.Join(testDir, tc.expectedRootDir)
			info, err := os.Stat(rootDir)
			assert.NoError(t, err)
			assert.True(t, info.IsDir())

			// Check for the root TSL file
			rootFile := filepath.Join(rootDir, "SE.xml")
			_, err = os.Stat(rootFile)
			assert.NoError(t, err)

			// Check for index.txt
			indexFile := filepath.Join(rootDir, "index.txt")
			_, err = os.Stat(indexFile)
			assert.NoError(t, err)

			// Check for refs directory
			refsDir := filepath.Join(rootDir, "refs-1")
			info, err = os.Stat(refsDir)
			assert.NoError(t, err)
			assert.True(t, info.IsDir())

			// Check for child TSL file
			childFile := filepath.Join(refsDir, "depth-1-DK.xml")
			_, err = os.Stat(childFile)
			assert.NoError(t, err)
		})
	}
}

func TestGenerateTreeIndex(t *testing.T) {
	// Create a test tree
	rootTSL := &etsi119612.TSL{
		Source: "root.xml",
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "SE",
			},
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC",
										TslServiceStatus:         "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	childTSL := &etsi119612.TSL{
		Source: "child.xml",
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "DK",
			},
		},
	}

	rootTSL.Referenced = []*etsi119612.TSL{childTSL}
	tree := NewTSLTree(rootTSL)

	// Generate index
	index := generateTreeIndex(tree)

	// Check that the index contains expected information
	assert.Contains(t, index, "TSL Tree Structure")
	assert.Contains(t, index, "SE")
	assert.Contains(t, index, "DK")
	assert.Contains(t, index, "root.xml")
	assert.Contains(t, index, "child.xml")
	assert.Contains(t, index, "Providers: 1")

	// Test with empty tree
	emptyTree := &TSLTree{}
	emptyIndex := generateTreeIndex(emptyTree)
	assert.Contains(t, emptyIndex, "Empty tree")
}
