package pipeline

import (
	"fmt"

	"github.com/sirosfoundation/g119612/pkg/dsig"
	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/jws"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

// LoadLoTL loads a LoTL (List of Trusted Lists) from a URL or file path
// and pushes it onto ctx.LoTLs. Automatically detects JSON vs XML format
// based on file extension and content type.
//
// Usage in pipeline YAML:
//
//   - load-lotl:
//   - https://example.com/lotl.json                            # JSON
//   - load-lotl:
//   - https://example.com/lotl.xml                             # XML (auto-detected)
//   - load-lotl:
//   - /path/to/lotl.json
//   - load-lotl:
//   - [url_or_path, /path/to/trusted-cert.pem]                 # with JWS/XAdES verification
func LoadLoTL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("load-lotl requires 1 argument: URL or file path")
	}

	location := args[0]

	var fetchOpts *etsi119602.FetchOptions
	if ctx.TSLFetchOptions != nil {
		fetchOpts = &etsi119602.FetchOptions{
			UserAgent: ctx.TSLFetchOptions.UserAgent,
			Timeout:   ctx.TSLFetchOptions.Timeout,
		}
	}

	// Check for verification cert (second arg)
	var verifierCertPath string
	if len(args) >= 2 {
		verifierCertPath = args[1]
	}

	// Try loading with auto-detection (handles both JSON and XML)
	lotl, err := etsi119602.FetchLoTL(location, fetchOpts)
	if err != nil {
		// If load fails and we have a verifier cert, try JWS verification
		if verifierCertPath != "" {
			verifier, vErr := jws.NewCertFileVerifier(verifierCertPath)
			if vErr != nil {
				return nil, fmt.Errorf("failed to create JWS verifier from %s: %w", verifierCertPath, vErr)
			}
			// Fetch raw and verify as JWS, then parse as LoTL
			data, fErr := etsi119602.FetchRaw(location, fetchOpts)
			if fErr != nil {
				return nil, fmt.Errorf("failed to fetch LoTL from %s: %w", location, fErr)
			}
			payload, vErr := verifier.Verify(string(data))
			if vErr != nil {
				return nil, fmt.Errorf("JWS verification failed for LoTL %s: %w", location, vErr)
			}
			lotl, err = etsi119602.ParseLoTL(payload)
			if err != nil {
				return nil, fmt.Errorf("failed to parse verified LoTL from %s: %w", location, err)
			}
		} else {
			return nil, fmt.Errorf("failed to load LoTL from %s: %w", location, err)
		}
	}

	// If we loaded XML and have a verification cert, verify XAdES signature
	if verifierCertPath != "" && isXMLLocation(location) {
		certs, vErr := loadVerificationCerts(verifierCertPath)
		if vErr != nil {
			return nil, fmt.Errorf("failed to load verification certs from %s: %w", verifierCertPath, vErr)
		}
		data, fErr := etsi119602.FetchRaw(location, fetchOpts)
		if fErr != nil {
			return nil, fmt.Errorf("failed to fetch XML for verification: %w", fErr)
		}
		if _, err := dsig.VerifyXMLSignature(data, certs); err != nil {
			return nil, fmt.Errorf("XAdES verification failed for %s: %w", location, err)
		}
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Loaded LoTL",
			logging.F("source", location),
			logging.F("pointers", len(lotl.PointersToOtherLoTEs)),
			logging.F("territory", lotl.SchemeInformation.Territory))
	}

	ctx.EnsureLoTLs()
	ctx.LoTLs.Push(lotl)
	return ctx, nil
}

func init() {
	RegisterFunction("load-lotl", LoadLoTL)
}
