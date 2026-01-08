package pipeline

import (
	"bytes"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
)

// Test certificate variables for reuse in tests
var TestCertBase64 string
var TestCertDER []byte
var TestCert *x509.Certificate

// generateTestCertBase64 runs openssl to generate a self-signed cert and returns the base64-encoded DER string.
func GenerateTestCertBase64() (string, []byte, *x509.Certificate, error) {
	keyFile, err := os.CreateTemp("", "testkey-*.pem")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp key file: %w", err)
	}
	defer os.Remove(keyFile.Name())
	keyFile.Close()
	certFile, err := os.CreateTemp("", "testcert-*.pem")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp cert file: %w", err)
	}
	defer os.Remove(certFile.Name())
	certFile.Close()
	derFile, err := os.CreateTemp("", "testcert-*.der")
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp der file: %w", err)
	}
	defer os.Remove(derFile.Name())
	derFile.Close()

	opensslCmd := fmt.Sprintf("openssl req -x509 -newkey rsa:2048 -keyout %s -out %s -days 365 -nodes -subj '/CN=Test Cert' 2>/dev/null && openssl x509 -outform der -in %s -out %s 2>/dev/null && openssl base64 -in %s -A 2>/dev/null", keyFile.Name(), certFile.Name(), certFile.Name(), derFile.Name(), derFile.Name())
	cmd := exec.Command("bash", "-c", opensslCmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	output := out.String()
	if err != nil {
		return "", nil, nil, fmt.Errorf("openssl error: %v\noutput: %s", err, output)
	}
	certBase64 := strings.TrimSpace(output)
	certDER, err := base64.StdEncoding.DecodeString(certBase64)
	if err != nil {
		return certBase64, nil, nil, fmt.Errorf("base64 decode error: %v\noutput: %s", err, output)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return certBase64, certDER, nil, fmt.Errorf("parse cert error: %v\noutput: %s", err, output)
	}
	return certBase64, certDER, cert, nil
}

func init() {
	var err error
	TestCertBase64, TestCertDER, TestCert, err = GenerateTestCertBase64()
	if err != nil {
		panic("failed to generate test cert: " + err.Error())
	}
}

// Test versions of step functions
func loadTSL(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	return LoadTSL(pl, ctx, args...)
}

func selectCertPool(pl *Pipeline, ctx *Context, args ...string) (*Context, error) {
	return SelectCertPool(pl, ctx, args...)
}

// generateTSL creates a test TSL with the given certificates
func generateTSL(serviceName string, serviceType string, certs []string) *etsi119612.TSL {
	var digitalIds []*etsi119612.DigitalIdentityType
	for _, cert := range certs {
		digitalIds = append(digitalIds, &etsi119612.DigitalIdentityType{
			X509Certificate: cert,
		})
	}

	return &etsi119612.TSL{
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
					{
						TslTSPInformation: &etsi119612.TSPInformationType{
							TSPName: &etsi119612.InternationalNamesType{
								Name: []*etsi119612.MultiLangNormStringType{
									{
										XmlLangAttr: func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
										NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
											s := etsi119612.NonEmptyNormalizedString("Test Provider")
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
										TslServiceStatus:         etsi119612.ServiceStatusGranted,
										ServiceName: &etsi119612.InternationalNamesType{
											Name: []*etsi119612.MultiLangNormStringType{
												{
													XmlLangAttr: func() *etsi119612.Lang { l := etsi119612.Lang("en"); return &l }(),
													NonEmptyNormalizedString: func() *etsi119612.NonEmptyNormalizedString {
														s := etsi119612.NonEmptyNormalizedString(serviceName)
														return &s
													}(),
												},
											},
										},
										TslServiceDigitalIdentity: &etsi119612.DigitalIdentityListType{
											DigitalId: digitalIds,
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
}
