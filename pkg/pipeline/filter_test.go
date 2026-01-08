package pipeline

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
)

func TestFilterTSLs(t *testing.T) {
	// Create test TSLs with different territories and service types
	tsl1 := createTestTSL("TSL1", "SE", []string{"http://service-type-1", "http://service-type-2"})
	tsl2 := createTestTSL("TSL2", "FI", []string{"http://service-type-3"})
	tsl3 := createTestTSL("TSL3", "NO", []string{"http://service-type-1", "http://service-type-4"})

	tsls := []*etsi119612.TSL{tsl1, tsl2, tsl3}

	tests := []struct {
		name          string
		filters       map[string][]string
		expectedCount int
		expectedTSLs  []string
	}{
		{
			name:          "No filters",
			filters:       map[string][]string{},
			expectedCount: 3,
			expectedTSLs:  []string{"TSL1", "TSL2", "TSL3"},
		},
		{
			name:          "Filter by territory SE",
			filters:       map[string][]string{"territory": {"SE"}},
			expectedCount: 1,
			expectedTSLs:  []string{"TSL1"},
		},
		{
			name:          "Filter by territory SE or FI",
			filters:       map[string][]string{"territory": {"SE", "FI"}},
			expectedCount: 2,
			expectedTSLs:  []string{"TSL1", "TSL2"},
		},
		{
			name:          "Filter by service type 1",
			filters:       map[string][]string{"service-type": {"service-type-1"}},
			expectedCount: 2,
			expectedTSLs:  []string{"TSL1", "TSL3"},
		},
		{
			name: "Filter by territory SE and service type 1",
			filters: map[string][]string{
				"territory":    {"SE"},
				"service-type": {"service-type-1"},
			},
			expectedCount: 1,
			expectedTSLs:  []string{"TSL1"},
		},
		{
			name:          "Filter with no matches",
			filters:       map[string][]string{"territory": {"DE"}},
			expectedCount: 0,
			expectedTSLs:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := NewContext()
			ctx.Data["tsl_filters"] = tc.filters

			result := FilterTSLs(ctx, tsls)
			assert.Equal(t, tc.expectedCount, len(result), "Expected %d filtered TSLs, got %d", tc.expectedCount, len(result))

			// Check that the expected TSLs are in the result
			resultNames := make([]string, 0, len(result))
			for _, tsl := range result {
				resultNames = append(resultNames, tsl.Source)
			}

			for _, expected := range tc.expectedTSLs {
				assert.Contains(t, resultNames, expected, "Expected result to contain TSL %s", expected)
			}
		})
	}
}

// Helper function to create a test TSL with specified territory and service types
func createTestTSL(source, territory string, serviceTypes []string) *etsi119612.TSL {
	tsl := &etsi119612.TSL{
		Source: source,
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: territory,
			},
		},
	}

	// Add providers and services
	if len(serviceTypes) > 0 {
		// Create a provider
		provider := &etsi119612.TSPType{
			TslTSPServices: &etsi119612.TSPServicesListType{
				TslTSPService: make([]*etsi119612.TSPServiceType, 0, len(serviceTypes)),
			},
		}

		// Add services with the specified types
		for _, serviceType := range serviceTypes {
			service := &etsi119612.TSPServiceType{
				TslServiceInformation: &etsi119612.TSPServiceInformationType{
					TslServiceTypeIdentifier: serviceType,
				},
			}
			provider.TslTSPServices.TslTSPService = append(provider.TslTSPServices.TslTSPService, service)
		}

		// Add the provider to the TSL
		tsl.StatusList.TslTrustServiceProviderList = &etsi119612.TrustServiceProviderListType{
			TslTrustServiceProvider: []*etsi119612.TSPType{provider},
		}
	}

	return tsl
}

func TestMatchesTerritory_EdgeCases(t *testing.T) {
	t.Run("Case insensitive match", func(t *testing.T) {
		tsl := &etsi119612.TSL{
			StatusList: etsi119612.TrustStatusListType{
				TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
					TslSchemeTerritory: "SE",
				},
			},
		}

		// Test lowercase filter
		assert.True(t, matchesTerritory(tsl, []string{"se"}))
		// Test uppercase filter
		assert.True(t, matchesTerritory(tsl, []string{"SE"}))
		// Test mixed case filter
		assert.True(t, matchesTerritory(tsl, []string{"Se"}))
	})

	t.Run("Returns false for nil scheme information", func(t *testing.T) {
		tsl := &etsi119612.TSL{
			StatusList: etsi119612.TrustStatusListType{
				TslSchemeInformation: nil,
			},
		}

		assert.False(t, matchesTerritory(tsl, []string{"SE"}))
	})
}

func TestMatchesServiceType_EdgeCases(t *testing.T) {
	t.Run("Returns false for nil provider list", func(t *testing.T) {
		tsl := &etsi119612.TSL{
			StatusList: etsi119612.TrustStatusListType{
				TslTrustServiceProviderList: nil,
			},
		}

		assert.False(t, matchesServiceType(tsl, []string{"CA/QC"}))
	})

	t.Run("Skips nil providers", func(t *testing.T) {
		tsl := &etsi119612.TSL{
			StatusList: etsi119612.TrustStatusListType{
				TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
					TslTrustServiceProvider: []*etsi119612.TSPType{
						nil,
						{
							TslTSPServices: &etsi119612.TSPServicesListType{
								TslTSPService: []*etsi119612.TSPServiceType{
									{
										TslServiceInformation: &etsi119612.TSPServiceInformationType{
											TslServiceTypeIdentifier: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		assert.True(t, matchesServiceType(tsl, []string{"CA/QC"}))
	})

	t.Run("Skips providers with nil services", func(t *testing.T) {
		tsl := &etsi119612.TSL{
			StatusList: etsi119612.TrustStatusListType{
				TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
					TslTrustServiceProvider: []*etsi119612.TSPType{
						{
							TslTSPServices: nil,
						},
					},
				},
			},
		}

		assert.False(t, matchesServiceType(tsl, []string{"CA/QC"}))
	})

	t.Run("Skips nil services in list", func(t *testing.T) {
		tsl := &etsi119612.TSL{
			StatusList: etsi119612.TrustStatusListType{
				TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
					TslTrustServiceProvider: []*etsi119612.TSPType{
						{
							TslTSPServices: &etsi119612.TSPServicesListType{
								TslTSPService: []*etsi119612.TSPServiceType{
									nil,
									{
										TslServiceInformation: &etsi119612.TSPServiceInformationType{
											TslServiceTypeIdentifier: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		assert.True(t, matchesServiceType(tsl, []string{"CA/QC"}))
	})

	t.Run("Skips services with nil information", func(t *testing.T) {
		tsl := &etsi119612.TSL{
			StatusList: etsi119612.TrustStatusListType{
				TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
					TslTrustServiceProvider: []*etsi119612.TSPType{
						{
							TslTSPServices: &etsi119612.TSPServicesListType{
								TslTSPService: []*etsi119612.TSPServiceType{
									{
										TslServiceInformation: nil,
									},
								},
							},
						},
					},
				},
			},
		}

		assert.False(t, matchesServiceType(tsl, []string{"CA/QC"}))
	})

	t.Run("Uses contains for partial matching", func(t *testing.T) {
		tsl := &etsi119612.TSL{
			StatusList: etsi119612.TrustStatusListType{
				TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
					TslTrustServiceProvider: []*etsi119612.TSPType{
						{
							TslTSPServices: &etsi119612.TSPServicesListType{
								TslTSPService: []*etsi119612.TSPServiceType{
									{
										TslServiceInformation: &etsi119612.TSPServiceInformationType{
											TslServiceTypeIdentifier: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Test partial match
		assert.True(t, matchesServiceType(tsl, []string{"CA/QC"}))
		assert.True(t, matchesServiceType(tsl, []string{"Svctype"}))
		assert.True(t, matchesServiceType(tsl, []string{"etsi.org"}))
	})
}
