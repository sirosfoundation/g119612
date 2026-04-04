package etsi119612_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/h2non/gock"
	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testServiceStatus = "https://uri.etsi.org/TrstSvc/TrustedList/Svcstatus/granted/"

// TestFetchTSLWithAllReferences is a thin wrapper around FetchTSLWithReferencesAndOptions.
func TestFetchTSLWithAllReferences(t *testing.T) {
	defer gock.Off()

	gock.New("https://example.com").
		Get("/main.xml").
		Reply(200).
		File("testdata/TSL-with-pointer.xml")

	gock.New("https://example.com").
		Get("/referenced.xml").
		Reply(200).
		File("testdata/EWC-TL.xml")

	tsls, err := etsi119612.FetchTSLWithAllReferences("https://example.com/main.xml")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tsls), 1)
}

func TestFetchTSLWithAllReferences_InvalidURL(t *testing.T) {
	defer gock.Off()

	gock.New("https://example.com").
		Get("/missing.xml").
		Reply(404).
		BodyString("not found")

	_, err := etsi119612.FetchTSLWithAllReferences("https://example.com/missing.xml")
	assert.Error(t, err)
}

// TestToCertPoolWithReferences tests the method that builds a cert pool from
// the TSL and all its referenced TSLs.
func TestToCertPoolWithReferences(t *testing.T) {
	// Generate test certificates
	cert1 := generateTestCert119612(t, "Main CA")
	cert2 := generateTestCert119612(t, "Referenced CA")

	b64Cert1 := base64.StdEncoding.EncodeToString(cert1.Raw)
	b64Cert2 := base64.StdEncoding.EncodeToString(cert2.Raw)

	// Build a TSL with a service that has cert1
	mainTSL := &etsi119612.TSL{
		Source: "https://main.example.com/tsl.xml",
		StatusList: etsi119612.TrustStatusListType{
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC",
										TslServiceStatus:         testServiceStatus,
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: []*etsi119612.DigitalIdentityType{
												{X509Certificate: b64Cert1},
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

	// Build a referenced TSL with cert2
	refTSL := &etsi119612.TSL{
		Source: "https://ref.example.com/tsl.xml",
		StatusList: etsi119612.TrustStatusListType{
			TslTrustServiceProviderList: &etsi119612.TrustServiceProviderListType{
				TslTrustServiceProvider: []*etsi119612.TSPType{
					{
						TslTSPServices: &etsi119612.TSPServicesListType{
							TslTSPService: []*etsi119612.TSPServiceType{
								{
									TslServiceInformation: &etsi119612.TSPServiceInformationType{
										TslServiceTypeIdentifier: "http://uri.etsi.org/TrstSvc/Svctype/CA/QC",
										TslServiceStatus:         testServiceStatus,
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: []*etsi119612.DigitalIdentityType{
												{X509Certificate: b64Cert2},
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

	mainTSL.AddReferencedTSL(refTSL)

	pool := mainTSL.ToCertPoolWithReferences(etsi119612.PolicyAll)
	require.NotNil(t, pool)
	// Verify both certs were added by using the pool for self-signed verification
	for _, cert := range []*x509.Certificate{cert1, cert2} {
		_, err := cert.Verify(x509.VerifyOptions{Roots: pool, CurrentTime: time.Now()})
		assert.NoError(t, err, "cert %s should be verifiable via the pool", cert.Subject.CommonName)
	}
}

func TestToCertPoolWithReferences_NoReferenced(t *testing.T) {
	cert := generateTestCert119612(t, "Solo CA")
	b64Cert := base64.StdEncoding.EncodeToString(cert.Raw)

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
										TslServiceStatus:         testServiceStatus,
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: []*etsi119612.DigitalIdentityType{
												{X509Certificate: b64Cert},
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

	pool := tsl.ToCertPoolWithReferences(etsi119612.PolicyAll)
	assert.NotNil(t, pool)
	opts := x509.VerifyOptions{Roots: pool}
	_, err := cert.Verify(opts)
	assert.NoError(t, err)
}

func TestToCertPoolWithReferences_NilReferencedElements(t *testing.T) {
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{},
	}
	tsl.AddReferencedTSL(nil) // should not panic

	pool := tsl.ToCertPoolWithReferences(etsi119612.PolicyAll)
	assert.NotNil(t, pool)
}

// TestDereferencePointersToOtherTSL_NoPointers verifies no-op on TSL without pointers.
func TestDereferencePointersToOtherTSL_NoPointers(t *testing.T) {
	tsl := &etsi119612.TSL{
		StatusList: etsi119612.TrustStatusListType{
			TslSchemeInformation: &etsi119612.TSLSchemeInformationType{},
		},
	}
	tsl.DereferencePointersToOtherTSL()
	assert.Nil(t, tsl.Referenced)
}

func TestDereferencePointersToOtherTSL_NilSchemeInfo(t *testing.T) {
	tsl := &etsi119612.TSL{}
	tsl.DereferencePointersToOtherTSL()
	assert.Nil(t, tsl.Referenced)
}

func generateTestCert119612(t *testing.T, cn string) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		IsCA:         true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}
