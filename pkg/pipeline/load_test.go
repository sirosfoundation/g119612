package pipeline

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestLoadTSLWithDepthControl(t *testing.T) {
	// Create temporary files for the TSLs
	tempDir, err := ioutil.TempDir("", "tsl-depth-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get absolute paths for referenced files
	ref1Path := filepath.Join(tempDir, "referenced1.xml")
	ref2Path := filepath.Join(tempDir, "referenced2.xml")

	// Create a main TSL file that points to other TSLs with absolute paths
	mainTSLContent := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
  <SchemeInformation>
    <TSLVersionIdentifier>5</TSLVersionIdentifier>
    <TSLSequenceNumber>1</TSLSequenceNumber>
    <TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
    <SchemeTerritory>TEST</SchemeTerritory>
    <PointersToOtherTSL>
      <OtherTSLPointer>
        <TSLLocation>file://` + ref1Path + `</TSLLocation>
      </OtherTSLPointer>
    </PointersToOtherTSL>
  </SchemeInformation>
  <TrustServiceProviderList>
    <TrustServiceProvider>
      <TSPInformation>
        <TSPName>
          <Name xml:lang="en">Test Provider 1</Name>
        </TSPName>
      </TSPInformation>
      <TSPServices>
        <TSPService>
          <ServiceInformation>
            <ServiceTypeIdentifier>http://test-service</ServiceTypeIdentifier>
          </ServiceInformation>
        </TSPService>
      </TSPServices>
    </TrustServiceProvider>
  </TrustServiceProviderList>
</TrustServiceStatusList>`

	refTSL1Content := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
  <SchemeInformation>
    <TSLVersionIdentifier>5</TSLVersionIdentifier>
    <TSLSequenceNumber>2</TSLSequenceNumber>
    <TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
    <SchemeTerritory>REF1</SchemeTerritory>
    <PointersToOtherTSL>
      <OtherTSLPointer>
        <TSLLocation>file://` + ref2Path + `</TSLLocation>
      </OtherTSLPointer>
    </PointersToOtherTSL>
  </SchemeInformation>
</TrustServiceStatusList>`

	refTSL2Content := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
  <SchemeInformation>
    <TSLVersionIdentifier>5</TSLVersionIdentifier>
    <TSLSequenceNumber>3</TSLSequenceNumber>
    <TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
    <SchemeTerritory>REF2</SchemeTerritory>
  </SchemeInformation>
</TrustServiceStatusList>`

	// Create temporary files for the TSLs
	mainTSLFile, err := os.Create(filepath.Join(tempDir, "main.xml"))
	if err != nil {
		t.Fatalf("Failed to create main TSL file: %v", err)
	}
	if _, err := mainTSLFile.Write([]byte(mainTSLContent)); err != nil {
		t.Fatalf("Failed to write main TSL: %v", err)
	}
	mainTSLFile.Close()

	refTSL1File, err := os.Create(ref1Path)
	if err != nil {
		t.Fatalf("Failed to create ref1 TSL file: %v", err)
	}
	if _, err := refTSL1File.Write([]byte(refTSL1Content)); err != nil {
		t.Fatalf("Failed to write ref1 TSL: %v", err)
	}
	refTSL1File.Close()

	refTSL2File, err := os.Create(ref2Path)
	if err != nil {
		t.Fatalf("Failed to create ref2 TSL file: %v", err)
	}
	if _, err := refTSL2File.Write([]byte(refTSL2Content)); err != nil {
		t.Fatalf("Failed to write ref2 TSL: %v", err)
	}
	refTSL2File.Close()

	// Test with different depth settings
	tests := []struct {
		name                string
		maxDepth            int
		expectedDepth       int
		expectedCount       int
		expectedTerritories []string
	}{
		{
			name:                "No references",
			maxDepth:            0,
			expectedDepth:       0,
			expectedCount:       1,
			expectedTerritories: []string{"TEST"},
		},
		{
			name:                "One level references",
			maxDepth:            1,
			expectedDepth:       1,
			expectedCount:       2,
			expectedTerritories: []string{"TEST", "REF1"},
		},
		{
			name:                "All references",
			maxDepth:            -1,
			expectedDepth:       2,
			expectedCount:       3,
			expectedTerritories: []string{"TEST", "REF1", "REF2"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create pipeline and context
			pl := &Pipeline{
				Logger: logging.NewLogger(logging.DebugLevel),
			}
			ctx := NewContext()
			ctx.EnsureTSLFetchOptions()

			// Set max dereference depth
			ctx.TSLFetchOptions.MaxDereferenceDepth = tc.maxDepth

			// Load the TSL
			resultCtx, err := LoadTSL(pl, ctx, filepath.Join(tempDir, "main.xml"))
			assert.NoError(t, err)

			// Check tree depth
			tree, ok := resultCtx.TSLTrees.Peek()
			assert.True(t, ok)
			assert.Equal(t, tc.expectedDepth, tree.Depth())

			// Check total TSL count (flattened)
			allTSLs := tree.ToSlice()
			assert.Equal(t, tc.expectedCount, len(allTSLs))

			// Verify territories are as expected
			territories := make(map[string]bool)
			for _, tsl := range allTSLs {
				if tsl.StatusList.TslSchemeInformation != nil {
					territory := tsl.StatusList.TslSchemeInformation.TslSchemeTerritory
					territories[territory] = true
				}
			}

			for _, expectedTerritory := range tc.expectedTerritories {
				assert.True(t, territories[expectedTerritory], "Expected territory %s not found", expectedTerritory)
			}
		})
	}
}

func TestLoadTSLServiceCounting(t *testing.T) {
	// Create a TSL with multiple providers and services
	tslContent := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
  <SchemeInformation>
    <TSLVersionIdentifier>5</TSLVersionIdentifier>
    <TSLSequenceNumber>1</TSLSequenceNumber>
    <TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
    <SchemeTerritory>TEST</SchemeTerritory>
  </SchemeInformation>
  <TrustServiceProviderList>
    <TrustServiceProvider>
      <TSPInformation>
        <TSPName>
          <Name xml:lang="en">Provider 1</Name>
        </TSPName>
      </TSPInformation>
      <TSPServices>
        <TSPService>
          <ServiceInformation>
            <ServiceTypeIdentifier>http://service-type-1</ServiceTypeIdentifier>
          </ServiceInformation>
        </TSPService>
        <TSPService>
          <ServiceInformation>
            <ServiceTypeIdentifier>http://service-type-2</ServiceTypeIdentifier>
          </ServiceInformation>
        </TSPService>
      </TSPServices>
    </TrustServiceProvider>
    <TrustServiceProvider>
      <TSPInformation>
        <TSPName>
          <Name xml:lang="en">Provider 2</Name>
        </TSPName>
      </TSPInformation>
      <TSPServices>
        <TSPService>
          <ServiceInformation>
            <ServiceTypeIdentifier>http://service-type-3</ServiceTypeIdentifier>
          </ServiceInformation>
        </TSPService>
      </TSPServices>
    </TrustServiceProvider>
  </TrustServiceProviderList>
</TrustServiceStatusList>`

	// Create temporary file for the TSL
	tempFile, err := ioutil.TempFile("", "service-count-test-*.xml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write([]byte(tslContent)); err != nil {
		t.Fatalf("Failed to write TSL content: %v", err)
	}
	tempFile.Close()

	// Create pipeline and context
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}
	ctx := NewContext()

	// Load the TSL
	resultCtx, err := LoadTSL(pl, ctx, tempFile.Name())
	assert.NoError(t, err)

	// Check provider and service counts
	tree, ok := resultCtx.TSLTrees.Peek()
	assert.True(t, ok)

	root := tree.Root.TSL
	providerCount := 0
	serviceCount := 0

	if root.StatusList.TslTrustServiceProviderList != nil {
		providerCount = len(root.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider)

		for _, provider := range root.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider {
			if provider != nil && provider.TslTSPServices != nil {
				serviceCount += len(provider.TslTSPServices.TslTSPService)
			}
		}
	}

	assert.Equal(t, 2, providerCount, "Should have 2 providers")
	assert.Equal(t, 3, serviceCount, "Should have 3 services")
}
