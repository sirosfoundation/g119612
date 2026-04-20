package pipeline

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirosfoundation/g119612/pkg/dsig"
	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/jws"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/sirosfoundation/g119612/pkg/validation"
)

// PublishLoTE writes LoTE documents from ctx.LoTEs to JSON files,
// optionally signing them with JWS.
//
// By default, signatures use JAdES-B-B profile (ETSI TS 119 182-1).
// Pass "jades:false" as an argument to disable JAdES headers and produce plain JWS.
//
// Usage in pipeline YAML:
//
//   - publish-lote:
//   - /path/to/output/dir                                              # unsigned JSON
//   - publish-lote:
//   - ["/path/to/dir", "/cert.pem", "/key.pem"]                       # JAdES-signed (default)
//   - publish-lote:
//   - ["/path/to/dir", "/cert.pem", "/key.pem", "jades:false"]        # plain JWS
//   - publish-lote:
//   - ["/path/to/dir", "pkcs11:module=/path;pin=1234", "key", "cert"] # JAdES-signed with PKCS#11
func PublishLoTE(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("publish-lote requires at least 1 argument: output directory")
	}

	outputDir := args[0]

	// Validate output directory
	if err := validation.ValidateOutputDirectory(outputDir); err != nil {
		return nil, fmt.Errorf("invalid output directory: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Create JWS signer from args
	signer, err := createLoTESigner(args[1:])
	if err != nil {
		return nil, err
	}

	if ctx.LoTEs == nil || ctx.LoTEs.Size() == 0 {
		if pl != nil && pl.Logger != nil {
			pl.Logger.Warn("No LoTEs in context to publish")
		}
		return ctx, nil
	}

	// Validate all LoTEs before writing anything
	lotes := ctx.LoTEs.ToSlice()
	for i, lote := range lotes {
		if err := lote.Validate(); err != nil {
			return nil, fmt.Errorf("LoTE %d failed validation: %w", i, err)
		}
	}

	usedFilenames := make(map[string]bool)
	for i, lote := range lotes {
		filename := loteFilename(lote, i)
		// Prevent filename collisions
		if usedFilenames[filename] {
			filename = fmt.Sprintf("lote-%s-%d.json", lote.SchemeInformation.Territory, i)
		}
		usedFilenames[filename] = true

		jsonData, err := lote.MarshalIndent()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal LoTE %d: %w", i, err)
		}

		outputPath := filepath.Join(outputDir, filename)

		if signer != nil {
			compact, err := signer.Sign(jsonData)
			if err != nil {
				return nil, fmt.Errorf("failed to sign LoTE %d: %w", i, err)
			}
			if err := os.WriteFile(outputPath+".jws", []byte(compact), 0640); err != nil {
				return nil, fmt.Errorf("failed to write signed LoTE: %w", err)
			}
			if pl != nil && pl.Logger != nil {
				pl.Logger.Info("Published signed LoTE",
					logging.F("path", outputPath+".jws"),
					logging.F("entities", len(lote.TrustedEntities)))
			}
		}

		// Always write unsigned JSON too
		if err := os.WriteFile(outputPath, jsonData, 0640); err != nil {
			return nil, fmt.Errorf("failed to write LoTE: %w", err)
		}
		if pl != nil && pl.Logger != nil {
			pl.Logger.Info("Published LoTE",
				logging.F("path", outputPath),
				logging.F("entities", len(lote.TrustedEntities)))
		}
	}

	return ctx, nil
}

// loteFilename determines the output filename for a LoTE document.
// Prefers distribution point URIs, then territory, then index-based.
func loteFilename(lote *etsi119602.ListOfTrustedEntities, index int) string {
	// Try distribution points first
	if len(lote.SchemeInformation.DistributionPoints) > 0 {
		u, err := url.Parse(lote.SchemeInformation.DistributionPoints[0])
		if err == nil && u.Path != "" {
			base := filepath.Base(u.Path)
			if base != "" && base != "." && base != "/" {
				// Replace extension with .json
				ext := filepath.Ext(base)
				if ext != "" {
					base = base[:len(base)-len(ext)]
				}
				return base + ".json"
			}
		}
	}

	// Fall back to territory
	if lote.SchemeInformation.Territory != "" {
		return fmt.Sprintf("lote-%s.json", lote.SchemeInformation.Territory)
	}

	return fmt.Sprintf("lote-%d.json", index)
}

// createLoTESigner creates a JWS signer from the remaining publish-lote arguments.
// Returns nil signer if no signing args provided.
// Supports "jades:false" argument to disable JAdES-B-B headers.
func createLoTESigner(args []string) (jws.JSONSigner, error) {
	if len(args) == 0 {
		return nil, nil
	}

	// Check for jades:false in any argument position and filter it out
	jadesEnabled := true
	var filteredArgs []string
	for _, arg := range args {
		if arg == "jades:false" {
			jadesEnabled = false
		} else {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	args = filteredArgs

	if len(args) == 0 {
		return nil, nil
	}

	// PKCS#11: first arg starts with "pkcs11:"
	if strings.HasPrefix(args[0], "pkcs11:") {
		config := dsig.ExtractPKCS11Config(args[0])
		if config == nil {
			return nil, fmt.Errorf("invalid PKCS#11 URI: %s", args[0])
		}
		keyLabel := "default-key"
		certLabel := "default-cert"
		if len(args) >= 2 {
			keyLabel = args[1]
		}
		if len(args) >= 3 {
			certLabel = args[2]
		}
		signer := jws.NewPKCS11Signer(config, keyLabel, certLabel)
		if len(args) >= 4 {
			signer.SetKeyID(args[3])
		}
		signer.SetJAdES(jadesEnabled)
		return signer, nil
	}

	// File-based: need cert and key paths
	if len(args) >= 2 {
		s, err := jws.NewFileSigner(args[0], args[1])
		if err != nil {
			return nil, fmt.Errorf("failed to create JWS signer: %w", err)
		}
		s.SetJAdES(jadesEnabled)
		return s, nil
	}

	return nil, fmt.Errorf("JWS signing requires cert and key paths, or a pkcs11: URI")
}
