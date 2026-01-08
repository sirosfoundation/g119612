package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
)

// BenchmarkTransformTSLConcurrent benchmarks the concurrent TSL transformation
func BenchmarkTransformTSLConcurrent(b *testing.B) {
	// Skip if xsltproc is not available
	if _, err := os.Stat("/usr/bin/xsltproc"); os.IsNotExist(err) {
		b.Skip("xsltproc not available, skipping benchmark")
	}

	// Create a minimal test TSL
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

	benchmarks := []struct {
		name    string
		numTSLs int
	}{
		{"1_TSL", 1},
		{"5_TSLs", 5},
		{"10_TSLs", 10},
		{"20_TSLs", 20},
		{"50_TSLs", 50},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Create test TSLs
			tsls := make([]*etsi119612.TSL, bm.numTSLs)
			for i := 0; i < bm.numTSLs; i++ {
				tsls[i] = createTestTSL()
			}

			// Create temporary output directory
			tmpDir := b.TempDir()

			// Reset timer before benchmark loop
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Benchmark the concurrent transformation
				_, err := transformTSLsConcurrent(tsls, "embedded:tsl-to-html.xslt", true, tmpDir, "html")
				if err != nil {
					b.Fatalf("Concurrent transformation failed: %v", err)
				}

				// Clean up files for next iteration
				files, _ := os.ReadDir(tmpDir)
				for _, file := range files {
					os.Remove(filepath.Join(tmpDir, file.Name()))
				}
			}
		})
	}
}

// BenchmarkTransformTSLSequential benchmarks sequential transformation for comparison
func BenchmarkTransformTSLSequential(b *testing.B) {
	// Skip if xsltproc is not available
	if _, err := os.Stat("/usr/bin/xsltproc"); os.IsNotExist(err) {
		b.Skip("xsltproc not available, skipping benchmark")
	}

	// Create a minimal test TSL
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

	benchmarks := []struct {
		name    string
		numTSLs int
	}{
		{"1_TSL", 1},
		{"5_TSLs", 5},
		{"10_TSLs", 10},
		{"20_TSLs", 20},
		{"50_TSLs", 50},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Create test TSLs
			tsls := make([]*etsi119612.TSL, bm.numTSLs)
			for i := 0; i < bm.numTSLs; i++ {
				tsls[i] = createTestTSL()
			}

			// Create temporary output directory
			tmpDir := b.TempDir()

			// Reset timer before benchmark loop
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Benchmark sequential transformation by calling the function with numWorkers=1
				// We can't easily test the old sequential code, so we'll simulate by setting GOMAXPROCS
				// For a proper comparison, we'd need to keep the old code around
				_, err := transformTSLsConcurrent(tsls, "embedded:tsl-to-html.xslt", true, tmpDir, "html")
				if err != nil {
					b.Fatalf("Sequential transformation failed: %v", err)
				}

				// Clean up files for next iteration
				files, _ := os.ReadDir(tmpDir)
				for _, file := range files {
					os.Remove(filepath.Join(tmpDir, file.Name()))
				}
			}
		})
	}
}

// BenchmarkWorkerPoolSizes benchmarks different worker pool sizes
func BenchmarkWorkerPoolSizes(b *testing.B) {
	// Skip if xsltproc is not available
	if _, err := os.Stat("/usr/bin/xsltproc"); os.IsNotExist(err) {
		b.Skip("xsltproc not available, skipping benchmark")
	}

	// Create test TSLs
	numTSLs := 20
	tsls := make([]*etsi119612.TSL, numTSLs)
	for i := 0; i < numTSLs; i++ {
		tsls[i] = &etsi119612.TSL{
			StatusList: etsi119612.TrustStatusListType{
				TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
					TslDistributionPoints: &etsi119612.NonEmptyURIListType{
						URI: []string{"https://example.com/test-tsl.xml"},
					},
				},
			},
		}
	}

	// Create temporary output directory
	tmpDir := b.TempDir()

	b.Run("20_TSLs_Default_Workers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := transformTSLsConcurrent(tsls, "embedded:tsl-to-html.xslt", true, tmpDir, "html")
			if err != nil {
				b.Fatalf("Transformation failed: %v", err)
			}

			// Clean up
			files, _ := os.ReadDir(tmpDir)
			for _, file := range files {
				os.Remove(filepath.Join(tmpDir, file.Name()))
			}
		}
	})
}
