package dsig

import (
	"crypto/x509"
	"testing"

	"github.com/beevik/etree"
	xmldsig "github.com/russellhaering/goxmldsig"
)

// TestVerifyXMLSignatureWithPool verifies that the pool wrapper delegates correctly
// to the core VerifyXMLSignature function.
func TestVerifyXMLSignatureWithPool_Valid(t *testing.T) {
	cert, privateKey := generateTestCert(t, "Pool Test CA")

	// Create and sign a document
	doc := etree.NewDocument()
	root := doc.CreateElement("root")
	root.CreateAttr("ID", "pool-test-id")
	root.CreateElement("data").SetText("pool test")

	ks := &testKeyStore{privateKey: privateKey, certDER: cert.Raw}
	ctx := xmldsig.NewDefaultSigningContext(ks)
	ctx.Canonicalizer = xmldsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")

	signedRoot, err := ctx.SignEnveloped(root)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	signedDoc := etree.NewDocument()
	signedDoc.SetRoot(signedRoot)
	xmlBytes, err := signedDoc.WriteToBytes()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	// Verify using the pool wrapper
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	verified, err := VerifyXMLSignatureWithPool(xmlBytes, pool, []*x509.Certificate{cert})
	if err != nil {
		t.Fatalf("VerifyXMLSignatureWithPool failed: %v", err)
	}
	if verified == nil {
		t.Fatal("expected verified element, got nil")
	}
}

func TestVerifyXMLSignatureWithPool_NoCerts(t *testing.T) {
	pool := x509.NewCertPool()
	_, err := VerifyXMLSignatureWithPool([]byte("<root/>"), pool, nil)
	if err == nil {
		t.Error("expected error for nil certs")
	}
}

// TestTSLSignatureVerifier_Verify tests the TSLSignatureVerifier wrapper.
func TestTSLSignatureVerifier_Verify_Valid(t *testing.T) {
	cert, privateKey := generateTestCert(t, "TSL Verifier CA")

	doc := etree.NewDocument()
	root := doc.CreateElement("TrustServiceStatusList")
	root.CreateAttr("ID", "tsl-verify-id")
	root.CreateElement("SchemeInformation").SetText("test")

	ks := &testKeyStore{privateKey: privateKey, certDER: cert.Raw}
	ctx := xmldsig.NewDefaultSigningContext(ks)
	ctx.Canonicalizer = xmldsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")

	signedRoot, err := ctx.SignEnveloped(root)
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	signedDoc := etree.NewDocument()
	signedDoc.SetRoot(signedRoot)
	xmlBytes, err := signedDoc.WriteToBytes()
	if err != nil {
		t.Fatalf("failed to serialize: %v", err)
	}

	verifier := NewTSLSignatureVerifier([]*x509.Certificate{cert})
	verified, err := verifier.Verify(xmlBytes)
	if err != nil {
		t.Fatalf("TSLSignatureVerifier.Verify failed: %v", err)
	}
	if verified == nil {
		t.Fatal("expected verified element, got nil")
	}
}

func TestTSLSignatureVerifier_Verify_NoCerts(t *testing.T) {
	verifier := NewTSLSignatureVerifier(nil)
	_, err := verifier.Verify([]byte("<root/>"))
	if err == nil {
		t.Error("expected error for nil certs")
	}
}

func TestTSLSignatureVerifier_AddTrustedCertificate(t *testing.T) {
	cert1, _ := generateTestCert(t, "CA1")
	cert2, _ := generateTestCert(t, "CA2")

	verifier := NewTSLSignatureVerifier([]*x509.Certificate{cert1})
	if len(verifier.trustedCerts) != 1 {
		t.Errorf("expected 1 cert, got %d", len(verifier.trustedCerts))
	}

	verifier.AddTrustedCertificate(cert2)
	if len(verifier.trustedCerts) != 2 {
		t.Errorf("expected 2 certs, got %d", len(verifier.trustedCerts))
	}
}
