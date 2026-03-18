package pipeline

import (
	"fmt"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

// LoadLoTE loads a LoTE (List of Trusted Entities) from a URL or file path
// and pushes it onto ctx.LoTEs.
//
// Usage in pipeline YAML:
//
//	- load-lote:
//	    - https://example.com/lote.json
//	- load-lote:
//	    - /path/to/lote.json
func LoadLoTE(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("load-lote requires 1 argument: URL or file path")
	}

	location := args[0]

	var opts *etsi119602.FetchOptions
	if ctx.TSLFetchOptions != nil {
		opts = &etsi119602.FetchOptions{
			UserAgent: ctx.TSLFetchOptions.UserAgent,
			Timeout:   ctx.TSLFetchOptions.Timeout,
		}
	}

	lote, err := etsi119602.FetchLoTE(location, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load LoTE from %s: %w", location, err)
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
