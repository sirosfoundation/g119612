package etsi119602

import (
	"testing"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromTSL_Basic(t *testing.T) {
	lang := etsi119612.Lang("en")
	name := etsi119612.NonEmptyNormalizedString("Swedish TSL")
	operatorName := etsi119612.NonEmptyNormalizedString("Swedish Authority")
	svcName := etsi119612.NonEmptyNormalizedString("Test Service")
	svcType := "http://uri.etsi.org/TrstSvc/Svctype/CA/QC"
	svcStatus := StatusGranted

	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "SE",
				TslSchemeOperatorName: &etsi119612.InternationalNamesType{
					Name: []*etsi119612.MultiLangNormStringType{
						{XmlLangAttr: &lang, NonEmptyNormalizedString: &operatorName},
					},
				},
				TslSchemeName: &etsi119612.InternationalNamesType{
					Name: []*etsi119612.MultiLangNormStringType{
						{XmlLangAttr: &lang, NonEmptyNormalizedString: &name},
					},
				},
				TslTSLType:        "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUlistofthelists",
				TSLSequenceNumber: 5,
				ListIssueDateTime: "2026-01-01T00:00:00Z",
			},
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPInformation: &etsi119612.TSPInformationType{
							TSPName: &etsi119612.InternationalNamesType{
								Name: []*etsi119612.MultiLangNormStringType{
									{XmlLangAttr: &lang, NonEmptyNormalizedString: &operatorName},
								},
							},
						},
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										ServiceName: &etsi119612.InternationalNamesType{
											Name: []*etsi119612.MultiLangNormStringType{
												{XmlLangAttr: &lang, NonEmptyNormalizedString: &svcName},
											},
										},
										TslServiceTypeIdentifier: svcType,
										TslServiceStatus:         svcStatus,
										StatusStartingTime:       "2025-06-01T00:00:00Z",
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: []*etsi119612.DigitalIdentityType{
												{X509Certificate: "MIIB...test..."},
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

	lote := FromTSL(tsl)

	assert.Equal(t, 1, lote.ListAndSchemeInformation.LoTEVersionIdentifier)
	assert.Equal(t, "SE", lote.ListAndSchemeInformation.SchemeTerritory)
	assert.Equal(t, 5, lote.ListAndSchemeInformation.LoTESequenceNumber)
	assert.Equal(t, "Swedish Authority", lote.ListAndSchemeInformation.SchemeOperatorName.Get("en", ""))
	assert.Equal(t, "2026-01-01T00:00:00Z", lote.ListAndSchemeInformation.ListIssueDateTime)

	require.Len(t, lote.TrustedEntitiesList, 1)
	entity := lote.TrustedEntitiesList[0]
	assert.Equal(t, "Test Service", entity.TrustedEntityInformation.TEName.Get("en", ""))

	require.Len(t, entity.TrustedEntityServices, 1)
	svc := entity.TrustedEntityServices[0]
	assert.Equal(t, svcType, svc.ServiceInformation.ServiceTypeIdentifier)
	assert.Equal(t, svcStatus, svc.ServiceInformation.ServiceStatus)
	require.Len(t, svc.ServiceInformation.ServiceDigitalIdentity.X509Certificates, 1)
	assert.Equal(t, "MIIB...test...", svc.ServiceInformation.ServiceDigitalIdentity.X509Certificates[0].Val)
}

func TestFromTSL_WithPointers(t *testing.T) {
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "EU",
				TslPointersToOtherTSL: &etsi119612.OtherTSLPointersType{
					TslOtherTSLPointer: []*etsi119612.OtherTSLPointerType{
						{
							TSLLocation: "https://example.se/tsl.xml",
							TslServiceDigitalIdentities: &etsi119612.ServiceDigitalIdentityListType{
								TslServiceDigitalIdentity: []*etsi119612.DigitalIdentityListType{
									{
										DigitalId: []*etsi119612.DigitalIdentityType{
											{X509Certificate: "MIIB...signer..."},
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

	lote := FromTSL(tsl)
	ptrs := lote.ListAndSchemeInformation.PointersToOtherLoTE
	require.Len(t, ptrs, 1)
	assert.Equal(t, "https://example.se/tsl.xml", ptrs[0].LoTELocation)
	require.Len(t, ptrs[0].ServiceDigitalIdentities, 1)
	require.Len(t, ptrs[0].ServiceDigitalIdentities[0].X509Certificates, 1)
}

func TestConvertAddress(t *testing.T) {
	lang := etsi119612.Lang("en")

	// nil address
	assert.Nil(t, convertAddress(nil))

	// Address with postal and electronic
	addr := &etsi119612.AddressType{
		TslPostalAddresses: &etsi119612.PostalAddressListType{
			TslPostalAddress: []*etsi119612.PostalAddressType{
				{
					XmlLangAttr:     &lang,
					StreetAddress:   "123 Main St",
					Locality:        "Stockholm",
					StateOrProvince: "Stockholm",
					PostalCode:      "111 22",
					CountryName:     "SE",
				},
			},
		},
		TslElectronicAddress: &etsi119612.ElectronicAddressType{
			URI: []*etsi119612.NonEmptyMultiLangURIType{
				{XmlLangAttr: &lang, Value: "https://example.se"},
			},
		},
	}

	result := convertAddress(addr)
	require.NotNil(t, result)
	require.Len(t, result.TEPostalAddress, 1)
	assert.Equal(t, "123 Main St", result.TEPostalAddress[0].StreetAddress)
	assert.Equal(t, "Stockholm", result.TEPostalAddress[0].Locality)
	assert.Equal(t, "Stockholm", result.TEPostalAddress[0].StateOrProvince)
	assert.Equal(t, "111 22", result.TEPostalAddress[0].PostalCode)
	assert.Equal(t, "SE", result.TEPostalAddress[0].Country)
	assert.Equal(t, "en", result.TEPostalAddress[0].Lang)

	require.Len(t, result.TEElectronicAddress, 1)
	assert.Equal(t, "https://example.se", result.TEElectronicAddress[0].URIValue)
	assert.Equal(t, "en", result.TEElectronicAddress[0].Lang)
}

func TestTimeToString(t *testing.T) {
	assert.Equal(t, "", timeToString(time.Time{}))

	ts := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, "2026-01-01T12:00:00Z", timeToString(ts))
}

func TestURIsFromInternational(t *testing.T) {
	assert.Nil(t, urisFromInternational(nil))

	lang := etsi119612.Lang("en")
	uris := &etsi119612.NonEmptyMultiLangURIListType{
		URI: []*etsi119612.NonEmptyMultiLangURIType{
			{XmlLangAttr: &lang, Value: "https://example.com"},
			{Value: "https://example.se"},
		},
	}
	result := urisFromInternational(uris)
	require.Len(t, result, 2)
	assert.Equal(t, "en", result[0].Lang)
	assert.Equal(t, "https://example.com", result[0].URIValue)
	assert.Equal(t, "", result[1].Lang)
	assert.Equal(t, "https://example.se", result[1].URIValue)
}
