package pipeline

import (
	"fmt"

	"github.com/sirosfoundation/g119612/pkg/logging"
)

// ValidateLoTE validates all LoTE documents on the context stack without publishing.
// This is useful for dry-run pipelines to check structural correctness.
//
// Usage in pipeline YAML:
//
//   - generate-lote:
//   - /path/to/data
//   - validate-lote:
func ValidateLoTE(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if ctx.LoTEs == nil || ctx.LoTEs.Size() == 0 {
		return nil, fmt.Errorf("no LoTEs in context to validate")
	}

	lotes := ctx.LoTEs.ToSlice()
	for i, lote := range lotes {
		if err := lote.Validate(); err != nil {
			return nil, fmt.Errorf("LoTE %d validation failed: %w", i, err)
		}
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("All LoTEs validated successfully",
			logging.F("count", len(lotes)))
	}

	return ctx, nil
}

// ValidateLoTL validates all LoTL documents on the context stack without publishing.
//
// Usage in pipeline YAML:
//
//   - generate-lotl:
//   - /path/to/data
//   - validate-lotl:
func ValidateLoTL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if ctx.LoTLs == nil || ctx.LoTLs.Size() == 0 {
		return nil, fmt.Errorf("no LoTLs in context to validate")
	}

	lotls := ctx.LoTLs.ToSlice()
	for i, lotl := range lotls {
		if err := lotl.Validate(); err != nil {
			return nil, fmt.Errorf("LoTL %d validation failed: %w", i, err)
		}
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("All LoTLs validated successfully",
			logging.F("count", len(lotls)))
	}

	return ctx, nil
}

func init() {
	RegisterFunction("validate-lote", ValidateLoTE)
	RegisterFunction("validate-lotl", ValidateLoTL)
}
