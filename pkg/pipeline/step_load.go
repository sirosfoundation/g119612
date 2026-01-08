package pipeline

import (
	"fmt"
	"strings"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
	"github.com/sirosfoundation/g119612/pkg/utils"
	"github.com/sirosfoundation/g119612/pkg/validation"
)

// LoadTSL is a pipeline step that loads Trust Service Lists (TSLs) from a URL or file path,
// builds a hierarchical TSL tree structure, and adds it to the pipeline context. It also
// maintains a backward-compatible flat stack of TSLs for legacy code.
//
// The step supports loading TSLs from files or HTTP/HTTPS URLs, with automatic content
// negotiation and reference handling. It uses the TSLFetchOptions in the context for
// request configuration (user-agent, timeout, reference depth, etc.).
//
// Parameters:
//   - pl: The pipeline instance for logging and configuration
//   - ctx: The pipeline context to update with loaded TSLs
//   - args: String arguments, where:
//   - args[0]: Required - URL or file path to the root TSL
//   - args[1]: Optional - Filter expression for including specific TSLs (not implemented yet)
//
// Returns:
//   - *Context: Updated context with the loaded TSL tree and legacy TSL stack
//   - error: Non-nil if loading fails
//
// Example usage in pipeline configuration:
//   - load:
//   - https://example.com/tsl.xml
//
// Or with a local file:
//   - load:
//   - /path/to/local/tsl.xml
//
// The loaded TSL tree structure represents the hierarchical relationship between the root TSL
// and its referenced TSLs, allowing for more efficient traversal and operations on the tree.
func LoadTSL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	if len(args) < 1 {
		return ctx, fmt.Errorf("missing argument: URL or file path")
	}

	url := args[0]
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "file://" + url
	}

	// Validate the URL before processing
	if err := validation.ValidateURL(url, validation.TSLURLOptions()); err != nil {
		return ctx, fmt.Errorf("invalid TSL URL: %w", err)
	}

	// Parse optional filter argument
	var filter string
	if len(args) > 1 {
		filter = args[1]
		pl.Logger.Debug("TSL filter provided", logging.F("filter", filter))
		// Note: Filter implementation will be added in a future update
	}

	// Ensure the TSLFetchOptions are initialized with default values if not set
	ctx.EnsureTSLFetchOptions()

	pl.Logger.Debug("Loading TSL",
		logging.F("url", url),
		logging.F("user-agent", ctx.TSLFetchOptions.UserAgent),
		logging.F("timeout", ctx.TSLFetchOptions.Timeout),
		logging.F("max-depth", ctx.TSLFetchOptions.MaxDereferenceDepth),
		logging.F("accept", ctx.TSLFetchOptions.AcceptHeaders))

	tsls, err := etsi119612.FetchTSLWithReferencesAndOptions(url, *ctx.TSLFetchOptions)
	if err != nil {
		return ctx, fmt.Errorf("failed to load TSL from %s: %w", url, err)
	}

	if len(tsls) == 0 {
		return ctx, fmt.Errorf("no TSLs returned from %s", url)
	}

	// Apply filters if any are defined
	originalCount := len(tsls)
	tsls = FilterTSLs(ctx, tsls)
	if len(tsls) < originalCount {
		pl.Logger.Info("Applied TSL filters",
			logging.F("original_count", originalCount),
			logging.F("filtered_count", len(tsls)))
	}

	// Ensure we still have TSLs after filtering
	if len(tsls) == 0 {
		return ctx, fmt.Errorf("no TSLs passed the filter criteria")
	}

	// Build a TSL tree from the loaded TSLs and add it to the stack of trees
	ctx.EnsureTSLTrees()

	// The first TSL is the root, use it to build a new tree
	rootTSL := tsls[0]
	tree := NewTSLTree(rootTSL)
	ctx.AddTSLTree(tree)

	// For backward compatibility, ensure the legacy TSLs stack is populated correctly
	// We need to add TSLs in reverse order: referenced TSLs first, then the root
	if ctx.TSLs == nil {
		ctx.TSLs = utils.NewStack[*etsi119612.TSL]()
	} else {
		// Clear the legacy stack as we're about to rebuild it
		for ctx.TSLs.Size() > 0 {
			ctx.TSLs.Pop()
		}
	}

	// Add referenced TSLs in reverse order (add them last but they'll be popped first)
	for i := len(tsls) - 1; i > 0; i-- {
		ctx.TSLs.Push(tsls[i])
	}

	// Add the root TSL last so it's at the bottom of the stack
	if len(tsls) > 0 {
		ctx.TSLs.Push(tsls[0])
	}

	// Count service providers and services
	var totalProviders int
	var totalServices int
	var schemeTerritory string

	// Log details about each TSL loaded
	for i, tsl := range tsls {
		// Extract scheme territory if available
		if i == 0 && tsl.StatusList.TslSchemeInformation != nil {
			schemeTerritory = tsl.StatusList.TslSchemeInformation.TslSchemeTerritory
		}

		// Count providers and services
		providerCount := 0
		serviceCount := 0
		if tsl.StatusList.TslTrustServiceProviderList != nil {
			providers := tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider
			providerCount = len(providers)
			totalProviders += providerCount

			// Count services for each provider
			for _, provider := range providers {
				if provider != nil && provider.TslTSPServices != nil {
					services := provider.TslTSPServices.TslTSPService
					serviceCount += len(services)
					totalServices += len(services)
				}
			}
		}

		// Log each TSL as it's loaded
		pl.Logger.Info("Loaded TSL",
			logging.F("url", tsl.Source),
			logging.F("providers", providerCount),
			logging.F("services", serviceCount),
			logging.F("referenced", i > 0))
	}

	pl.Logger.Info("Loaded TSLs",
		logging.F("root_url", url),
		logging.F("territory", schemeTerritory),
		logging.F("tree_depth", tree.Depth()),
		logging.F("total_count", len(tsls)),
		logging.F("total_providers", totalProviders),
		logging.F("total_services", totalServices))

	return ctx, nil
}
