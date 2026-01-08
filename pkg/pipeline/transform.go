// Package pipeline provides a pipeline framework for processing Trust Status Lists (TSLs).
package pipeline

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/validation"
	"github.com/sirosfoundation/g119612/pkg/xslt"
)

// xsltCache caches XSLT stylesheet content to avoid repeated reads
type xsltCache struct {
	mu    sync.RWMutex
	cache map[string][]byte
}

// Global XSLT cache
var globalXSLTCache = &xsltCache{
	cache: make(map[string][]byte),
}

// get retrieves XSLT content from cache or loads it
func (c *xsltCache) get(key string, loader func() ([]byte, error)) ([]byte, error) {
	// Try read lock first for cache hit
	c.mu.RLock()
	if content, ok := c.cache[key]; ok {
		c.mu.RUnlock()
		return content, nil
	}
	c.mu.RUnlock()

	// Cache miss - acquire write lock and load
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock (another goroutine may have loaded it)
	if content, ok := c.cache[key]; ok {
		return content, nil
	}

	// Load the content
	content, err := loader()
	if err != nil {
		return nil, err
	}

	// Store in cache
	c.cache[key] = content
	return content, nil
}

// clear removes all entries from the cache (useful for testing)
func (c *xsltCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string][]byte)
}

// TransformTSL applies an XSLT transformation to each TSL in the context.
// This pipeline step allows for flexible transformation of TSL XML documents
// using XSLT stylesheets. It can either replace the TSLs in the pipeline context
// with their transformed versions, or output the transformed documents to a
// specified directory.
//
// The step requires the 'xsltproc' command to be available on the system.
//
// Arguments:
//   - arg[0]: Path to the XSLT stylesheet. Can be a filesystem path or an embedded XSLT path.
//     For embedded XSLTs, use the format 'embedded:filename.xslt'.
//     e.g., 'embedded:tsl-to-html.xslt' for the embedded TSL-to-HTML stylesheet.
//   - arg[1]: Mode: "replace" or directory path.
//   - If "replace", transformed TSLs replace the originals in the context.
//   - Otherwise, it's treated as a directory path where transformed TSLs are saved.
//   - arg[2]: (Optional) Output file extension (default: "xml")
//
// Example usage in pipeline YAML for file-based XSLT:
//
//   - transform:
//   - /path/to/stylesheet.xslt
//   - replace
//
// OR for embedded XSLT:
//
//   - transform:
//   - embedded:tsl-to-html.xslt
//   - /output/directory
//   - html
func TransformTSL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 2 {
		return ctx, fmt.Errorf("missing required arguments: need XSLT stylesheet path and mode ('replace' or output directory)")
	}

	// Parse arguments
	xsltPath := args[0]
	mode := args[1]
	extension := "xml"
	if len(args) >= 3 {
		extension = args[2]
	}

	// Validate XSLT path before processing
	if err := validation.ValidateXSLTPath(xsltPath); err != nil {
		return ctx, fmt.Errorf("invalid XSLT path: %w", err)
	}

	// If mode is not "replace", validate it as an output directory
	if mode != "replace" {
		if err := validation.ValidateOutputDirectory(mode); err != nil {
			return ctx, fmt.Errorf("invalid output directory: %w", err)
		}
	}

	// Check if this is an embedded XSLT or a file path
	isEmbedded := xslt.IsEmbeddedPath(xsltPath)

	// Check if the XSLT file exists (if it's not embedded)
	if !isEmbedded {
		if _, err := os.Stat(xsltPath); os.IsNotExist(err) {
			return ctx, fmt.Errorf("XSLT stylesheet not found at path: %s", xsltPath)
		}
	}

	// Check if we need to create an output directory
	isReplace := mode == "replace"
	var outputDir string
	if !isReplace {
		outputDir = mode
		// Create output directory if it doesn't exist
		info, err := os.Stat(outputDir)
		if err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return ctx, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
				}
			} else {
				return ctx, fmt.Errorf("error accessing output directory %s: %w", outputDir, err)
			}
		} else if !info.IsDir() {
			return ctx, fmt.Errorf("%s is not a directory", outputDir)
		}
	}

	if ctx.TSLTrees == nil || ctx.TSLTrees.IsEmpty() {
		return ctx, fmt.Errorf("no TSLs to transform")
	}

	// Collect all TSLs from all trees
	var allTSLs []*etsi119612.TSL
	treeSlice := ctx.TSLTrees.ToSlice()
	for _, tree := range treeSlice {
		if tree == nil {
			continue
		}
		allTSLs = append(allTSLs, tree.ToSlice()...)
	}

	// Perform concurrent transformations
	var transformedTSLs []*etsi119612.TSL
	var err error

	if isReplace {
		transformedTSLs, err = transformTSLsConcurrent(allTSLs, xsltPath, isEmbedded, "", extension)
	} else {
		_, err = transformTSLsConcurrent(allTSLs, xsltPath, isEmbedded, outputDir, extension)
	}

	if err != nil {
		return ctx, err
	}

	// Replace the TSLs in the context if in replace mode
	if isReplace {
		// Clear the existing tree stack
		ctx.TSLTrees = nil
		ctx.EnsureTSLTrees()

		// Add each transformed TSL as a new tree
		for _, transformedTSL := range transformedTSLs {
			tree := NewTSLTree(transformedTSL)
			ctx.AddTSLTree(tree)
		}
	}

	return ctx, nil
}

// transformResult holds the result of a single TSL transformation
type transformResult struct {
	index          int
	transformedXML []byte
	transformedTSL *etsi119612.TSL
	filename       string
	err            error
}

// transformTSLsConcurrent performs concurrent XSLT transformations on multiple TSLs.
//
// This function implements a worker pool pattern to parallelize XSLT transformations,
// providing significant performance improvements when processing multiple TSLs.
//
// Performance characteristics:
//   - Uses a worker pool with up to min(GOMAXPROCS, 8) workers
//   - Achieves 2-3x speedup on multi-core systems compared to sequential processing
//   - Automatically scales to available CPU cores
//   - Each worker processes TSLs independently without shared state
//
// Parameters:
//   - tsls: Slice of TSLs to transform
//   - xsltPath: Path to XSLT stylesheet (file or embedded)
//   - isEmbedded: Whether the XSLT is embedded in the binary
//   - outputDir: Directory for output files (empty for replace mode)
//   - extension: File extension for output files
//
// Returns:
//   - Transformed TSLs (in replace mode) or nil (when writing to files)
//   - Error if any transformation fails
func transformTSLsConcurrent(tsls []*etsi119612.TSL, xsltPath string, isEmbedded bool, outputDir string, extension string) ([]*etsi119612.TSL, error) {
	if len(tsls) == 0 {
		return nil, nil
	}

	// Determine optimal number of workers (use number of CPUs, max 8)
	// We cap at 8 because xsltproc is CPU-intensive and too many concurrent
	// processes can lead to resource contention and diminishing returns
	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers > 8 {
		numWorkers = 8
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	// Create channels for work distribution and result collection
	jobs := make(chan int, len(tsls))
	results := make(chan transformResult, len(tsls))

	// Worker pool
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				result := transformResult{index: i}

				tsl := tsls[i]
				if tsl == nil {
					result.err = fmt.Errorf("TSL at index %d is nil", i)
					results <- result
					continue
				}

				// Create a wrapper struct with the proper XML namespace and element name
				type TrustServiceStatusList struct {
					XMLName                        xml.Name `xml:"http://uri.etsi.org/02231/v2# TrustServiceStatusList"`
					etsi119612.TrustStatusListType `xml:",innerxml"`
				}

				wrapper := TrustServiceStatusList{
					TrustStatusListType: tsl.StatusList,
				}

				xmlData, err := xml.MarshalIndent(wrapper, "", "  ")
				if err != nil {
					result.err = fmt.Errorf("failed to marshal TSL to XML: %w", err)
					results <- result
					continue
				}

				// Add XML header
				xmlData = append([]byte(xml.Header), xmlData...)

				// Apply XSLT transformation
				var transformedXML []byte
				if isEmbedded {
					embeddedName := xslt.ExtractNameFromPath(xsltPath)
					transformedXML, err = applyEmbeddedXSLTTransformation(xmlData, embeddedName)
				} else {
					transformedXML, err = applyFileXSLTTransformation(xmlData, xsltPath)
				}

				if err != nil {
					result.err = fmt.Errorf("XSLT transformation failed: %w", err)
					results <- result
					continue
				}

				result.transformedXML = transformedXML

				// If outputDir is empty (replace mode), parse back to TSL
				if outputDir == "" {
					var transformedTSL etsi119612.TSL
					if err := xml.Unmarshal(transformedXML, &transformedTSL); err != nil {
						result.err = fmt.Errorf("failed to parse transformed XML: %w", err)
						results <- result
						continue
					}
					result.transformedTSL = &transformedTSL
				} else {
					// Determine filename for output
					filename := fmt.Sprintf("transformed-tsl-%d.%s", i, extension)
					if tsl.StatusList.TslSchemeInformation != nil &&
						tsl.StatusList.TslSchemeInformation.TslDistributionPoints != nil &&
						len(tsl.StatusList.TslSchemeInformation.TslDistributionPoints.URI) > 0 {

						uri := tsl.StatusList.TslSchemeInformation.TslDistributionPoints.URI[0]
						parts := strings.Split(uri, "/")
						if len(parts) > 0 && parts[len(parts)-1] != "" {
							baseName := parts[len(parts)-1]
							filename = fmt.Sprintf("%s.%s", strings.TrimSuffix(baseName, filepath.Ext(baseName)), extension)
						}
					}
					result.filename = filename
				}

				results <- result
			}
		}()
	}

	// Send all jobs to the channel
	for i := range tsls {
		jobs <- i
	}
	close(jobs)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	resultMap := make(map[int]transformResult)
	for result := range results {
		if result.err != nil {
			return nil, fmt.Errorf("TSL %d transformation failed: %w", result.index, result.err)
		}
		resultMap[result.index] = result
	}

	// Write files to disk if outputDir specified (must be done sequentially to avoid race conditions)
	if outputDir != "" {
		for i := 0; i < len(tsls); i++ {
			result, ok := resultMap[i]
			if !ok {
				continue
			}
			filePath := filepath.Join(outputDir, result.filename)
			if err := os.WriteFile(filePath, result.transformedXML, 0644); err != nil {
				return nil, fmt.Errorf("failed to write transformed TSL to file %s: %w", filePath, err)
			}
		}
		return nil, nil
	}

	// Return transformed TSLs in original order
	transformedTSLs := make([]*etsi119612.TSL, 0, len(tsls))
	for i := 0; i < len(tsls); i++ {
		result, ok := resultMap[i]
		if !ok || result.transformedTSL == nil {
			continue
		}
		transformedTSLs = append(transformedTSLs, result.transformedTSL)
	}

	return transformedTSLs, nil
}

// applyFileXSLTTransformation applies an XSLT transformation to XML data using an external XSLT file
// The XSLT content is cached after first read to improve performance on subsequent transformations.
func applyFileXSLTTransformation(xmlData []byte, xsltPath string) ([]byte, error) {
	// Get XSLT content from cache or load it
	xsltContent, err := globalXSLTCache.get("file:"+xsltPath, func() ([]byte, error) {
		return os.ReadFile(xsltPath)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read XSLT file: %w", err)
	}

	// Create a temporary file for the input XML
	tempXmlFile, err := os.CreateTemp("", "input-*.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp XML file: %w", err)
	}
	defer os.Remove(tempXmlFile.Name())

	// Write XML data to the temp file
	if _, err := tempXmlFile.Write(xmlData); err != nil {
		return nil, fmt.Errorf("failed to write XML to temp file: %w", err)
	}
	if err := tempXmlFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp XML file: %w", err)
	}

	// Create a temporary file for the XSLT
	tempXsltFile, err := os.CreateTemp("", "style-*.xslt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp XSLT file: %w", err)
	}
	defer os.Remove(tempXsltFile.Name())

	// Write cached XSLT data to the temp file
	if _, err := tempXsltFile.Write(xsltContent); err != nil {
		return nil, fmt.Errorf("failed to write XSLT to temp file: %w", err)
	}
	if err := tempXsltFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp XSLT file: %w", err)
	}

	// Run xsltproc command to apply the transformation
	cmd := exec.Command("xsltproc", tempXsltFile.Name(), tempXmlFile.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("xsltproc error: %w - %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// applyEmbeddedXSLTTransformation applies an XSLT transformation to XML data using an embedded XSLT file
// The embedded XSLT content is cached after first access to improve performance.
func applyEmbeddedXSLTTransformation(xmlData []byte, xsltName string) ([]byte, error) {
	// Get embedded XSLT content from cache or load it
	xsltContent, err := globalXSLTCache.get("embedded:"+xsltName, func() ([]byte, error) {
		return xslt.Get(xsltName)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get embedded XSLT: %w", err)
	}

	// Create a temporary file for the input XML
	tempXmlFile, err := os.CreateTemp("", "input-*.xml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp XML file: %w", err)
	}
	defer os.Remove(tempXmlFile.Name())

	// Write XML data to the temp file
	if _, err := tempXmlFile.Write(xmlData); err != nil {
		return nil, fmt.Errorf("failed to write XML to temp file: %w", err)
	}
	if err := tempXmlFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp XML file: %w", err)
	}

	// Create a temporary file for the XSLT
	tempXsltFile, err := os.CreateTemp("", "style-*.xslt")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp XSLT file: %w", err)
	}
	defer os.Remove(tempXsltFile.Name())

	// Write cached XSLT data to the temp file
	if _, err := tempXsltFile.Write(xsltContent); err != nil {
		return nil, fmt.Errorf("failed to write XSLT to temp file: %w", err)
	}
	if err := tempXsltFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp XSLT file: %w", err)
	}

	// Run xsltproc command to apply the transformation
	cmd := exec.Command("xsltproc", tempXsltFile.Name(), tempXmlFile.Name())
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("xsltproc error: %w - %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

func init() {
	// Register the TransformTSL function
	RegisterFunction("transform", TransformTSL)
}
