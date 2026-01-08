package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
)

// BenchmarkXSLTCaching benchmarks the performance improvement from XSLT caching
func BenchmarkXSLTCaching(b *testing.B) {
	// Skip if xsltproc is not available
	if _, err := os.Stat("/usr/bin/xsltproc"); os.IsNotExist(err) {
		b.Skip("xsltproc not available, skipping benchmark")
	}

	// Create a temporary XSLT file
	tempDir := b.TempDir()
	xsltPath := filepath.Join(tempDir, "test.xslt")
	xsltContent := []byte(`<?xml version="1.0"?>
<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:output method="xml" indent="yes"/>
  <xsl:template match="/">
    <result>transformed</result>
  </xsl:template>
</xsl:stylesheet>`)

	if err := os.WriteFile(xsltPath, xsltContent, 0644); err != nil {
		b.Fatalf("Failed to create test XSLT file: %v", err)
	}

	xmlData := []byte(`<?xml version="1.0"?><input>test</input>`)

	b.Run("WithCache", func(b *testing.B) {
		// Clear cache before benchmark
		globalXSLTCache.clear()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := applyFileXSLTTransformation(xmlData, xsltPath)
			if err != nil {
				b.Fatalf("Transformation failed: %v", err)
			}
		}
	})

	b.Run("WithoutCache", func(b *testing.B) {
		// Clear cache before each iteration to simulate no caching
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			globalXSLTCache.clear()
			_, err := applyFileXSLTTransformation(xmlData, xsltPath)
			if err != nil {
				b.Fatalf("Transformation failed: %v", err)
			}
		}
	})
}

// BenchmarkEmbeddedXSLTCaching benchmarks caching for embedded XSLTs
func BenchmarkEmbeddedXSLTCaching(b *testing.B) {
	// Skip if xsltproc is not available
	if _, err := os.Stat("/usr/bin/xsltproc"); os.IsNotExist(err) {
		b.Skip("xsltproc not available, skipping benchmark")
	}

	xsltName := "tsl-to-html.xslt"
	xmlData := []byte(`<?xml version="1.0"?>
<TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#">
  <SchemeInformation>
    <TSLSequenceNumber>1</TSLSequenceNumber>
  </SchemeInformation>
</TrustServiceStatusList>`)

	b.Run("WithCache", func(b *testing.B) {
		// Clear cache before benchmark
		globalXSLTCache.clear()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := applyEmbeddedXSLTTransformation(xmlData, xsltName)
			if err != nil {
				b.Fatalf("Transformation failed: %v", err)
			}
		}
	})

	b.Run("WithoutCache", func(b *testing.B) {
		// Clear cache before each iteration
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			globalXSLTCache.clear()
			_, err := applyEmbeddedXSLTTransformation(xmlData, xsltName)
			if err != nil {
				b.Fatalf("Transformation failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentWithCaching benchmarks the combined effect of concurrency + caching
func BenchmarkConcurrentWithCaching(b *testing.B) {
	// Skip if xsltproc is not available
	if _, err := os.Stat("/usr/bin/xsltproc"); os.IsNotExist(err) {
		b.Skip("xsltproc not available, skipping benchmark")
	}

	// Create temporary output directory
	tempDir := b.TempDir()

	createTestTSL := func() *etsi119612.TSL {
		return &etsi119612.TSL{
			StatusList: etsi119612.TrustStatusListType{
				TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
					TslDistributionPoints: &etsi119612.NonEmptyURIListType{
						URI: []string{"https://example.com/test-tsl.xml"},
					},
				},
			},
		}
	}

	// Test with 20 TSLs
	numTSLs := 20
	tsls := make([]*etsi119612.TSL, numTSLs)
	for i := 0; i < numTSLs; i++ {
		tsls[i] = createTestTSL()
	}

	b.Run("WithCache", func(b *testing.B) {
		// Clear cache once before benchmark (simulates warm cache)
		globalXSLTCache.clear()

		// Do one warmup transformation to populate cache
		outputDir := filepath.Join(tempDir, "warmup")
		os.MkdirAll(outputDir, 0755)
		_, _ = transformTSLsConcurrent(tsls[:1], "embedded:tsl-to-html.xslt", true, outputDir, "html")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			outputDir := filepath.Join(tempDir, "with-cache", fmt.Sprintf("%d", i))
			os.MkdirAll(outputDir, 0755)
			_, err := transformTSLsConcurrent(tsls, "embedded:tsl-to-html.xslt", true, outputDir, "html")
			if err != nil {
				b.Fatalf("Transformation failed: %v", err)
			}
		}
	})

	b.Run("WithoutCache", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			globalXSLTCache.clear()
			outputDir := filepath.Join(tempDir, "without-cache", fmt.Sprintf("%d", i))
			os.MkdirAll(outputDir, 0755)
			_, err := transformTSLsConcurrent(tsls, "embedded:tsl-to-html.xslt", true, outputDir, "html")
			if err != nil {
				b.Fatalf("Transformation failed: %v", err)
			}
		}
	})
}
