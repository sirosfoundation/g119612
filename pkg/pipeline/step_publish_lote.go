package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirosfoundation/g119612/pkg/jws"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/sirosfoundation/g119612/pkg/validation"
)

// PublishLoTE writes LoTE documents from ctx.LoTEs to JSON files,
// optionally signing them with JWS.
//
// Usage in pipeline YAML:
//
//   - publish-lote:
//   - /path/to/output/dir                          # unsigned JSON
//   - publish-lote:
//   - ["/path/to/dir", "/cert.pem", "/key.pem"]   # JWS-signed
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

	// Create JWS signer if cert+key provided
	var signer jws.JSONSigner
	if len(args) >= 3 {
		certPath := args[1]
		keyPath := args[2]
		s, err := jws.NewFileSigner(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create JWS signer: %w", err)
		}
		signer = s
	}

	if ctx.LoTEs == nil || ctx.LoTEs.Size() == 0 {
		if pl != nil && pl.Logger != nil {
			pl.Logger.Warn("No LoTEs in context to publish")
		}
		return ctx, nil
	}

	lotes := ctx.LoTEs.ToSlice()
	for i, lote := range lotes {
		// Generate filename from territory or index
		filename := fmt.Sprintf("lote-%d.json", i)
		if lote.SchemeInformation.Territory != "" {
			filename = fmt.Sprintf("lote-%s.json", lote.SchemeInformation.Territory)
		}

		jsonData, err := lote.MarshalIndent()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal LoTE %d: %w", i, err)
		}

		outputPath := filepath.Join(outputDir, filename)

		if signer != nil {
			// Sign and write as JWS
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
