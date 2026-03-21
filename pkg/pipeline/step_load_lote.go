package pipeline

import (
	"fmt"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/jws"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

// LoadLoTE loads a LoTE (List of Trusted Entities) from a URL or file path
// and pushes it onto ctx.LoTEs.
//
// Usage in pipeline YAML:
//
//   - load-lote:
//   - https://example.com/lote.json
//   - load-lote:
//   - /path/to/lote.json
//   - load-lote:
//   - [url_or_path, /path/to/trusted-cert.pem]   # with JWS verification
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

	// Check for JWS verification cert (second arg)
	var verifier jws.JSONVerifier
	if len(args) >= 2 {
		v, err := jws.NewCertFileVerifier(args[1])
		if err != nil {
			return nil, fmt.Errorf("failed to create JWS verifier from %s: %w", args[1], err)
		}
		verifier = v
	}

	lote, err := etsi119602.FetchLoTE(location, fetchOpts)
	if err != nil {
		// If the plain JSON load fails and we have a verifier, try loading as JWS
		if verifier != nil {
			lote, err = etsi119602.FetchAndVerifyLoTE(location, fetchOpts, verifier)
			if err != nil {
				return nil, fmt.Errorf("failed to load and verify LoTE from %s: %w", location, err)
			}
		} else {
			return nil, fmt.Errorf("failed to load LoTE from %s: %w", location, err)
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
