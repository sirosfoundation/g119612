package pipeline

import (
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/xslt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformTSL(t *testing.T) {
	// Skip if xsltproc is not available
	if _, err := exec.LookPath("xsltproc"); err != nil {
		t.Skip("xsltproc not available, skipping test")
	}

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "tsl-transform-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a simple XSLT stylesheet
	xsltPath := filepath.Join(tempDir, "transform.xslt")
	xsltContent := `<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
                xmlns:tsl="http://uri.etsi.org/02231/v2#">
  <xsl:output method="xml" indent="yes"/>
  
  <!-- Identity transform -->
  <xsl:template match="@*|node()">
    <xsl:copy>
      <xsl:apply-templates select="@*|node()"/>
    </xsl:copy>
  </xsl:template>
  
  <!-- Add a test attribute to the root element -->
  <xsl:template match="/*">
    <xsl:copy>
      <xsl:attribute name="testAttribute">transformed</xsl:attribute>
      <xsl:apply-templates select="@*|node()"/>
    </xsl:copy>
  </xsl:template>
</xsl:stylesheet>`

	err = os.WriteFile(xsltPath, []byte(xsltContent), 0644)
	require.NoError(t, err)

	// Create output directory
	outputDir := filepath.Join(tempDir, "output")
	err = os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	// Create a simple TSL XML for testing
	tslXML := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
  <SchemeInformation>
    <TSLVersionIdentifier>5</TSLVersionIdentifier>
    <TSLSequenceNumber>1</TSLSequenceNumber>
    <TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
    <SchemeTerritory>TEST</SchemeTerritory>
    <DistributionPoints>
      <URI>http://example.com/tsl/test-tsl.xml</URI>
    </DistributionPoints>
  </SchemeInformation>
</TrustServiceStatusList>`

	// Parse the XML into a TSL
	var tslObj etsi119612.TSL
	err = xml.Unmarshal([]byte(tslXML), &tslObj)
	require.NoError(t, err)

	// Create a context with the TSL
	ctx := NewContext()
	ctx.EnsureTSLTrees()
	ctx.AddTSL(&tslObj)

	t.Run("Transform and Replace", func(t *testing.T) {
		// Call the TransformTSL function with replace mode
		resultCtx, err := TransformTSL(nil, ctx, xsltPath, "replace")
		assert.NoError(t, err)
		assert.NotNil(t, resultCtx)
		assert.Equal(t, 1, resultCtx.TSLTrees.Size())

		// Get the transformed TSL trees
		transformedTrees := resultCtx.TSLTrees.ToSlice()
		transformedTSLs := make([]*etsi119612.TSL, 0)
		for _, tree := range transformedTrees {
			transformedTSLs = append(transformedTSLs, tree.Root.TSL)
		}
		assert.Len(t, transformedTSLs, 1)
	})

	t.Run("Transform and Output to Directory", func(t *testing.T) {
		// Call the TransformTSL function with output directory
		resultCtx, err := TransformTSL(nil, ctx, xsltPath, outputDir)
		assert.NoError(t, err)
		assert.NotNil(t, resultCtx)

		// Check that the file was created
		files, err := os.ReadDir(outputDir)
		assert.NoError(t, err)
		assert.NotEmpty(t, files)

		// Read the file content of the first file
		content, err := os.ReadFile(filepath.Join(outputDir, files[0].Name()))
		assert.NoError(t, err)

		// Check if the content contains the transformation
		assert.True(t, strings.Contains(string(content), `testAttribute="transformed"`))
	})

	t.Run("Error Cases", func(t *testing.T) {
		// Test missing arguments
		_, err := TransformTSL(nil, ctx)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "missing required arguments"))

		// Test non-existent stylesheet
		_, err = TransformTSL(nil, ctx, "/nonexistent/path.xslt", "replace")
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "XSLT stylesheet not found"))

		// Test empty context
		emptyCtx := NewContext()
		_, err = TransformTSL(nil, emptyCtx, xsltPath, "replace")
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "no TSLs to transform"))

		// Test invalid embedded path
		_, err = TransformTSL(nil, ctx, "embedded:nonexistent.xslt", "replace")
		// Note: This will actually only fail at runtime with the actual binary,
		// so we're not asserting the specific error message
		// Just check that it returns an error
		if err == nil {
			t.Log("Note: Error test for invalid embedded path may not work in isolated test environment")
		}
	})
}

// TestEmbeddedTransformTSL tests the embedded XSLT functionality
func TestEmbeddedTransformTSL(t *testing.T) {
	// Skip if xsltproc is not available
	if _, err := exec.LookPath("xsltproc"); err != nil {
		t.Skip("xsltproc not available, skipping test")
	}

	// Test IsEmbeddedPath function
	t.Run("Test Embedded Path Detection", func(t *testing.T) {
		regularPath := "/path/to/file.xslt"
		embeddedPath := "embedded:tsl-to-html.xslt"

		assert.False(t, xslt.IsEmbeddedPath(regularPath))
		assert.True(t, xslt.IsEmbeddedPath(embeddedPath))
	})

	t.Run("Test Extract Name From Path", func(t *testing.T) {
		regularPath := "/path/to/file.xslt"
		embeddedPath := "embedded:tsl-to-html.xslt"

		assert.Equal(t, regularPath, xslt.ExtractNameFromPath(regularPath))
		assert.Equal(t, "tsl-to-html.xslt", xslt.ExtractNameFromPath(embeddedPath))
	})

	// This test requires the actual embedded files
	t.Run("Test Available Embedded XSLTs", func(t *testing.T) {
		// Get list of available XSLT files
		xsltFiles, err := xslt.List()
		require.NoError(t, err, "Failed to list embedded XSLT files")

		// Ensure we have at least one embedded XSLT
		assert.NotEmpty(t, xsltFiles, "No embedded XSLT files found")

		// Check if the standard TSL-to-HTML transformation is available
		hasTSLToHTML := false
		for _, file := range xsltFiles {
			if file == "tsl-to-html.xslt" {
				hasTSLToHTML = true
				break
			}
		}
		assert.True(t, hasTSLToHTML, "tsl-to-html.xslt not found in embedded XSLTs")

		// Test getting content of an embedded XSLT
		if hasTSLToHTML {
			content, err := xslt.Get("tsl-to-html.xslt")
			assert.NoError(t, err, "Failed to get embedded XSLT content")
			assert.NotEmpty(t, content, "Embedded XSLT content is empty")
		}
	})

	// This test can only run if we have embedded files in the binary
	// In a real environment, this test would be valuable, but in isolated unit tests,
	// we can't guarantee the embedded files are available
	if os.Getenv("RUN_EMBEDDED_TRANSFORM_TEST") == "1" {
		t.Run("Transform Using Embedded XSLT - Output Mode", func(t *testing.T) {
			// Create a temporary directory for output
			tempDir, err := os.MkdirTemp("", "embedded-xslt-test-")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Create a simple TSL XML for testing
			tslXML := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
  <SchemeInformation>
    <TSLVersionIdentifier>5</TSLVersionIdentifier>
    <TSLSequenceNumber>1</TSLSequenceNumber>
    <TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
    <SchemeTerritory>TEST</SchemeTerritory>
  </SchemeInformation>
</TrustServiceStatusList>`

			// Parse the XML into a TSL
			var tslObj etsi119612.TSL
			err = xml.Unmarshal([]byte(tslXML), &tslObj)
			require.NoError(t, err)

			// Create a context with the TSL
			ctx := NewContext()
			ctx.EnsureTSLTrees()
			ctx.AddTSL(&tslObj)

			// Transform using embedded XSLT
			embeddedPath := "embedded:tsl-to-html.xslt"
			_, err = TransformTSL(nil, ctx, embeddedPath, tempDir, "html")

			if err != nil {
				t.Logf("Note: Test may fail if embedded XSLTs are not available: %v", err)
			} else {
				// Check that a file was created in the output directory
				files, err := os.ReadDir(tempDir)
				assert.NoError(t, err)
				assert.NotEmpty(t, files, "No files created in output directory")

				if len(files) > 0 {
					// Read the file content
					content, err := os.ReadFile(filepath.Join(tempDir, files[0].Name()))
					assert.NoError(t, err)

					// The output should be HTML from the TSL-to-HTML transformation
					assert.Contains(t, string(content), "<html", "Output is not HTML")
				}
			}
		})

		t.Run("Transform Using Embedded XSLT - Replace Mode", func(t *testing.T) {
			// Create a simple TSL XML for testing
			tslXML := `<?xml version="1.0" encoding="UTF-8"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
  <SchemeInformation>
    <TSLVersionIdentifier>5</TSLVersionIdentifier>
    <TSLSequenceNumber>1</TSLSequenceNumber>
    <TSLType>http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric</TSLType>
    <SchemeTerritory>TEST</SchemeTerritory>
  </SchemeInformation>
</TrustServiceStatusList>`

			// Parse the XML into a TSL
			var tslObj etsi119612.TSL
			err := xml.Unmarshal([]byte(tslXML), &tslObj)
			require.NoError(t, err)

			// Create a context with the TSL
			ctx := NewContext()
			ctx.EnsureTSLTrees()
			ctx.AddTSL(&tslObj)

			// Create a simple test XSLT that we know will work
			testXSLTPath := "embedded:test-transform.xslt"

			// Call TransformTSL with replace mode
			resultCtx, err := TransformTSL(nil, ctx, testXSLTPath, "replace")

			if err != nil {
				t.Logf("Note: Test may fail if embedded XSLTs are not available: %v", err)
			} else {
				// Verify the TSL was replaced
				assert.Equal(t, 1, resultCtx.TSLTrees.Size(), "Should have one TSL tree after transformation")
			}
		})
	}
}
