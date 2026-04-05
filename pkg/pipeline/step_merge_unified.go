package pipeline

import (
	"fmt"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

// Merge is a unified pipeline step that merges all trust lists of the same type
// in the context into a single trust list per type.
//
// For LoTEs:
//   - Merges all LoTEs into one, taking scheme info from the first
//   - Concatenates trusted entities and pointers from all sources
//
// For TSLs:
//   - Merges all TSLs into one, taking scheme info from the first
//   - Concatenates trust service providers from all sources
//
// Usage in pipeline YAML:
//
//   - load:
//   - https://example.com/list-se.json
//   - load:
//   - https://example.com/list-de.json
//   - merge:
//
// To merge only a specific format, use the explicit steps:
//   - merge-tsl: ...
//   - merge-lote: ...
func Merge(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	hasTSLs := ctx.TSLs != nil && ctx.TSLs.Size() > 1
	hasLoTEs := ctx.LoTEs != nil && ctx.LoTEs.Size() > 1

	if !hasTSLs && !hasLoTEs {
		// Nothing to merge (need at least 2 of one type)
		if ctx.TSLs != nil && ctx.TSLs.Size() == 1 {
			if pl != nil && pl.Logger != nil {
				pl.Logger.Debug("Only one TSL in context, nothing to merge")
			}
			return ctx, nil
		}
		if ctx.LoTEs != nil && ctx.LoTEs.Size() == 1 {
			if pl != nil && pl.Logger != nil {
				pl.Logger.Debug("Only one LoTE in context, nothing to merge")
			}
			return ctx, nil
		}
		return nil, fmt.Errorf("no trust lists in context to merge")
	}

	// Merge TSLs if present
	if hasTSLs {
		ctx = mergeTSLs(pl, ctx)
	}

	// Merge LoTEs if present
	if hasLoTEs {
		_, err := MergeLoTEs(pl, ctx, args...)
		if err != nil {
			return nil, fmt.Errorf("merge LoTEs: %w", err)
		}
	}

	return ctx, nil
}

// mergeTSLs merges all TSLs in the context into a single TSL.
func mergeTSLs(pl *Pipeline, ctx *Context) *Context {
	tsls := ctx.TSLs.ToSlice()
	if len(tsls) < 2 {
		return ctx
	}

	// Take scheme information from the first TSL
	merged := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TSLTagAttr:           tsls[0].StatusList.TSLTagAttr,
			IdAttr:               tsls[0].StatusList.IdAttr,
			TslSchemeInformation: tsls[0].StatusList.TslSchemeInformation,
		},
	}

	// Merge trust service providers from all TSLs
	var allProviders []*etsi119612.TSPType
	for _, tsl := range tsls {
		if tsl.StatusList.TslTrustServiceProviderList != nil {
			allProviders = append(allProviders, tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider...)
		}
	}

	merged.StatusList.TslTrustServiceProviderList = &etsi119612.TrustServiceProviderListType{
		TslTrustServiceProvider: allProviders,
	}

	// Also merge pointers to other TSLs
	var allPointers []*etsi119612.OtherTSLPointerType
	for _, tsl := range tsls {
		if tsl.StatusList.TslSchemeInformation != nil &&
			tsl.StatusList.TslSchemeInformation.TslPointersToOtherTSL != nil {
			allPointers = append(allPointers, tsl.StatusList.TslSchemeInformation.TslPointersToOtherTSL.TslOtherTSLPointer...)
		}
	}
	if len(allPointers) > 0 && merged.StatusList.TslSchemeInformation != nil {
		merged.StatusList.TslSchemeInformation.TslPointersToOtherTSL = &etsi119612.OtherTSLPointersType{
			TslOtherTSLPointer: allPointers,
		}
	}

	// Replace the stack with just the merged TSL
	ctx.TSLs.Clear()
	ctx.TSLs.Push(merged)

	if pl != nil && pl.Logger != nil {
		pl.Logger.Info("Merged TSLs",
			logging.F("sources", len(tsls)),
			logging.F("providers", len(allProviders)))
	}

	return ctx
}

// MergeTSLs merges all TSLs on the context stack into a single TSL.
// Exported for explicit use via "merge-tsl:" step.
func MergeTSLs(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if ctx.TSLs == nil || ctx.TSLs.Size() == 0 {
		return nil, fmt.Errorf("no TSLs in context to merge")
	}
	return mergeTSLs(pl, ctx), nil
}
