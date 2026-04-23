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

// PublishLoTE writes LoTE and LoTL documents from context to JSON and/or XML files,
// optionally signing them with JWS (JSON) or XAdES (XML).
//
// Publishes both ctx.LoTEs and ctx.LoTLs in a single step.
// This step is registered as both "publish-lote" and "publish-lotl".
//
// By default, signatures use JAdES-B-B profile (ETSI TS 119 182-1).
// Pass "jades:false" as an argument to disable JAdES headers and produce plain JWS.
// Pass "xml" or "xml-only" to also/only produce XML output.
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
//   - ["/path/to/dir", "/cert.pem", "/key.pem", "xml"]                # JSON + XML output
//   - publish-lote:
//   - ["/path/to/dir", "/cert.pem", "/key.pem", "xml-only"]           # XML only
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

	// Check for XML output flags
	wantXML, xmlOnly := parseLoteXMLFlags(args[1:])

	// Create XML signer if producing XML output
	var xmlSigner dsig.XMLSigner
	if wantXML {
		xmlSigner, err = createXMLSigner(filterLoteXMLArgs(args[1:]))
		if err != nil {
			return nil, err
		}
	}

	if ctx.LoTEs == nil || ctx.LoTEs.Size() == 0 {
		if pl != nil && pl.Logger != nil {
			pl.Logger.Warn("No LoTEs in context to publish")
		}
	} else {

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
			filename = fmt.Sprintf("lote-%s-%d.json", lote.ListAndSchemeInformation.SchemeTerritory, i)
		}
		usedFilenames[filename] = true

		// JSON output (unless xml-only)
		if !xmlOnly {
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
						logging.F("entities", len(lote.TrustedEntitiesList)))
				}
			}

			// Always write unsigned JSON too
			if err := os.WriteFile(outputPath, jsonData, 0640); err != nil {
				return nil, fmt.Errorf("failed to write LoTE: %w", err)
			}
			if pl != nil && pl.Logger != nil {
				pl.Logger.Info("Published LoTE (JSON)",
					logging.F("path", outputPath),
					logging.F("entities", len(lote.TrustedEntitiesList)))
			}
		}

		// XML output
		if wantXML {
			xmlFilename := strings.TrimSuffix(filename, ".json") + ".xml"
			xmlPath := filepath.Join(outputDir, xmlFilename)

			xmlData, err := lote.EncodeXML()
			if err != nil {
				return nil, fmt.Errorf("failed to marshal LoTE %d to XML: %w", i, err)
			}

			if xmlSigner != nil {
				signed, err := xmlSigner.Sign(xmlData)
				if err != nil {
					return nil, fmt.Errorf("failed to sign LoTE %d (XAdES): %w", i, err)
				}
				if err := os.WriteFile(xmlPath, signed, 0640); err != nil {
					return nil, fmt.Errorf("failed to write signed LoTE XML: %w", err)
				}
			} else {
				fullXML := append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), xmlData...)
				if err := os.WriteFile(xmlPath, fullXML, 0640); err != nil {
					return nil, fmt.Errorf("failed to write LoTE XML: %w", err)
				}
			}
			if pl != nil && pl.Logger != nil {
				pl.Logger.Info("Published LoTE (XML)",
					logging.F("path", xmlPath),
					logging.F("entities", len(lote.TrustedEntitiesList)))
			}
		}
	}
	} // end LoTEs block

	// Publish LoTLs from context
	if ctx.LoTLs != nil && ctx.LoTLs.Size() > 0 {
		lotls := ctx.LoTLs.ToSlice()
		for i, lotl := range lotls {
			if err := lotl.Validate(); err != nil {
				return nil, fmt.Errorf("LoTL %d failed validation: %w", i, err)
			}
		}

		for i, lotl := range lotls {
			basename := lotlFilename(lotl, i)

			if !xmlOnly {
				jsonData, err := lotl.MarshalIndent()
				if err != nil {
					return nil, fmt.Errorf("failed to marshal LoTL %d to JSON: %w", i, err)
				}

				jsonPath := filepath.Join(outputDir, basename+".json")
				if err := os.WriteFile(jsonPath, jsonData, 0640); err != nil {
					return nil, fmt.Errorf("failed to write LoTL JSON: %w", err)
				}

				if signer != nil {
					compact, err := signer.Sign(jsonData)
					if err != nil {
						return nil, fmt.Errorf("failed to sign LoTL %d (JWS): %w", i, err)
					}
					if err := os.WriteFile(jsonPath+".jws", []byte(compact), 0640); err != nil {
						return nil, fmt.Errorf("failed to write signed LoTL JWS: %w", err)
					}
				}

				if pl != nil && pl.Logger != nil {
					pl.Logger.Info("Published LoTL (JSON)",
						logging.F("path", jsonPath),
						logging.F("pointers", len(lotl.ListAndSchemeInformation.PointersToOtherLoTE)))
				}
			}

			if wantXML {
				xmlData, err := lotl.EncodeXML()
				if err != nil {
					return nil, fmt.Errorf("failed to marshal LoTL %d to XML: %w", i, err)
				}

				xmlPath := filepath.Join(outputDir, basename+".xml")

				if xmlSigner != nil {
					signed, err := xmlSigner.Sign(xmlData)
					if err != nil {
						return nil, fmt.Errorf("failed to sign LoTL %d (XAdES): %w", i, err)
					}
					if err := os.WriteFile(xmlPath, signed, 0640); err != nil {
						return nil, fmt.Errorf("failed to write signed LoTL XML: %w", err)
					}
				} else {
					fullXML := append([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"), xmlData...)
					if err := os.WriteFile(xmlPath, fullXML, 0640); err != nil {
						return nil, fmt.Errorf("failed to write LoTL XML: %w", err)
					}
				}

				if pl != nil && pl.Logger != nil {
					pl.Logger.Info("Published LoTL (XML)",
						logging.F("path", xmlPath),
						logging.F("pointers", len(lotl.ListAndSchemeInformation.PointersToOtherLoTE)))
				}
			}
		}
	}

	return ctx, nil
}

// loteFilename determines the output filename for a LoTE document.
// Prefers distribution point URIs, then territory, then index-based.
func loteFilename(lote *etsi119602.ListOfTrustedEntities, index int) string {
	// Try distribution points first
	if len(lote.ListAndSchemeInformation.DistributionPoints) > 0 {
		u, err := url.Parse(lote.ListAndSchemeInformation.DistributionPoints[0])
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
	if lote.ListAndSchemeInformation.SchemeTerritory != "" {
		return fmt.Sprintf("lote-%s.json", lote.ListAndSchemeInformation.SchemeTerritory)
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
		switch arg {
		case "jades:false":
			jadesEnabled = false
		case "xml", "xml-only", "json-only":
			// Skip format flags
		default:
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

// parseLoteXMLFlags checks args for "xml" or "xml-only" flags.
// Returns (wantXML, xmlOnly).
func parseLoteXMLFlags(args []string) (bool, bool) {
	for _, arg := range args {
		switch arg {
		case "xml-only":
			return true, true
		case "xml":
			return true, false
		}
	}
	return false, false
}

// filterLoteXMLArgs removes LoTE-specific flags from args for XML signer creation.
func filterLoteXMLArgs(args []string) []string {
	var filtered []string
	for _, arg := range args {
		switch arg {
		case "xml", "xml-only", "json-only", "jades:false":
			continue
		default:
			filtered = append(filtered, arg)
		}
	}
	return filtered
}
