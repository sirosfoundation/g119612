package etsi119602

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- convertAddress coverage ---

func TestConvertAddress_PostalOnly(t *testing.T) {
	lang := etsi119612.Lang("en")
	addr := &etsi119612.AddressType{
		TslPostalAddresses: &etsi119612.PostalAddressListType{
			TslPostalAddress: []*etsi119612.PostalAddressType{
				{
					StreetAddress:   "123 Main St",
					Locality:        "Stockholm",
					CountryName:     "SE",
					PostalCode:      "11122",
					StateOrProvince: "Stockholm County",
					XmlLangAttr:     &lang,
				},
			},
		},
	}

	result := convertAddress(addr)
	require.Contains(t, result, "postal")
	addrs := result["postal"].([]map[string]string)
	require.Len(t, addrs, 1)
	assert.Equal(t, "123 Main St", addrs[0]["streetAddress"])
	assert.Equal(t, "Stockholm", addrs[0]["locality"])
	assert.Equal(t, "SE", addrs[0]["countryName"])
	assert.Equal(t, "11122", addrs[0]["postalCode"])
	assert.Equal(t, "Stockholm County", addrs[0]["stateOrProvince"])
	assert.Equal(t, "en", addrs[0]["language"])
}

func TestConvertAddress_ElectronicOnly(t *testing.T) {
	lang := etsi119612.Lang("en")
	addr := &etsi119612.AddressType{
		TslElectronicAddress: &etsi119612.ElectronicAddressType{
			URI: []*etsi119612.NonEmptyMultiLangURIType{
				{XmlLangAttr: &lang, Value: "mailto:info@example.com"},
				{Value: "https://example.com"},
			},
		},
	}

	result := convertAddress(addr)
	require.Contains(t, result, "electronic")
	uris := result["electronic"].([]LangURI)
	require.Len(t, uris, 2)
	assert.Equal(t, "en", uris[0].Language)
	assert.Equal(t, "mailto:info@example.com", uris[0].URI)
	assert.Equal(t, "", uris[1].Language)
}

func TestConvertAddress_PostalNoOptionalFields(t *testing.T) {
	addr := &etsi119612.AddressType{
		TslPostalAddresses: &etsi119612.PostalAddressListType{
			TslPostalAddress: []*etsi119612.PostalAddressType{
				{
					StreetAddress: "1 Street",
					Locality:      "City",
					CountryName:   "DE",
				},
			},
		},
	}

	result := convertAddress(addr)
	addrs := result["postal"].([]map[string]string)
	_, hasPostal := addrs[0]["postalCode"]
	_, hasState := addrs[0]["stateOrProvince"]
	assert.False(t, hasPostal)
	assert.False(t, hasState)
}

func TestConvertAddress_Empty(t *testing.T) {
	addr := &etsi119612.AddressType{}
	result := convertAddress(addr)
	assert.Empty(t, result)
}

// --- FromTSL coverage: address, trade name, service history, pointer+ ---

func TestFromTSL_WithTSPAddress(t *testing.T) {
	lang := etsi119612.Lang("en")
	svcName := etsi119612.NonEmptyNormalizedString("Address Service")

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
							TSPAddress: &etsi119612.AddressType{
								TslPostalAddresses: &etsi119612.PostalAddressListType{
									TslPostalAddress: []*etsi119612.PostalAddressType{
										{StreetAddress: "1 Test St", Locality: "Test City", CountryName: "SE"},
									},
								},
							},
						},
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://svctype/addr",
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
	require.NotNil(t, lote.TrustedEntities[0].Extensions)
	assert.Contains(t, lote.TrustedEntities[0].Extensions, "tsp_address")
}

func TestFromTSL_WithTradeName(t *testing.T) {
	lang := etsi119612.Lang("en")
	svcName := etsi119612.NonEmptyNormalizedString("Trade Name Svc")
	tradeName := etsi119612.NonEmptyNormalizedString("ACME Corp")

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
							TSPTradeName: &etsi119612.InternationalNamesType{
								Name: []*etsi119612.MultiLangNormStringType{
									{XmlLangAttr: &lang, NonEmptyNormalizedString: &tradeName},
								},
							},
						},
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://svctype/trade",
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
	require.NotNil(t, lote.TrustedEntities[0].Extensions)
	assert.Contains(t, lote.TrustedEntities[0].Extensions, "tsp_trade_name")
}

func TestFromTSL_WithServiceHistory(t *testing.T) {
	lang := etsi119612.Lang("en")
	svcName := etsi119612.NonEmptyNormalizedString("History Service")
	histName := etsi119612.NonEmptyNormalizedString("Old Name")

	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://svctype/hist",
										ServiceName: &etsi119612.InternationalNamesType{
											Name: []*etsi119612.MultiLangNormStringType{
												{XmlLangAttr: &lang, NonEmptyNormalizedString: &svcName},
											},
										},
										TslServiceStatus: StatusGranted,
									},
									TslServiceHistory: &etsi119612.ServiceHistoryType{
										TslServiceHistoryInstance: []*etsi119612.ServiceHistoryInstanceType{
											{
												TslServiceTypeIdentifier: "http://svctype/hist",
												ServiceName: &etsi119612.InternationalNamesType{
													Name: []*etsi119612.MultiLangNormStringType{
														{XmlLangAttr: &lang, NonEmptyNormalizedString: &histName},
													},
												},
												TslServiceStatus:   "http://old/status",
												StatusStartingTime: "2020-01-01T00:00:00Z",
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
	require.NotNil(t, lote.TrustedEntities[0].Extensions)
	assert.Contains(t, lote.TrustedEntities[0].Extensions, "service_history")
	history := lote.TrustedEntities[0].Extensions["service_history"].([]ServiceStatusHistory)
	require.Len(t, history, 1)
	assert.Equal(t, "http://old/status", history[0].ServiceStatus)
	assert.NotNil(t, history[0].StatusStartingTime)
}

func TestFromTSL_WithPointerDigitalIdentities(t *testing.T) {
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslPointersToOtherTSL: &etsi119612.OtherTSLPointersType{
					TslOtherTSLPointer: []*etsi119612.OtherTSLPointerType{
						{
							TSLLocation: "https://other.example.com/tsl.xml",
							TslServiceDigitalIdentities: &etsi119612.ServiceDigitalIdentityListType{
								TslServiceDigitalIdentity: []*etsi119612.DigitalIdentityListType{
									{
										DigitalId: []*etsi119612.DigitalIdentityType{
											{X509Certificate: "AAAA"},
										},
									},
									nil, // nil entry should be skipped
								},
							},
						},
					},
				},
			},
		},
	}
	lote := FromTSL(tsl)
	require.Len(t, lote.PointersToOtherLoTEs, 1)
	ptr := lote.PointersToOtherLoTEs[0]
	require.Len(t, ptr.DigitalIdentities, 1)
	assert.Equal(t, "x509", ptr.DigitalIdentities[0].Type)
}

func TestFromTSL_WithPointerAdditionalInformation(t *testing.T) {
	lang := etsi119612.Lang("en")
	text := etsi119612.NonEmptyString("Additional info text")

	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{
				TslPointersToOtherTSL: &etsi119612.OtherTSLPointersType{
					TslOtherTSLPointer: []*etsi119612.OtherTSLPointerType{
						{
							TSLLocation: "https://example.com/tsl.xml",
							TslAdditionalInformation: &etsi119612.AdditionalInformationType{
								TextualInformation: []*etsi119612.MultiLangStringType{
									{XmlLangAttr: &lang, NonEmptyString: &text},
								},
							},
						},
					},
				},
			},
		},
	}
	lote := FromTSL(tsl)
	require.Len(t, lote.PointersToOtherLoTEs, 1)
	ptr := lote.PointersToOtherLoTEs[0]
	require.NotNil(t, ptr.AdditionalInformation)
	assert.Contains(t, ptr.AdditionalInformation, "textualInformation")
}

func TestFromTSL_WithServiceSupplyPoints(t *testing.T) {
	lang := etsi119612.Lang("en")
	svcName := etsi119612.NonEmptyNormalizedString("Supply Point Svc")

	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://svctype/supply",
										ServiceName: &etsi119612.InternationalNamesType{
											Name: []*etsi119612.MultiLangNormStringType{
												{XmlLangAttr: &lang, NonEmptyNormalizedString: &svcName},
											},
										},
										TslServiceStatus: StatusGranted,
										TslServiceSupplyPoints: &etsi119612.ServiceSupplyPointsType{
											ServiceSupplyPoint: &etsi119612.AttributedNonEmptyURIType{
												Value: "https://supply.example.com",
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
	require.Len(t, lote.TrustedEntities[0].Services, 1)
	assert.Equal(t, []string{"https://supply.example.com"}, lote.TrustedEntities[0].Services[0].ServiceSupplyPoints)
}

// --- FetchAndVerifyLoTE coverage ---

type mockJWSVerifier struct {
	payload []byte
	err     error
}

func (m *mockJWSVerifier) Verify(compact string) ([]byte, error) {
	return m.payload, m.err
}

func TestFetchAndVerifyLoTE_Success(t *testing.T) {
	lote := &ListOfTrustedEntities{
		Version:           LoTEVersion,
		SchemeInformation: SchemeInformation{Territory: "SE"},
	}
	payload, err := json.Marshal(lote)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/jose")
		w.Write([]byte("eyJhbGciOiJub25lIn0.fake.signature"))
	}))
	defer srv.Close()

	verifier := &mockJWSVerifier{payload: payload, err: nil}
	result, err := FetchAndVerifyLoTE(srv.URL+"/lote.jws", nil, verifier)
	require.NoError(t, err)
	assert.Equal(t, "SE", result.SchemeInformation.Territory)
}

func TestFetchAndVerifyLoTE_VerifyFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/jose")
		w.Write([]byte("fake.jws.data"))
	}))
	defer srv.Close()

	verifier := &mockJWSVerifier{err: fmt.Errorf("bad signature")}
	_, err := FetchAndVerifyLoTE(srv.URL+"/lote.jws", nil, verifier)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWS verification failed")
}

func TestFetchAndVerifyLoTE_FetchFails(t *testing.T) {
	verifier := &mockJWSVerifier{payload: []byte("{}"), err: nil}
	_, err := FetchAndVerifyLoTE("http://127.0.0.1:1/unreachable", nil, verifier)
	assert.Error(t, err)
}

func TestFetchAndVerifyLoTE_WithOptions(t *testing.T) {
	lote := &ListOfTrustedEntities{Version: LoTEVersion}
	payload, err := json.Marshal(lote)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "TestFetch/1.0", r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "application/jose")
		w.Write([]byte("jws-content"))
	}))
	defer srv.Close()

	opts := &FetchOptions{
		UserAgent: "TestFetch/1.0",
		Timeout:   5 * time.Second,
	}
	verifier := &mockJWSVerifier{payload: payload}
	result, err := FetchAndVerifyLoTE(srv.URL+"/lote.jws", opts, verifier)
	require.NoError(t, err)
	assert.Equal(t, LoTEVersion, result.Version)
}
