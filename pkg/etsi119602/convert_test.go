package etsi119602

import (
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestFromTSL_WithNextUpdate(t *testing.T) {
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				ListIssueDateTime: "2026-01-01T00:00:00Z",
				TslNextUpdate: &etsi119612.NextUpdateType{
					DateTime: "2026-07-01T00:00:00Z",
				},
			},
		},
	}
	lote := FromTSL(tsl)
	require.NotNil(t, lote.SchemeInformation.NextUpdate)
	assert.Equal(t, 2026, lote.SchemeInformation.NextUpdate.Year())
	assert.Equal(t, 7, int(lote.SchemeInformation.NextUpdate.Month()))
}

func TestFromTSL_WithDistributionPoints(t *testing.T) {
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslDistributionPoints: &etsi119612.NonEmptyURIListType{
					URI: []string{"https://example.com/tsl.xml", "https://backup.example.com/tsl.xml"},
				},
			},
		},
	}
	lote := FromTSL(tsl)
	assert.Equal(t, []string{"https://example.com/tsl.xml", "https://backup.example.com/tsl.xml"}, lote.SchemeInformation.DistributionPoints)
}

func TestFromTSL_WithSchemeInformationURI(t *testing.T) {
	lang := etsi119612.Lang("en")
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeInformationURI: &etsi119612.NonEmptyMultiLangURIListType{
					URI: []*etsi119612.NonEmptyMultiLangURIType{
						{XmlLangAttr: &lang, Value: "https://info.example.com"},
						{Value: "https://nolang.example.com"},
					},
				},
			},
		},
	}
	lote := FromTSL(tsl)
	require.Len(t, lote.SchemeInformation.SchemeInformationURI, 2)
	assert.Equal(t, "en", lote.SchemeInformation.SchemeInformationURI[0].Language)
	assert.Equal(t, "https://info.example.com", lote.SchemeInformation.SchemeInformationURI[0].URI)
	assert.Equal(t, "", lote.SchemeInformation.SchemeInformationURI[1].Language)
}

func TestFromTSL_WithPolicyNotice(t *testing.T) {
	lang := etsi119612.Lang("en")
	notice := etsi119612.NonEmptyString("Legal notice text")
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslPolicyOrLegalNotice: &etsi119612.PolicyOrLegalnoticeType{
					TSLLegalNotice: []*etsi119612.MultiLangStringType{
						{XmlLangAttr: &lang, NonEmptyString: &notice},
						{}, // empty notice — tests nil guard
					},
				},
			},
		},
	}
	lote := FromTSL(tsl)
	require.Len(t, lote.SchemeInformation.PolicyOrLegalNotice, 2)
	assert.Equal(t, "Legal notice text", lote.SchemeInformation.PolicyOrLegalNotice[0].Value)
	assert.Equal(t, "", lote.SchemeInformation.PolicyOrLegalNotice[1].Value)
}

func TestFromTSL_WithPointers(t *testing.T) {
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslPointersToOtherTSL: &etsi119612.OtherTSLPointersType{
					TslOtherTSLPointer: []*etsi119612.OtherTSLPointerType{
						{TSLLocation: "https://other.example.com/tsl.xml"},
						{TSLLocation: "https://another.example.com/tsl.xml"},
					},
				},
			},
		},
	}
	lote := FromTSL(tsl)
	require.Len(t, lote.PointersToOtherLoTEs, 2)
	assert.Equal(t, "https://other.example.com/tsl.xml", lote.PointersToOtherLoTEs[0].Location)
}

func TestFromTSL_WithX509SubjectName(t *testing.T) {
	lang := etsi119612.Lang("en")
	svcName := etsi119612.NonEmptyNormalizedString("Subject Name Service")
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://svctype/test",
										ServiceName: &etsi119612.InternationalNamesType{
											Name: []*etsi119612.MultiLangNormStringType{
												{XmlLangAttr: &lang, NonEmptyNormalizedString: &svcName},
											},
										},
										TslServiceStatus: StatusGranted,
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: []*etsi119612.DigitalIdentityType{
												{X509SubjectName: "CN=Test,O=Org,C=SE"},
												{X509Certificate: "AAAA"},
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
	require.Len(t, lote.TrustedEntities, 1)
	require.Len(t, lote.TrustedEntities[0].DigitalIdentities, 2)
	assert.Equal(t, "x509_subject_name", lote.TrustedEntities[0].DigitalIdentities[0].Type)
	assert.Equal(t, "CN=Test,O=Org,C=SE", lote.TrustedEntities[0].DigitalIdentities[0].X509SubjectName)
	assert.Equal(t, "x509", lote.TrustedEntities[0].DigitalIdentities[1].Type)
}

func TestFromTSL_WithInformationURIs(t *testing.T) {
	lang := etsi119612.Lang("en")
	svcName := etsi119612.NonEmptyNormalizedString("URI Service")
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPInformation: &etsi119612.TSPInformationType{
							TSPName: &etsi119612.InternationalNamesType{
								Name: []*etsi119612.MultiLangNormStringType{
									{XmlLangAttr: &lang, NonEmptyNormalizedString: &svcName},
								},
							},
							TSPInformationURI: &etsi119612.NonEmptyMultiLangURIListType{
								URI: []*etsi119612.NonEmptyMultiLangURIType{
									{XmlLangAttr: &lang, Value: "https://provider.example.com"},
								},
							},
						},
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://svctype/uri",
										ServiceName: &etsi119612.InternationalNamesType{
											Name: []*etsi119612.MultiLangNormStringType{
												{XmlLangAttr: &lang, NonEmptyNormalizedString: &svcName},
											},
										},
										TslServiceStatus: StatusGranted,
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
	require.Len(t, lote.TrustedEntities, 1)
	require.Len(t, lote.TrustedEntities[0].InformationURIs, 1)
	assert.Equal(t, "https://provider.example.com", lote.TrustedEntities[0].InformationURIs[0].URI)
}

func TestFromTSL_TSPWithNoServices(t *testing.T) {
	// TSP without services should be skipped
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPInformation: &etsi119612.TSPInformationType{},
						TslTSPServices:    nil, // no services
					},
				},
			},
		},
	}
	lote := FromTSL(tsl)
	assert.Empty(t, lote.TrustedEntities)
}

func TestFromTSL_ServiceWithNilInfo(t *testing.T) {
	// Service with nil ServiceInformation should be skipped
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{TslServiceInformation: nil},
							},
						},
					},
				},
			},
		},
	}
	lote := FromTSL(tsl)
	assert.Empty(t, lote.TrustedEntities)
}

func TestFromTSL_InvalidDates(t *testing.T) {
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				ListIssueDateTime: "not-a-date",
				TslNextUpdate: &etsi119612.NextUpdateType{
					DateTime: "also-not-a-date",
				},
			},
		},
	}
	lote := FromTSL(tsl)
	// Dates should be zero values when parsing fails
	assert.True(t, lote.SchemeInformation.IssueDate.IsZero())
	assert.Nil(t, lote.SchemeInformation.NextUpdate)
}

func TestFromTSL_SchemeName(t *testing.T) {
	lang := etsi119612.Lang("en")
	name := etsi119612.NonEmptyNormalizedString("Scheme Name")
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslSchemeName: &etsi119612.InternationalNamesType{
					Name: []*etsi119612.MultiLangNormStringType{
						{XmlLangAttr: &lang, NonEmptyNormalizedString: &name},
					},
				},
			},
		},
	}
	lote := FromTSL(tsl)
	assert.Equal(t, "Scheme Name", lote.SchemeInformation.SchemeName.Get("en", ""))
}
