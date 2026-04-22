package pipeline

import (
	"fmt"

	"github.com/sirosfoundation/g119612/pkg/logging"
)

// Validate validates all ETSI TS 119 602 documents (LoTEs and LoTLs) on the
// context stacks without publishing. This is useful for dry-run pipelines to
// check structural correctness.
//
// This step is registered as "validate-lote", "validate-lotl", and "validate"
// since it handles all document types.
//
// Usage in pipeline YAML:
//
//   - generate-lote:
//   - /path/to/data
//   - validate-lote:          # validates both LoTEs and LoTLs in context
func Validate(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	loteCount := 0
	lotlCount := 0

	if ctx.LoTEs != nil && ctx.LoTEs.Size() > 0 {
		lotes := ctx.LoTEs.ToSlice()
		for i, lote := range lotes {
			if err := lote.Validate(); err != nil {
				return nil, fmt.Errorf("LoTE %d validation failed: %w", i, err)
			}
		}
		loteCount = len(lotes)
	}

	if ctx.LoTLs != nil && ctx.LoTLs.Size() > 0 {
		lotls := ctx.LoTLs.ToSlice()
		for i, lotl := range lotls {
			if err := lotl.Validate(); err != nil {
				return nil, fmt.Errorf("LoTL %d validation failed: %w", i, err)
			}
		}
		lotlCount = len(lotls)
	}

	if loteCount == 0 && lotlCount == 0 {
		return nil, fmt.Errorf("no LoTEs or LoTLs in context to validate")
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("All documents validated successfully",
			logging.F("lotes", loteCount),
			logging.F("lotls", lotlCount))
	}

	return ctx, nil
}

// ValidateLoTE is an alias for Validate. Validates both LoTEs and LoTLs.
var ValidateLoTE = Validate

// ValidateLoTL is an alias for Validate. Validates both LoTEs and LoTLs.
var ValidateLoTL = Validate

func init() {
	RegisterFunction("validate", Validate)
	RegisterFunction("validate-lote", ValidateLoTE)
	RegisterFunction("validate-lotl", ValidateLoTL)
}
