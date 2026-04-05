package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirosfoundation/g119612/pkg/logging"
)

// Generate is a unified pipeline step that generates trust lists from a directory
// structure, automatically detecting the target format:
//
// For TSL (ETSI TS 119 612), expects:
//
//	root/
//	  ├── schema.yaml          # TSL scheme metadata
//	  └── providers/           # One subdirectory per trust service provider
//	      └── provider1/
//	          ├── provider.yaml
//	          └── *.pem
//
// For LoTE (ETSI TS 119 602), expects:
//
//	root/
//	  ├── scheme.yaml          # LoTE scheme metadata
//	  └── entities/            # One subdirectory per trusted entity
//	      └── entity1/
//	          ├── entity.yaml
//	          └── *.pem
//
// Format detection:
//   - If `entities/` directory exists → LoTE
//   - If `providers/` directory exists → TSL
//   - If both exist → error (ambiguous)
//
// Usage in pipeline YAML:
//
//   - generate:
//   - /path/to/tsl/source     # Auto-detect based on directory structure
//
// To force a specific format, use the explicit steps:
//   - generate-tsl: ...
//   - generate-lote: ...
func Generate(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("generate requires 1 argument: path to root directory")
	}

	rootDir := args[0]

	// Detect format by directory structure
	providersDir := filepath.Join(rootDir, "providers")
	entitiesDir := filepath.Join(rootDir, "entities")

	hasProviders := dirExists(providersDir)
	hasEntities := dirExists(entitiesDir)

	if hasProviders && hasEntities {
		return nil, fmt.Errorf("ambiguous directory structure in %s: found both 'providers/' (TSL) and 'entities/' (LoTE)", rootDir)
	}

	if !hasProviders && !hasEntities {
		// Check for scheme.yaml as a hint for LoTE (may have empty entities)
		if fileExists(filepath.Join(rootDir, "scheme.yaml")) {
			return GenerateLoTE(pl, ctx, args...)
		}
		return nil, fmt.Errorf("cannot detect format: %s has neither 'providers/' nor 'entities/' directory", rootDir)
	}

	if hasEntities {
		if pl != nil && pl.Logger != nil {
			pl.Logger.Debug("Detected LoTE format (entities/ directory found)",
				logging.F("root", rootDir))
		}
		return GenerateLoTE(pl, ctx, args...)
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Debug("Detected TSL format (providers/ directory found)",
			logging.F("root", rootDir))
	}
	return GenerateTSL(pl, ctx, args...)
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
