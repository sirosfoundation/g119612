package pipeline

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/dsig"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

// processTreeForPublishing processes a TSL tree for publishing,
// maintaining the tree structure in the file system
func processTreeForPublishing(pl *Pipeline, ctx *Context, tree *TSLTree, baseDir string, treeIndex int, subdirFormat string, signer dsig.XMLSigner) error {
	if tree == nil || tree.Root == nil {
		return nil
	}

	// Start with the root TSL
	rootTSL := tree.Root.TSL
	if rootTSL == nil {
		return nil
	}

	// Determine the root directory for this tree
	var treeDir string
	if subdirFormat == "territory" && rootTSL.StatusList.TslSchemeInformation != nil {
		territory := rootTSL.StatusList.TslSchemeInformation.TslSchemeTerritory
		if territory != "" {
			treeDir = filepath.Join(baseDir, territory)
		} else {
			treeDir = filepath.Join(baseDir, fmt.Sprintf("tree-%d", treeIndex))
		}
	} else {
		// Use index-based directory
		treeDir = filepath.Join(baseDir, fmt.Sprintf("tree-%d", treeIndex))
	}

	// Create the tree directory if it doesn't exist
	pl.Logger.Info("Creating tree directory",
		logging.F("directory", treeDir),
		logging.F("territory", rootTSL.StatusList.TslSchemeInformation.TslSchemeTerritory),
		logging.F("format", subdirFormat))
	if err := os.MkdirAll(treeDir, 0755); err != nil {
		return fmt.Errorf("failed to create tree directory %s: %w", treeDir, err)
	}

	// Process the tree recursively
	return processNodeForPublishing(pl, ctx, tree.Root, treeDir, 0, signer)
}

// publishTSLToFile writes a TSL to a file, optionally signing it
func publishTSLToFile(pl *Pipeline, tsl *etsi119612.TSL, filePath string, signer dsig.XMLSigner) error {
	if tsl == nil {
		return fmt.Errorf("cannot publish nil TSL")
	}

	// Create XML representation with root element
	type TrustStatusListWrapper struct {
		XMLName xml.Name                       `xml:"TrustServiceStatusList"`
		List    etsi119612.TrustStatusListType `xml:",innerxml"`
	}
	wrapper := TrustStatusListWrapper{List: tsl.StatusList}
	xmlData, err := xml.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal TSL to XML: %w", err)
	}

	// Add XML header
	xmlData = append([]byte(xml.Header), xmlData...)

	// Sign the XML if a signer is provided
	if signer != nil {
		xmlData, err = signer.Sign(xmlData)
		if err != nil {
			return fmt.Errorf("failed to sign XML: %w", err)
		}
	}

	// Write to file
	if err := os.WriteFile(filePath, xmlData, 0644); err != nil {
		return fmt.Errorf("failed to write TSL to file %s: %w", filePath, err)
	}

	// Log success
	pl.Logger.Info("Published TSL",
		logging.F("file", filePath),
		logging.F("signed", signer != nil),
		logging.F("size", len(xmlData)))

	return nil
}

// processNodeForPublishing recursively processes a TSL node for publishing
func processNodeForPublishing(pl *Pipeline, ctx *Context, node *TSLNode, dirPath string, depth int, signer dsig.XMLSigner) error {
	if node == nil || node.TSL == nil {
		return nil
	}

	// Publish this node's TSL
	tsl := node.TSL

	// Determine filename
	filename := fmt.Sprintf("tsl-depth-%d.xml", depth)
	if tsl.StatusList.TslSchemeInformation != nil {
		// Use scheme territory if available
		territory := tsl.StatusList.TslSchemeInformation.TslSchemeTerritory
		if territory != "" {
			filename = fmt.Sprintf("%s.xml", territory)
		}

		// Use distribution point if available
		if tsl.StatusList.TslSchemeInformation.TslDistributionPoints != nil &&
			len(tsl.StatusList.TslSchemeInformation.TslDistributionPoints.URI) > 0 {

			uri := tsl.StatusList.TslSchemeInformation.TslDistributionPoints.URI[0]
			parts := strings.Split(uri, "/")
			if len(parts) > 0 && parts[len(parts)-1] != "" {
				filename = parts[len(parts)-1]
			}
		}
	}

	// For referenced TSLs at deeper levels, create subdirectories
	nodePath := dirPath
	if depth > 0 {
		// Create a depth-based subdirectory
		nodePath = filepath.Join(dirPath, fmt.Sprintf("refs-%d", depth))
		if err := os.MkdirAll(nodePath, 0755); err != nil {
			return fmt.Errorf("failed to create depth directory %s: %w", nodePath, err)
		}

		// Add depth prefix to filename for clarity
		if !strings.HasPrefix(filename, fmt.Sprintf("depth-%d-", depth)) {
			filename = fmt.Sprintf("depth-%d-%s", depth, filename)
		}
	}

	// Publish the TSL
	filePath := filepath.Join(nodePath, filename)
	if err := publishTSLToFile(pl, tsl, filePath, signer); err != nil {
		return fmt.Errorf("failed to publish TSL to %s: %w", filePath, err)
	}

	// Create an index file that shows the tree structure
	if depth == 0 {
		// Find the tree that contains this node (for the index)
		nodeTree := &TSLTree{Root: node}
		indexContent := generateTreeIndex(nodeTree)
		indexPath := filepath.Join(dirPath, "index.txt")
		if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
			pl.Logger.Warn("Failed to write tree index", logging.F("path", indexPath), logging.F("error", err))
		}
	}

	// Process all child nodes
	for i, child := range node.Children {
		if err := processNodeForPublishing(pl, ctx, child, dirPath, depth+1, signer); err != nil {
			return fmt.Errorf("failed to process child %d: %w", i, err)
		}
	}

	return nil
}

// generateTreeIndex creates a text representation of the TSL tree structure
func generateTreeIndex(tree *TSLTree) string {
	if tree == nil || tree.Root == nil {
		return "Empty tree"
	}

	var sb strings.Builder
	sb.WriteString("TSL Tree Structure:\n")
	sb.WriteString("==================\n\n")

	generateNodeIndex(tree.Root, &sb, 0)

	return sb.String()
}

// generateNodeIndex recursively builds a text representation of a TSL node and its children
func generateNodeIndex(node *TSLNode, sb *strings.Builder, depth int) {
	if node == nil || node.TSL == nil {
		return
	}

	// Indent based on depth
	indent := strings.Repeat("  ", depth)

	// Get TSL information
	source := "unknown"
	territory := "unknown"
	if node.TSL.Source != "" {
		source = node.TSL.Source
	}
	if node.TSL.StatusList.TslSchemeInformation != nil &&
		node.TSL.StatusList.TslSchemeInformation.TslSchemeTerritory != "" {
		territory = node.TSL.StatusList.TslSchemeInformation.TslSchemeTerritory
	}

	// Count providers and services
	providerCount := 0
	serviceCount := 0
	if node.TSL.StatusList.TslTrustServiceProviderList != nil {
		providers := node.TSL.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider
		providerCount = len(providers)

		// Count services
		for _, provider := range providers {
			if provider != nil && provider.TslTSPServices != nil {
				serviceCount += len(provider.TslTSPServices.TslTSPService)
			}
		}
	}

	// Write the node information
	sb.WriteString(fmt.Sprintf("%s- [%s] %s (Providers: %d, Services: %d)\n",
		indent, territory, source, providerCount, serviceCount))

	// Process child nodes
	for _, child := range node.Children {
		generateNodeIndex(child, sb, depth+1)
	}
}
