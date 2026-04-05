package pipeline

import (
	"fmt"
	"strings"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/jws"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

// Load is a unified pipeline step that loads trust lists from a URL or file path,
// automatically detecting the format (TSL/XML or LoTE/JSON) based on:
// 1. File extension (.xml → TSL, .json → LoTE)
// 2. HTTP Content-Type header
// 3. Content probing (first non-whitespace byte)
//
// For TSL, it delegates to LoadTSL which handles references and builds a tree.
// For LoTE, it delegates to LoadLoTE which supports JWS verification.
//
// Usage in pipeline YAML:
//
//   - load:
//   - https://example.com/tsl.xml              # Auto-detected as TSL
//   - load:
//   - https://example.com/lote.json            # Auto-detected as LoTE
//   - load:
//   - [url_or_path, /path/to/cert.pem]         # LoTE with JWS verification
//
// To force a specific format, use the explicit steps:
//   - load-tsl: ...
//   - load-lote: ...
func Load(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("load requires at least 1 argument: URL or file path")
	}

	location := args[0]

	// Determine fetch options from context
	var fetchOpts *FetchRawOptions
	if ctx.TSLFetchOptions != nil {
		fetchOpts = &FetchRawOptions{
			UserAgent: ctx.TSLFetchOptions.UserAgent,
			Timeout:   ctx.TSLFetchOptions.Timeout,
		}
	}

	// Fetch raw content to detect format
	data, contentType, err := FetchRaw(location, fetchOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", location, err)
	}

	// Detect format
	format := DetectFormat(location, contentType, data)
	if pl != nil && pl.Logger != nil {
		LogFormat(pl.Logger, location, format)
	}

	switch format {
	case FormatTSL:
		// Delegate to TSL loader
		return LoadTSL(pl, ctx, args...)

	case FormatLoTE:
		// Parse LoTE directly from the fetched data
		return loadLoTEFromData(pl, ctx, location, data, args)

	default:
		return nil, fmt.Errorf("unable to detect format for %s (not XML or JSON)", location)
	}
}

// loadLoTEFromData parses LoTE from already-fetched data, handling JWS if needed.
func loadLoTEFromData(pl *Pipeline, ctx *Context, location string, data []byte, args []string) (*Context, error) {
	var lote *etsi119602.ListOfTrustedEntities
	var err error

	// Check for JWS verification cert (second arg)
	var verifier jws.JSONVerifier
	if len(args) >= 2 {
		v, verr := jws.NewCertFileVerifier(args[1])
		if verr != nil {
			return nil, fmt.Errorf("failed to create JWS verifier from %s: %w", args[1], verr)
		}
		verifier = v
	}

	// Try parsing as plain JSON first
	lote, err = etsi119602.ParseLoTE(data)
	if err != nil {
		// If plain JSON parse fails and we have a verifier, try as JWS
		if verifier != nil {
			payload, verifyErr := verifier.Verify(string(data))
			if verifyErr != nil {
				return nil, fmt.Errorf("failed to verify JWS for %s: %w", location, verifyErr)
			}
			lote, err = etsi119602.ParseLoTE(payload)
			if err != nil {
				return nil, fmt.Errorf("failed to parse verified LoTE from %s: %w", location, err)
			}
		} else {
			// Try as JWS compact serialization (unsigned verification just extracts payload)
			if isJWSCompact(data) {
				return nil, fmt.Errorf("LoTE at %s appears to be JWS-signed but no verification cert provided", location)
			}
			return nil, fmt.Errorf("failed to parse LoTE from %s: %w", location, err)
		}
	}

	// If we got a LoTE but also have a verifier, the data was plain JSON
	// but we should warn that verification was requested but not applicable
	if lote != nil && verifier != nil && err == nil {
		if pl != nil && pl.Logger != nil {
			pl.Logger.Debug("LoTE parsed as plain JSON, JWS verification cert was provided but not needed",
				logging.F("source", location))
		}
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Loaded LoTE",
			logging.F("source", location),
			logging.F("entities", len(lote.TrustedEntities)),
			logging.F("territory", lote.SchemeInformation.Territory))
	}

	ctx.AddLoTE(lote)
	return ctx, nil
}

// isJWSCompact checks if data looks like a JWS compact serialization.
func isJWSCompact(data []byte) bool {
	s := strings.TrimSpace(string(data))
	// JWS compact has exactly 2 dots: header.payload.signature
	parts := strings.Split(s, ".")
	return len(parts) == 3 && len(parts[0]) > 0 && len(parts[2]) > 0
}
