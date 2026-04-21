package pipeline

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/sirosfoundation/g119612/pkg/dsig"
	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/jws"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

// LoadLoTE loads an ETSI TS 119 602 document (LoTE or LoTL) from a URL or file path.
// Automatically detects JSON vs XML format based on file extension and content type.
// Automatically classifies the document as a LoTE or LoTL based on its scheme type:
//   - LoTL scheme types (containing "/LoTLType/") → pushed onto ctx.LoTLs
//   - All other scheme types → pushed onto ctx.LoTEs
//
// This step is registered as both "load-lote" and "load-lotl" since it handles both.
//
// Usage in pipeline YAML:
//
//   - load-lote:
//   - https://example.com/lote.json                            # JSON LoTE
//   - load-lote:
//   - https://example.com/lotl.xml                             # XML LoTL (auto-classified)
//   - load-lote:
//   - [url_or_path, /path/to/trusted-cert.pem]                 # with JWS/XAdES verification
func LoadLoTE(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("load-lote requires 1 argument: URL or file path")
	}

	location := args[0]

	var fetchOpts *etsi119602.FetchOptions
	if ctx.TSLFetchOptions != nil {
		fetchOpts = &etsi119602.FetchOptions{
			UserAgent: ctx.TSLFetchOptions.UserAgent,
			Timeout:   ctx.TSLFetchOptions.Timeout,
		}
	}

	// Check for verification cert (second arg) — used for JWS or XAdES verification
	var verifierCertPath string
	if len(args) >= 2 {
		verifierCertPath = args[1]
	}

	// Try loading with auto-detection (handles both JSON and XML)
	lote, err := etsi119602.FetchLoTE(location, fetchOpts)
	if err != nil {
		// If load fails and we have a verifier cert, try JWS verification
		if verifierCertPath != "" {
			verifier, vErr := jws.NewCertFileVerifier(verifierCertPath)
			if vErr != nil {
				return nil, fmt.Errorf("failed to create JWS verifier from %s: %w", verifierCertPath, vErr)
			}
			lote, err = etsi119602.FetchAndVerifyLoTE(location, fetchOpts, verifier)
			if err != nil {
				return nil, fmt.Errorf("failed to load and verify LoTE from %s: %w", location, err)
			}
		} else {
			return nil, fmt.Errorf("failed to load LoTE from %s: %w", location, err)
		}
	}

	// If we loaded XML and have a verification cert, verify XAdES signature
	if verifierCertPath != "" && isXMLLocation(location) {
		certs, vErr := loadVerificationCerts(verifierCertPath)
		if vErr != nil {
			return nil, fmt.Errorf("failed to load verification certs from %s: %w", verifierCertPath, vErr)
		}
		if err := verifyLoTEXMLSignature(location, fetchOpts, certs); err != nil {
			return nil, fmt.Errorf("XAdES verification failed for %s: %w", location, err)
		}
	}

	// Auto-classify: LoTL scheme types go to the LoTLs stack, others to LoTEs
	if etsi119602.IsLoTLSchemeType(lote.SchemeInformation.SchemeType) {
		lotl := etsi119602.LoTLFromLoTE(lote)
		ctx.EnsureLoTLs()
		ctx.LoTLs.Push(lotl)

		if pl != nil && pl.Logger != nil {
			pl.Logger.Info("Loaded LoTL",
				logging.F("source", location),
				logging.F("pointers", len(lotl.PointersToOtherLoTEs)),
				logging.F("territory", lotl.SchemeInformation.Territory))
		}
	} else {
		ctx.AddLoTE(lote)

		if pl != nil && pl.Logger != nil {
			pl.Logger.Info("Loaded LoTE",
				logging.F("source", location),
				logging.F("entities", len(lote.TrustedEntities)),
				logging.F("territory", lote.SchemeInformation.Territory))
		}
	}

	return ctx, nil
}

// isXMLLocation returns true if the location looks like it points to an XML resource.
func isXMLLocation(location string) bool {
	loc := strings.ToLower(location)
	// Strip query/fragment
	if idx := strings.IndexAny(loc, "?#"); idx >= 0 {
		loc = loc[:idx]
	}
	return strings.HasSuffix(loc, ".xml")
}

// loadVerificationCerts loads X.509 certificates from a PEM or DER file.
func loadVerificationCerts(certPath string) ([]*x509.Certificate, error) {
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert file %s: %w", certPath, err)
	}

	var certs []*x509.Certificate
	// Try PEM first
	rest := data
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse PEM certificate: %w", err)
			}
			certs = append(certs, cert)
		}
	}
	if len(certs) > 0 {
		return certs, nil
	}

	// Fall back to DER
	cert, err := x509.ParseCertificate(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from %s: %w", certPath, err)
	}
	return []*x509.Certificate{cert}, nil
}

// verifyLoTEXMLSignature fetches raw XML and verifies its XAdES/XML-DSIG signature.
func verifyLoTEXMLSignature(location string, opts *etsi119602.FetchOptions, certs []*x509.Certificate) error {
	data, err := etsi119602.FetchRaw(location, opts)
	if err != nil {
		return fmt.Errorf("failed to fetch XML for verification: %w", err)
	}
	_, err = dsig.VerifyXMLSignature(data, certs)
	return err
}

// IncrementLoTESequence increments the sequence number on all LoTEs in context.
//
// Usage in pipeline YAML:
//
//   - increment-lote-sequence:
func IncrementLoTESequence(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if ctx.LoTEs == nil || ctx.LoTEs.Size() == 0 {
		return nil, fmt.Errorf("no LoTEs in context")
	}

	lotes := ctx.LoTEs.ToSlice()
	for _, lote := range lotes {
		lote.SchemeInformation.SequenceNumber++
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Incremented LoTE sequence numbers",
			logging.F("count", len(lotes)))
	}

	return ctx, nil
}
