package pipeline

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/sirosfoundation/g119612/pkg/validation"
)

// PublishLoTL writes LoTL documents from ctx.LoTLs to JSON and/or XML files,
// optionally signing them with JWS (JSON) or XAdES (XML).
//
// Usage in pipeline YAML:
//
//   - publish-lotl:
//   - /path/to/output/dir                                    # unsigned JSON + XML
//   - publish-lotl:
//   - ["/path/to/dir", "/cert.pem", "/key.pem"]              # JAdES-signed JSON + XAdES-signed XML
//   - publish-lotl:
//   - ["/path/to/dir", "/cert.pem", "/key.pem", "json-only"] # JAdES-signed JSON only
//   - publish-lotl:
//   - ["/path/to/dir", "/cert.pem", "/key.pem", "xml-only"]  # XAdES-signed XML only
func PublishLoTL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("publish-lotl requires at least 1 argument: output directory")
	}

	outputDir := args[0]
	if err := validation.ValidateOutputDirectory(outputDir); err != nil {
		return nil, fmt.Errorf("invalid output directory: %w", err)
	}
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Parse format flags
	format := parseLotlFormat(args[1:])

	// Create JWS signer for JSON output
	jwsSigner, err := createLoTESigner(filterSignerArgs(args[1:]))
	if err != nil {
		return nil, err
	}

	// Create XML signer for XML output
	xmlSigner, err := createXMLSigner(filterSignerArgs(args[1:]))
	if err != nil {
		return nil, err
	}

	if ctx.LoTLs == nil || ctx.LoTLs.Size() == 0 {
		if pl != nil && pl.Logger != nil {
			pl.Logger.Warn("No LoTLs in context to publish")
		}
		return ctx, nil
	}

	lotls := ctx.LoTLs.ToSlice()
	for i, lotl := range lotls {
		if err := lotl.Validate(); err != nil {
			return nil, fmt.Errorf("LoTL %d failed validation: %w", i, err)
		}
	}

	for i, lotl := range lotls {
		basename := lotlFilename(lotl, i)

		if format.json {
			lote := lotl.ToLoTE()
			jsonData, err := lote.MarshalIndent()
			if err != nil {
				return nil, fmt.Errorf("failed to marshal LoTL %d to JSON: %w", i, err)
			}

			jsonPath := filepath.Join(outputDir, basename+".json")
			if err := os.WriteFile(jsonPath, jsonData, 0640); err != nil {
				return nil, fmt.Errorf("failed to write LoTL JSON: %w", err)
			}

			if jwsSigner != nil {
				compact, err := jwsSigner.Sign(jsonData)
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
					logging.F("pointers", len(lotl.PointersToOtherLoTEs)))
			}
		}

		if format.xml {
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
					logging.F("pointers", len(lotl.PointersToOtherLoTEs)))
			}
		}
	}

	return ctx, nil
}

type lotlFormat struct {
	json bool
	xml  bool
}

func parseLotlFormat(args []string) lotlFormat {
	for _, arg := range args {
		switch arg {
		case "json-only":
			return lotlFormat{json: true, xml: false}
		case "xml-only":
			return lotlFormat{json: false, xml: true}
		}
	}
	return lotlFormat{json: true, xml: true}
}

// filterSignerArgs removes format flags from the args to pass to signer creation.
func filterSignerArgs(args []string) []string {
	var filtered []string
	for _, arg := range args {
		switch arg {
		case "json-only", "xml-only":
			continue
		default:
			filtered = append(filtered, arg)
		}
	}
	return filtered
}

func lotlFilename(lotl *etsi119602.ListOfTrustedLists, index int) string {
	if lotl.SchemeInformation.Territory != "" {
		return fmt.Sprintf("list_of_trusted_lists-%s", lotl.SchemeInformation.Territory)
	}
	if index == 0 {
		return "list_of_trusted_lists"
	}
	return fmt.Sprintf("list_of_trusted_lists-%d", index)
}

func init() {
	RegisterFunction("publish-lotl", PublishLoTL)
}
