package etsi119612

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateValidCertBase64 generates a self-signed P-256 certificate and returns its base64 DER encoding.
func generateValidCertBase64(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test Cert"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(der)
}

// generateBrainpoolCertBase64 generates a certificate DER with the EC curve OID replaced
// with an unsupported curve OID, which Go's x509 parser does not support.
// We replace only the OID value bytes (not the TLV envelope) to preserve DER structure.
func generateBrainpoolCertBase64(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Test Brainpool Cert"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	// P-256 OID value bytes (without tag/length): 2a 86 48 ce 3d 03 01 07
	// Replace with a fake OID of the same length so DER structure stays valid
	// but Go's x509 parser will report "unsupported elliptic curve".
	p256OIDValue := []byte{0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07}
	fakeOIDValue := []byte{0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x63} // 1.2.840.10045.3.1.99

	modified := bytes.Replace(der, p256OIDValue, fakeOIDValue, 1)
	if bytes.Equal(modified, der) {
		t.Fatal("Failed to replace P-256 OID in certificate DER")
	}
	return base64.StdEncoding.EncodeToString(modified)
}

func makeSvcWithCerts(certs ...string) *TSPServiceType {
	ids := make([]*DigitalIdentityType, 0, len(certs))
	for _, c := range certs {
		ids = append(ids, &DigitalIdentityType{X509Certificate: c})
	}
	lang := Lang("en")
	nens := NonEmptyNormalizedString("Test Service")
	return &TSPServiceType{
		TslServiceInformation: &TSPServiceInformationType{
			TslServiceDigitalIdentity: &DigitalIdentityListType{DigitalId: ids},
			ServiceName: &InternationalNamesType{
				Name: []*MultiLangNormStringType{{NonEmptyNormalizedString: &nens, XmlLangAttr: &lang}},
			},
		},
	}
}

func TestClassifyCertParseError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected CertParseErrorKind
	}{
		{"unsupported curve", "x509: unsupported elliptic curve", CertParseErrUnsupportedCurve},
		{"invalid RSA modulus", "x509: RSA modulus is not a positive number", CertParseErrInvalidRSA},
		{"malformed RSA encoding", "x509: RSA key missing NULL parameters", CertParseErrMalformedRSA},
		{"invalid RDN sequence", "x509: invalid RDNSequence: invalid attribute value: invalid PrintableString", CertParseErrInvalidASN1},
		{"invalid basic constraints", "x509: invalid basic constraints", CertParseErrInvalidASN1},
		{"unknown error", "x509: something completely different", CertParseErrOther},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &x509ParseError{msg: tt.errMsg}
			assert.Equal(t, tt.expected, ClassifyCertParseError(err))
		})
	}
}

// x509ParseError is a simple error type for testing classification.
type x509ParseError struct{ msg string }

func (e *x509ParseError) Error() string { return e.msg }

func TestCertParseStats(t *testing.T) {
	t.Run("NewCertParseStats", func(t *testing.T) {
		stats := NewCertParseStats()
		assert.Equal(t, 0, stats.Total)
		assert.Equal(t, 0, stats.Parsed)
		assert.Equal(t, 0, stats.TotalSkipped())
		assert.NotNil(t, stats.Skipped)
	})

	t.Run("RecordSuccess", func(t *testing.T) {
		stats := NewCertParseStats()
		stats.RecordSuccess()
		stats.RecordSuccess()
		assert.Equal(t, 2, stats.Total)
		assert.Equal(t, 2, stats.Parsed)
		assert.Equal(t, 0, stats.TotalSkipped())
	})

	t.Run("RecordSkip", func(t *testing.T) {
		stats := NewCertParseStats()
		stats.RecordSkip(CertParseErrUnsupportedCurve)
		stats.RecordSkip(CertParseErrUnsupportedCurve)
		stats.RecordSkip(CertParseErrInvalidRSA)
		assert.Equal(t, 3, stats.Total)
		assert.Equal(t, 0, stats.Parsed)
		assert.Equal(t, 3, stats.TotalSkipped())
		assert.Equal(t, 2, stats.Skipped[CertParseErrUnsupportedCurve])
		assert.Equal(t, 1, stats.Skipped[CertParseErrInvalidRSA])
	})

	t.Run("Merge", func(t *testing.T) {
		s1 := NewCertParseStats()
		s1.RecordSuccess()
		s1.RecordSkip(CertParseErrUnsupportedCurve)

		s2 := NewCertParseStats()
		s2.RecordSuccess()
		s2.RecordSuccess()
		s2.RecordSkip(CertParseErrUnsupportedCurve)
		s2.RecordSkip(CertParseErrInvalidASN1)

		s1.Merge(s2)
		assert.Equal(t, 6, s1.Total)
		assert.Equal(t, 3, s1.Parsed)
		assert.Equal(t, 3, s1.TotalSkipped())
		assert.Equal(t, 2, s1.Skipped[CertParseErrUnsupportedCurve])
		assert.Equal(t, 1, s1.Skipped[CertParseErrInvalidASN1])
	})

	t.Run("Merge nil", func(t *testing.T) {
		s := NewCertParseStats()
		s.RecordSuccess()
		s.Merge(nil)
		assert.Equal(t, 1, s.Total)
	})
}

func TestWithCertificateResults_ValidCert(t *testing.T) {
	validB64 := generateValidCertBase64(t)
	svc := makeSvcWithCerts(validB64)

	var certs []*x509.Certificate
	stats := svc.WithCertificateResults(func(cert *x509.Certificate) {
		certs = append(certs, cert)
	})

	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 1, stats.Parsed)
	assert.Equal(t, 0, stats.TotalSkipped())
	assert.Len(t, certs, 1)
	assert.Equal(t, "Test Cert", certs[0].Subject.CommonName)
}

func TestWithCertificateResults_UnsupportedCurve(t *testing.T) {
	brainpoolB64 := generateBrainpoolCertBase64(t)
	svc := makeSvcWithCerts(brainpoolB64)

	var certs []*x509.Certificate
	stats := svc.WithCertificateResults(func(cert *x509.Certificate) {
		certs = append(certs, cert)
	})

	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 0, stats.Parsed)
	assert.Equal(t, 1, stats.TotalSkipped())
	assert.Equal(t, 1, stats.Skipped[CertParseErrUnsupportedCurve])
	assert.Len(t, certs, 0)
}

func TestWithCertificateResults_MixedCerts(t *testing.T) {
	validB64 := generateValidCertBase64(t)
	brainpoolB64 := generateBrainpoolCertBase64(t)
	svc := makeSvcWithCerts(validB64, brainpoolB64, validB64)

	var certs []*x509.Certificate
	stats := svc.WithCertificateResults(func(cert *x509.Certificate) {
		certs = append(certs, cert)
	})

	assert.Equal(t, 3, stats.Total)
	assert.Equal(t, 2, stats.Parsed)
	assert.Equal(t, 1, stats.TotalSkipped())
	assert.Equal(t, 1, stats.Skipped[CertParseErrUnsupportedCurve])
	assert.Len(t, certs, 2)
}

func TestWithCertificateResults_InvalidBase64(t *testing.T) {
	svc := makeSvcWithCerts("not-valid-base64!!!")

	var certs []*x509.Certificate
	stats := svc.WithCertificateResults(func(cert *x509.Certificate) {
		certs = append(certs, cert)
	})

	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 0, stats.Parsed)
	assert.Equal(t, 1, stats.TotalSkipped())
	assert.Equal(t, 1, stats.Skipped[CertParseErrBase64])
	assert.Len(t, certs, 0)
}

func TestWithCertificateResults_GarbageDER(t *testing.T) {
	// Valid base64 but not a valid certificate DER
	garbageB64 := base64.StdEncoding.EncodeToString([]byte("this is not a certificate"))
	svc := makeSvcWithCerts(garbageB64)

	var certs []*x509.Certificate
	stats := svc.WithCertificateResults(func(cert *x509.Certificate) {
		certs = append(certs, cert)
	})

	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 0, stats.Parsed)
	assert.Equal(t, 1, stats.TotalSkipped())
	assert.Len(t, certs, 0)
}

func TestWithCertificateResults_NilDigitalIdentity(t *testing.T) {
	svc := &TSPServiceType{
		TslServiceInformation: &TSPServiceInformationType{
			TslServiceDigitalIdentity: nil,
		},
	}

	var certs []*x509.Certificate
	stats := svc.WithCertificateResults(func(cert *x509.Certificate) {
		certs = append(certs, cert)
	})

	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Parsed)
	assert.Len(t, certs, 0)
}

func TestWithCertificateResults_NilServiceName(t *testing.T) {
	// Service with an unparseable cert but nil ServiceName — should not panic
	brainpoolB64 := generateBrainpoolCertBase64(t)
	ids := []*DigitalIdentityType{{X509Certificate: brainpoolB64}}
	svc := &TSPServiceType{
		TslServiceInformation: &TSPServiceInformationType{
			TslServiceDigitalIdentity: &DigitalIdentityListType{DigitalId: ids},
			ServiceName:               nil, // nil ServiceName
		},
	}

	// Should not panic
	stats := svc.WithCertificateResults(func(cert *x509.Certificate) {})
	assert.Equal(t, 1, stats.TotalSkipped())
}

func TestWithCertificates_BackwardCompatibility(t *testing.T) {
	// WithCertificates should still work and call the callback for valid certs
	validB64 := generateValidCertBase64(t)
	svc := makeSvcWithCerts(validB64)

	var certs []*x509.Certificate
	svc.WithCertificates(func(cert *x509.Certificate) {
		certs = append(certs, cert)
	})

	assert.Len(t, certs, 1)
}
