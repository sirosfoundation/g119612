package pipeline

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

// SetFetchOptions is a pipeline step that configures the options for fetching Trust Status Lists.
// This function sets up the user-agent, timeout, content negotiation, and other options used when fetching TSLs.
//
// Parameters:
//   - pl: Pipeline instance managing the step execution
//   - ctx: Pipeline context containing state information
//   - args: String slice with options in the format "key:value", where key can be:
//   - user-agent: Custom User-Agent header for HTTP requests
//   - timeout: Maximum time to wait for HTTP requests (any valid Go duration string)
//   - max-depth: Maximum depth for following TSL references (integer, 0=none, -1=unlimited)
//   - accept: Comma-separated list of Accept header values for content negotiation (e.g., "application/xml,text/xml")
//   - prefer-xml: If set to "true", the fetcher will try .xml extension if .pdf fails
//   - filter-territory: Only include TSLs from the specified territory (e.g., "SE,FI,NO")
//   - filter-service-type: Only include TSLs with services of the specified type(s) (comma-separated)
//
// Returns:
//   - *Context: Updated context with the configured fetch options
//   - error: Non-nil if an option cannot be parsed
//
// Example usage in pipeline configuration:
//   - set-fetch-options:
//   - user-agent:MyCustomUserAgent/1.0
//   - timeout:60s
//   - max-depth:2
//   - accept:application/xml,text/xml
//   - prefer-xml:true
//   - filter-territory:SE
func SetFetchOptions(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	// Ensure the TSLFetchOptions are initialized
	ctx.EnsureTSLFetchOptions()

	// Create custom filters field if it doesn't exist
	if ctx.Data["tsl_filters"] == nil {
		ctx.Data["tsl_filters"] = make(map[string][]string)
	}
	filters, ok := ctx.Data["tsl_filters"].(map[string][]string)
	if !ok {
		// If it's not the right type, recreate it
		filters = make(map[string][]string)
		ctx.Data["tsl_filters"] = filters
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "user-agent:") {
			ctx.TSLFetchOptions.UserAgent = strings.TrimPrefix(arg, "user-agent:")
			pl.Logger.Debug("Set TSL fetch User-Agent", logging.F("user-agent", ctx.TSLFetchOptions.UserAgent))
		} else if strings.HasPrefix(arg, "timeout:") {
			timeoutStr := strings.TrimPrefix(arg, "timeout:")
			if timeout, err := time.ParseDuration(timeoutStr); err == nil {
				ctx.TSLFetchOptions.Timeout = timeout
				pl.Logger.Debug("Set TSL fetch timeout", logging.F("timeout", ctx.TSLFetchOptions.Timeout))
			} else {
				return ctx, fmt.Errorf("invalid timeout value: %s (%w)", timeoutStr, err)
			}
		} else if strings.HasPrefix(arg, "max-depth:") {
			depthStr := strings.TrimPrefix(arg, "max-depth:")
			if depth, err := strconv.Atoi(depthStr); err == nil {
				ctx.TSLFetchOptions.MaxDereferenceDepth = depth
				pl.Logger.Debug("Set TSL fetch maximum dereference depth", logging.F("max-depth", depth))
			} else {
				return ctx, fmt.Errorf("invalid max-depth value: %s (%w)", depthStr, err)
			}
		} else if strings.HasPrefix(arg, "accept:") {
			// Handle Accept header for content negotiation
			accepts := strings.TrimPrefix(arg, "accept:")
			if accepts == "" {
				// Reset to default if empty
				ctx.TSLFetchOptions.AcceptHeaders = etsi119612.DefaultTSLFetchOptions.AcceptHeaders
			} else {
				// Parse comma-separated list of Accept header values
				headers := strings.Split(accepts, ",")
				for i, h := range headers {
					headers[i] = strings.TrimSpace(h)
				}
				ctx.TSLFetchOptions.AcceptHeaders = headers
			}
			pl.Logger.Debug("Set TSL fetch Accept headers", logging.F("accept", ctx.TSLFetchOptions.AcceptHeaders))
		} else if strings.HasPrefix(arg, "prefer-xml:") {
			preferXML := strings.TrimPrefix(arg, "prefer-xml:")
			if preferXML == "true" || preferXML == "1" || preferXML == "yes" {
				// Store in context data instead since we can't modify the TSLFetchOptions structure
				ctx.Data["prefer_xml_over_pdf"] = true
				pl.Logger.Debug("Set TSL fetch prefer XML over PDF", logging.F("prefer-xml", true))
			} else {
				ctx.Data["prefer_xml_over_pdf"] = false
				pl.Logger.Debug("Set TSL fetch prefer XML over PDF", logging.F("prefer-xml", false))
			}
		} else if strings.HasPrefix(arg, "filter-territory:") {
			// Parse territory filter
			territories := strings.TrimPrefix(arg, "filter-territory:")
			if territories != "" {
				filters["territory"] = strings.Split(territories, ",")
				for i, t := range filters["territory"] {
					filters["territory"][i] = strings.TrimSpace(t)
				}
				pl.Logger.Debug("Set TSL filter by territory", logging.F("territories", filters["territory"]))
			}
		} else if strings.HasPrefix(arg, "filter-service-type:") {
			// Parse service type filter
			serviceTypes := strings.TrimPrefix(arg, "filter-service-type:")
			if serviceTypes != "" {
				filters["service-type"] = strings.Split(serviceTypes, ",")
				for i, t := range filters["service-type"] {
					filters["service-type"][i] = strings.TrimSpace(t)
				}
				pl.Logger.Debug("Set TSL filter by service type", logging.F("service-types", filters["service-type"]))
			}
		} else {
			pl.Logger.Warn("Unknown fetch option", logging.F("option", arg))
		}
	}

	// Store filters in the context data
	ctx.Data["tsl_filters"] = filters

	return ctx, nil
}
