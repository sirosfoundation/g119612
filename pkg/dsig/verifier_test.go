package dsig

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/beevik/etree"
	xmldsig "github.com/russellhaering/goxmldsig"
)

// generateTestCert creates a self-signed test certificate for testing
func generateTestCert(t *testing.T, cn string) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{"Test Organization"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	return cert, privateKey
}

// testKeyStore implements xmldsig.X509KeyStore for testing
type testKeyStore struct {
	privateKey *rsa.PrivateKey
	certDER    []byte
}

func (ks *testKeyStore) GetKeyPair() (*rsa.PrivateKey, []byte, error) {
	return ks.privateKey, ks.certDER, nil
}

func TestCertPoolStore_Certificates(t *testing.T) {
	cert, _ := generateTestCert(t, "Test CA")

	pool := x509.NewCertPool()
	pool.AddCert(cert)

	store := NewCertPoolStore(pool, []*x509.Certificate{cert})

	certs, err := store.Certificates()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(certs) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(certs))
	}

	if certs[0].Subject.CommonName != "Test CA" {
		t.Errorf("expected CN 'Test CA', got '%s'", certs[0].Subject.CommonName)
	}
}

func TestVerifyXMLSignature_NoCertificates(t *testing.T) {
	_, err := VerifyXMLSignature([]byte("<root/>"), nil)
	if err == nil {
		t.Error("expected error for nil certificates")
	}

	_, err = VerifyXMLSignature([]byte("<root/>"), []*x509.Certificate{})
	if err == nil {
		t.Error("expected error for empty certificates")
	}
}

func TestVerifyXMLSignature_InvalidXML(t *testing.T) {
	cert, _ := generateTestCert(t, "Test CA")

	_, err := VerifyXMLSignature([]byte("not xml"), []*x509.Certificate{cert})
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestVerifyXMLSignature_NoSignature(t *testing.T) {
	cert, _ := generateTestCert(t, "Test CA")

	xml := []byte(`<?xml version="1.0"?><root><data>test</data></root>`)

	_, err := VerifyXMLSignature(xml, []*x509.Certificate{cert})
	if err == nil {
		t.Error("expected error for XML without signature")
	}
}

func TestVerifyXMLSignature_ValidSignature(t *testing.T) {
	cert, privateKey := generateTestCert(t, "Test CA")

	// Create a simple XML document
	doc := etree.NewDocument()
	root := doc.CreateElement("root")
	root.CreateAttr("ID", "test-id")
	data := root.CreateElement("data")
	data.SetText("test content")

	// Sign the document using goxmldsig
	keyStore := &testKeyStore{
		privateKey: privateKey,
		certDER:    cert.Raw,
	}

	ctx := xmldsig.NewDefaultSigningContext(keyStore)
	ctx.Canonicalizer = xmldsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")

	signedRoot, err := ctx.SignEnveloped(root)
	if err != nil {
		t.Fatalf("failed to sign XML: %v", err)
	}

	signedDoc := etree.NewDocument()
	signedDoc.SetRoot(signedRoot)
	signedXML, err := signedDoc.WriteToBytes()
	if err != nil {
		t.Fatalf("failed to serialize signed XML: %v", err)
	}

	// Verify the signature
	verified, err := VerifyXMLSignature(signedXML, []*x509.Certificate{cert})
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}

	if verified == nil {
		t.Error("expected non-nil verified element")
	}
}

func TestVerifyXMLSignature_WrongCertificate(t *testing.T) {
	cert1, privateKey1 := generateTestCert(t, "Signer CA")
	cert2, _ := generateTestCert(t, "Other CA")

	// Create and sign with cert1
	doc := etree.NewDocument()
	root := doc.CreateElement("root")
	root.CreateAttr("ID", "test-id")

	keyStore := &testKeyStore{
		privateKey: privateKey1,
		certDER:    cert1.Raw,
	}

	ctx := xmldsig.NewDefaultSigningContext(keyStore)
	ctx.Canonicalizer = xmldsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")

	signedRoot, err := ctx.SignEnveloped(root)
	if err != nil {
		t.Fatalf("failed to sign XML: %v", err)
	}

	signedDoc := etree.NewDocument()
	signedDoc.SetRoot(signedRoot)
	signedXML, err := signedDoc.WriteToBytes()
	if err != nil {
		t.Fatalf("failed to serialize signed XML: %v", err)
	}

	// Try to verify with cert2 (should fail)
	_, err = VerifyXMLSignature(signedXML, []*x509.Certificate{cert2})
	if err == nil {
		t.Error("expected verification to fail with wrong certificate")
	}
}

func TestTSLSignatureVerifier(t *testing.T) {
	cert, _ := generateTestCert(t, "TSL Signer")

	verifier := NewTSLSignatureVerifier([]*x509.Certificate{cert})

	if len(verifier.TrustedCertificates()) != 1 {
		t.Errorf("expected 1 trusted certificate, got %d", len(verifier.TrustedCertificates()))
	}

	// Add another certificate
	cert2, _ := generateTestCert(t, "TSL Signer 2")
	verifier.AddTrustedCertificate(cert2)

	if len(verifier.TrustedCertificates()) != 2 {
		t.Errorf("expected 2 trusted certificates, got %d", len(verifier.TrustedCertificates()))
	}
}
