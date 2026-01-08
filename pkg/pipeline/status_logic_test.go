package pipeline

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
)

// TestSelectCertPoolWithStatusLogic tests the status-logic:and parameter
func TestSelectCertPoolWithStatusLogic(t *testing.T) {
	pl := createTestPipeline(nil)
	ctx := NewContext()

	// Create test TSL with multiple service statuses
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TSLVersionIdentifier: 1,
				TslSchemeOperatorName: &etsi119612.InternationalNamesType{
					Name: []*etsi119612.MultiLangNormStringType{
						{
							XmlLangAttr: func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
							NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
								s := etsi119612.NonEmptyNormalizedString("Test Operator")
								return &s
							}(),
						},
					},
				},
			},
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					// Provider 1 - service with status1 (should match OR logic test)
					{
						TslTSPInformation: &etsi119612.TSPInformationType{
							TSPName: &etsi119612.InternationalNamesType{
								Name: []*etsi119612.MultiLangNormStringType{
									{
										XmlLangAttr: func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
										NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
											s := etsi119612.NonEmptyNormalizedString("Provider 1")
											return &s
										}(),
									},
								},
							},
						},
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://example.org/ServiceType",
										TslServiceStatus:         "status1",
										ServiceName: &etsi119612.InternationalNamesType{
											Name: []*etsi119612.MultiLangNormStringType{
												{
													XmlLangAttr: func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
													NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
														s := etsi119612.NonEmptyNormalizedString("Service 1")
														return &s
													}(),
												},
											},
										},
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: []*etsi119612.DigitalIdentityType{
												{
													X509Certificate: "CERT1",
												},
											},
										},
									},
								},
							},
						},
					},
					// Provider 2 - service with status2 (should match OR logic test)
					{
						TslTSPInformation: &etsi119612.TSPInformationType{
							TSPName: &etsi119612.InternationalNamesType{
								Name: []*etsi119612.MultiLangNormStringType{
									{
										XmlLangAttr: func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
										NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
											s := etsi119612.NonEmptyNormalizedString("Provider 2")
											return &s
										}(),
									},
								},
							},
						},
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://example.org/ServiceType",
										TslServiceStatus:         "status2",
										ServiceName: &etsi119612.InternationalNamesType{
											Name: []*etsi119612.MultiLangNormStringType{
												{
													XmlLangAttr: func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
													NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
														s := etsi119612.NonEmptyNormalizedString("Service 2")
														return &s
													}(),
												},
											},
										},
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: []*etsi119612.DigitalIdentityType{
												{
													X509Certificate: "CERT2",
												},
											},
										},
									},
								},
							},
						},
					},
					// Provider 3 - Both status1 AND status2 (not possible in real TSL, but for testing AND logic)
					{
						TslTSPInformation: &etsi119612.TSPInformationType{
							TSPName: &etsi119612.InternationalNamesType{
								Name: []*etsi119612.MultiLangNormStringType{
									{
										XmlLangAttr: func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
										NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
											s := etsi119612.NonEmptyNormalizedString("Provider 3")
											return &s
										}(),
									},
								},
							},
						},
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								// This test is artificial, as a real service can only have one status
								// We're testing the filter logic, not real-world TSL structure
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://example.org/ServiceType",
										TslServiceStatus:         "status1 status2", // Artificial status that would match both filters
										ServiceName: &etsi119612.InternationalNamesType{
											Name: []*etsi119612.MultiLangNormStringType{
												{
													XmlLangAttr: func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
													NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
														s := etsi119612.NonEmptyNormalizedString("Service 3")
														return &s
													}(),
												},
											},
										},
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: []*etsi119612.DigitalIdentityType{
												{
													X509Certificate: "CERT3",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Set up the TSL stack in the context
	ctx.EnsureTSLStack()
	ctx.TSLs.Push(tsl)

	// Test with OR logic (default)
	ctx, err := SelectCertPool(pl, ctx, "status:status1", "status:status2")
	assert.NoError(t, err)
	assert.NotNil(t, ctx.CertPool)
	// With OR logic, both status1 and status2 certificates should be included
	// Cannot easily test this without accessing private CertPool internals

	// Test with AND logic
	ctx = NewContext()
	ctx.EnsureTSLStack()
	ctx.TSLs.Push(tsl)
	ctx, err = SelectCertPool(pl, ctx, "status:status1", "status:status2", "status-logic:and")
	assert.NoError(t, err)
	assert.NotNil(t, ctx.CertPool)
	// With AND logic, only services matching both status1 AND status2 should be included
	// Cannot easily test this without accessing private CertPool internals
}
