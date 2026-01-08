package pipeline

import (
	"crypto/x509"
	"testing"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

func TestSelectCertPoolWithFilters(t *testing.T) {
	// Create a test pipeline with a logger
	pl := &Pipeline{
		Logger: logging.DefaultLogger(), // Use default logger for testing
	}

	// Create a context with test TSLs
	ctx := &Context{}
	ctx.EnsureTSLStack()

	// Use the test certificate from test_utils.go
	cert := TestCert
	if cert == nil {
		t.Fatal("TestCert is nil, make sure test_utils.go has initialized the test certificate properly")
	}

	// Create TSLs with different service types
	testCases := []struct {
		serviceType string
		status      string
	}{
		{"http://uri.etsi.org/TrstSvc/Svctype/CA/QC", "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/"},
		{"http://uri.etsi.org/TrstSvc/Svctype/TSA/QTST", "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/"},
		{"http://uri.etsi.org/TrstSvc/Svctype/CA/PKC", "http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/withdrawn/"},
	}

	for _, tc := range testCases {
		tsl := createTestTSLWithCert(cert, tc.serviceType, tc.status)
		ctx.TSLs.Push(tsl)
	}

	// Test case 1: No filters - should include all certificates
	ctx1, err := SelectCertPool(pl, ctx)
	if err != nil {
		t.Errorf("SelectCertPool failed: %v", err)
	}
	if ctx1.CertPool == nil {
		t.Fatal("CertPool is nil")
	}

	// Test case 2: Filter by service type
	ctx2 := ctx.Copy()
	_, err = SelectCertPool(pl, ctx2, "service-type:http://uri.etsi.org/TrstSvc/Svctype/CA/QC")
	if err != nil {
		t.Errorf("SelectCertPool with service type filter failed: %v", err)
	}

	// Test case 3: Filter by service type and status
	ctx3 := ctx.Copy()
	ctx3, err = SelectCertPool(pl, ctx3,
		"service-type:http://uri.etsi.org/TrstSvc/Svctype/CA/QC",
		"status:http://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/")
	if err != nil {
		t.Errorf("SelectCertPool with service type and status filters failed: %v", err)
	}

	// Test case 4: Filter with no matches
	ctx4 := ctx.Copy()
	ctx4, err = SelectCertPool(pl, ctx4, "service-type:not-exist")
	if err != nil {
		t.Errorf("SelectCertPool with non-matching filter failed: %v", err)
	}
}

// createTestTSLWithCert creates a test TSL with a single certificate and specified service type and status
func createTestTSLWithCert(cert *x509.Certificate, serviceType, status string) *etsi119612.TSL {
	// Use the Base64 encoded certificate from test_utils.go
	certStr := TestCertBase64

	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPInformation: &etsi119612.TSPInformationType{
							TSPName: &etsi119612.InternationalNamesType{
								Name: []*etsi119612.MultiLangNormStringType{
									{
										XmlLangAttr: func() *etsi119612.Lang {
											l := etsi119612.Lang("en")
											return &l
										}(),
										NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
											s := etsi119612.NonEmptyNormalizedString("Test TSP")
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
										TslServiceTypeIdentifier: serviceType,
										TslServiceStatus:         status,
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: []*etsi119612.DigitalIdentityType{
												{
													X509Certificate: certStr,
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

	return tsl
}

// Using TestCertBase64 and TestCert from test_utils.go
