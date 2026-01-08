package pipeline

import (
	"strings"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
)

// FilterTSLs applies filters to the TSLs based on the filters defined in the context.
// It returns a new slice containing only the TSLs that match the filters.
func FilterTSLs(ctx *Context, tsls []*etsi119612.TSL) []*etsi119612.TSL {
	// Get filters from context
	filtersAny, ok := ctx.Data["tsl_filters"]
	if !ok {
		// No filters defined, return the original slice
		return tsls
	}

	filters, ok := filtersAny.(map[string][]string)
	if !ok || len(filters) == 0 {
		// No valid filters, return the original slice
		return tsls
	}

	// Apply filters
	result := make([]*etsi119612.TSL, 0, len(tsls))
	for _, tsl := range tsls {
		if matchesFilters(tsl, filters) {
			result = append(result, tsl)
		}
	}

	return result
}

// matchesFilters checks if a TSL matches all the specified filters
func matchesFilters(tsl *etsi119612.TSL, filters map[string][]string) bool {
	// Check territory filter
	if territories, ok := filters["territory"]; ok && len(territories) > 0 {
		if !matchesTerritory(tsl, territories) {
			return false
		}
	}

	// Check service type filter
	if serviceTypes, ok := filters["service-type"]; ok && len(serviceTypes) > 0 {
		if !matchesServiceType(tsl, serviceTypes) {
			return false
		}
	}

	// All filters passed
	return true
}

// matchesTerritory checks if a TSL's territory matches any of the specified territories
func matchesTerritory(tsl *etsi119612.TSL, territories []string) bool {
	if tsl.StatusList.TslSchemeInformation == nil {
		return false
	}

	territory := tsl.StatusList.TslSchemeInformation.TslSchemeTerritory

	for _, filter := range territories {
		if strings.EqualFold(territory, filter) {
			return true
		}
	}

	return false
}

// matchesServiceType checks if a TSL has any service that matches the specified types
func matchesServiceType(tsl *etsi119612.TSL, serviceTypes []string) bool {
	if tsl.StatusList.TslTrustServiceProviderList == nil {
		return false
	}

	for _, provider := range tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider {
		if provider == nil || provider.TslTSPServices == nil {
			continue
		}

		for _, service := range provider.TslTSPServices.TslTSPService {
			if service == nil || service.TslServiceInformation == nil {
				continue
			}

			serviceType := service.TslServiceInformation.TslServiceTypeIdentifier
			for _, filter := range serviceTypes {
				// Use contains for partial matching of service types
				if strings.Contains(serviceType, filter) {
					return true
				}
			}
		}
	}

	return false
}
