package etsi119602

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
)

func TestFromTSL_Empty(t *testing.T) {
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{},
	}
	lote := FromTSL(tsl)
	assert.Equal(t, LoTEVersion, lote.Version)
	assert.Empty(t, lote.TrustedEntities)
}

func TestFromTSL_WithSchemeInfo(t *testing.T) {
	lang := etsi119612.Lang("en")
	name := etsi119612.NonEmptyNormalizedString("Test Operator")

	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeTerritory: "SE",
				TslTSLType:         "http://uri.etsi.org/TrstSvc/TrustedList/TSLType/EUgeneric",
				TSLSequenceNumber:  5,
				TslSchemeOperatorName: &etsi119612.InternationalNamesType{
					Name: []*etsi119612.MultiLangNormStringType{
						{XmlLangAttr: &lang, NonEmptyNormalizedString: &name},
					},
				},
				ListIssueDateTime: "2026-01-15T10:00:00Z",
			},
		},
	}

	lote := FromTSL(tsl)
	assert.Equal(t, "SE", lote.SchemeInformation.Territory)
	assert.Equal(t, 5, lote.SchemeInformation.SequenceNumber)
	assert.Equal(t, "Test Operator", lote.SchemeInformation.SchemeOperator.Get("en", ""))
}

func TestFromTSL_WithServices(t *testing.T) {
	lang := etsi119612.Lang("en")
	svcName := etsi119612.NonEmptyNormalizedString("Test CA")

	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslTSLType:        "http://example.com/tsl",
				ListIssueDateTime: "2026-01-01T00:00:00Z",
			},
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPInformation: &etsi119612.TSPInformationType{
							TSPName: &etsi119612.InternationalNamesType{
								Name: []*etsi119612.MultiLangNormStringType{
									{XmlLangAttr: &lang, NonEmptyNormalizedString: &svcName},
								},
							},
						},
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC",
										ServiceName: &etsi119612.InternationalNamesType{
											Name: []*etsi119612.MultiLangNormStringType{
												{XmlLangAttr: &lang, NonEmptyNormalizedString: &svcName},
											},
										},
										TslServiceStatus:   "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted",
										StatusStartingTime: "2025-06-01T00:00:00Z",
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: []*etsi119612.DigitalIdentityType{
												{X509Certificate: "MIIB...base64..."},
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
	assert.Len(t, lote.TrustedEntities, 1)

	entity := lote.TrustedEntities[0]
	assert.Equal(t, "http://uri.etsi.org/TrstSvc/Svctype/CA/QC", entity.EntityID)
	assert.Equal(t, "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted", entity.EntityStatus)
	assert.Len(t, entity.DigitalIdentities, 1)
	assert.Equal(t, "x509", entity.DigitalIdentities[0].Type)
	assert.NotEmpty(t, entity.StatusStartingTime)
}
