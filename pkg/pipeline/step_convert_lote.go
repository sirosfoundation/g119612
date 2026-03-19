package pipeline

import (
	"fmt"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

// ConvertTSLToLoTE converts all TSLs on the context stack to LoTE format
// and pushes the results onto ctx.LoTEs. The original TSLs remain on the stack.
//
// Usage in pipeline YAML:
//
//   - load:
//   - https://example.com/tsl.xml
//   - convert-to-lote:
func ConvertTSLToLoTE(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if ctx.TSLs == nil || ctx.TSLs.Size() == 0 {
		return nil, fmt.Errorf("no TSLs in context to convert")
	}

	ctx.EnsureLoTEs()
	tsls := ctx.TSLs.ToSlice()
	converted := 0

	for _, tsl := range tsls {
		lote := etsi119602.FromTSL(tsl)
		ctx.LoTEs.Push(lote)
		converted++
	}

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Converted TSLs to LoTE",
			logging.F("tsl_count", len(tsls)),
			logging.F("lote_count", converted))
	}

	return ctx, nil
}

func init() {
	RegisterFunction("convert-to-lote", ConvertTSLToLoTE)
}
