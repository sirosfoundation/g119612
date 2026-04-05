package pipeline

import (
	"fmt"

	"github.com/sirosfoundation/g119612/pkg/logging"
)

// Publish is a unified pipeline step that publishes all trust lists in the context
// to the specified output directory. It automatically handles both TSLs and LoTEs:
//
//   - TSLs are written as XML files (optionally signed with XML-DSIG)
//   - LoTEs are written as JSON files (optionally signed with JWS)
//
// The step delegates to PublishTSL and/or PublishLoTE based on what's in the context.
//
// Usage in pipeline YAML:
//
//   - publish:
//   - /path/to/output/dir                                              # Unsigned
//   - publish:
//   - ["/path/to/dir", "/cert.pem", "/key.pem"]                       # Signed (TSL=XML-DSIG, LoTE=JWS)
//   - publish:
//   - ["/path/to/dir", "pkcs11:module=/path;pin=1234", "key", "cert"] # PKCS#11 signing
//
// To publish only a specific format, use the explicit steps:
//   - publish-tsl: ...   (or the alias: publish-xml:)
//   - publish-lote: ...  (or the alias: publish-json:)
func Publish(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("publish requires at least 1 argument: output directory")
	}

	hasTSLs := ctx.TSLs != nil && ctx.TSLs.Size() > 0
	hasLoTEs := ctx.LoTEs != nil && ctx.LoTEs.Size() > 0

	if !hasTSLs && !hasLoTEs {
		return nil, fmt.Errorf("no trust lists in context to publish")
	}

	var errs []error

	// Publish TSLs if present
	if hasTSLs {
		if pl != nil && pl.Logger != nil {
			pl.Logger.Info("Publishing TSLs",
				logging.F("count", ctx.TSLs.Size()),
				logging.F("output", args[0]))
		}
		var err error
		ctx, err = PublishTSL(pl, ctx, args...)
		if err != nil {
			errs = append(errs, fmt.Errorf("publish TSL: %w", err))
		}
	}

	// Publish LoTEs if present
	if hasLoTEs {
		if pl != nil && pl.Logger != nil {
			pl.Logger.Info("Publishing LoTEs",
				logging.F("count", ctx.LoTEs.Size()),
				logging.F("output", args[0]))
		}
		var err error
		ctx, err = PublishLoTE(pl, ctx, args...)
		if err != nil {
			errs = append(errs, fmt.Errorf("publish LoTE: %w", err))
		}
	}

	if len(errs) > 0 {
		return ctx, fmt.Errorf("publish errors: %v", errs)
	}

	return ctx, nil
}
