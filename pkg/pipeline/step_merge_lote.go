package pipeline

import (
	"fmt"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

// MergeLoTEs merges all LoTEs on the context stack into a single LoTE.
// The scheme information is taken from the first LoTE; trusted entities from
// all LoTEs are concatenated. Pointers are also merged.
//
// Usage in pipeline YAML:
//
//   - load-lote:
//   - https://example.com/lote-se.json
//   - load-lote:
//   - https://example.com/lote-de.json
//   - merge-lote:
func MergeLoTEs(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if ctx.LoTEs == nil || ctx.LoTEs.Size() == 0 {
		return nil, fmt.Errorf("no LoTEs in context to merge")
	}

	lotes := ctx.LoTEs.ToSlice()
	if len(lotes) < 2 {
		// Nothing to merge
		return ctx, nil
	}

	merged := &etsi119602.ListOfTrustedEntities{
		ListAndSchemeInformation: lotes[0].ListAndSchemeInformation,
	}
	// Clear pointers since the loop below re-collects them from all LoTEs including the first
	merged.ListAndSchemeInformation.PointersToOtherLoTE = nil

	for _, lote := range lotes {
		merged.TrustedEntitiesList = append(merged.TrustedEntitiesList, lote.TrustedEntitiesList...)
		merged.ListAndSchemeInformation.PointersToOtherLoTE = append(merged.ListAndSchemeInformation.PointersToOtherLoTE, lote.ListAndSchemeInformation.PointersToOtherLoTE...)
	}

	// Replace the stack with just the merged LoTE
	ctx.LoTEs.Clear()
	ctx.LoTEs.Push(merged)

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Merged LoTEs",
			logging.F("sources", len(lotes)),
			logging.F("entities", len(merged.TrustedEntitiesList)))
	}

	return ctx, nil
}

func init() {
	RegisterFunction("merge-lote", MergeLoTEs)
}
