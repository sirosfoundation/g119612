package dsig

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
)

// setupTestCert generates a self-signed certificate and key for testing.
// Returns cert path and key path.
func setupTestCert(t *testing.T) (string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test private key: %v", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		t.Fatalf("failed to generate certificate serial number: %v", err)
	}

	notBefore := time.Now().Add(-time.Hour)
	notAfter := notBefore.Add(24 * time.Hour)

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "XAdES Test Certificate",
			Organization: []string{"Test Org"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create test certificate: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("failed to write test certificate: %v", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		t.Fatalf("failed to write test private key: %v", err)
	}
	return certPath, keyPath
}

func TestFileSigner_XAdES_QualifyingProperties(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping XAdES test in CI environment")
	}

	certPath, keyPath := setupTestCert(t)

	signer := NewFileSigner(certPath, keyPath)
	// XAdES is on by default

	xmlData := []byte(`<?xml version="1.0" encoding="UTF-8"?><TrustServiceStatusList xmlns="http://uri.etsi.org/02231/v2#"><SchemeInformation><TSLType>test</TSLType></SchemeInformation></TrustServiceStatusList>`)

	signedData, err := signer.Sign(xmlData)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(signedData); err != nil {
		t.Fatalf("Failed to parse signed XML: %v", err)
	}

	root := doc.Root()

	// Verify ds:Signature exists
	sig := root.FindElement("//ds:Signature")
	if sig == nil {
		sig = root.FindElement("//Signature")
	}
	if sig == nil {
		t.Fatal("ds:Signature element not found in signed output")
	}

	// Verify QualifyingProperties
	qp := sig.FindElement(".//xades:QualifyingProperties")
	if qp == nil {
		qp = sig.FindElement(".//QualifyingProperties")
	}
	if qp == nil {
		t.Fatal("xades:QualifyingProperties not found")
	}

	// Verify Target attribute on QualifyingProperties
	sigID := sig.SelectAttrValue("Id", "")
	if sigID == "" {
		t.Fatal("ds:Signature missing Id attribute")
	}
	target := qp.SelectAttrValue("Target", "")
	if target != "#"+sigID {
		t.Errorf("QualifyingProperties Target: got %q, want %q", target, "#"+sigID)
	}

	// Verify SignedProperties with Id
	sp := qp.FindElement(".//xades:SignedProperties")
	if sp == nil {
		sp = qp.FindElement(".//SignedProperties")
	}
	if sp == nil {
		t.Fatal("xades:SignedProperties not found")
	}
	spID := sp.SelectAttrValue("Id", "")
	if spID == "" {
		t.Fatal("xades:SignedProperties missing Id attribute")
	}

	// Verify SigningTime
	sigTime := sp.FindElement(".//xades:SigningTime")
	if sigTime == nil {
		sigTime = sp.FindElement(".//SigningTime")
	}
	if sigTime == nil {
		t.Fatal("xades:SigningTime not found")
	}
	if sigTime.Text() == "" {
		t.Fatal("xades:SigningTime is empty")
	}

	// Verify SigningCertificate
	sigCert := sp.FindElement(".//xades:SigningCertificate")
	if sigCert == nil {
		sigCert = sp.FindElement(".//SigningCertificate")
	}
	if sigCert == nil {
		t.Fatal("xades:SigningCertificate not found")
	}

	// Verify CertDigest
	certDigest := sigCert.FindElement(".//xades:CertDigest")
	if certDigest == nil {
		certDigest = sigCert.FindElement(".//CertDigest")
	}
	if certDigest == nil {
		t.Fatal("xades:CertDigest not found")
	}

	// Verify DataObjectFormat
	dof := sp.FindElement(".//xades:DataObjectFormat")
	if dof == nil {
		dof = sp.FindElement(".//DataObjectFormat")
	}
	if dof == nil {
		t.Fatal("xades:DataObjectFormat not found")
	}
	mimeType := dof.FindElement(".//xades:MimeType")
	if mimeType == nil {
		mimeType = dof.FindElement(".//MimeType")
	}
	if mimeType == nil || mimeType.Text() != "text/xml" {
		t.Error("xades:MimeType should be 'text/xml'")
	}
}

func TestFileSigner_XAdES_TwoReferences(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping XAdES test in CI environment")
	}

	certPath, keyPath := setupTestCert(t)

	signer := NewFileSigner(certPath, keyPath)

	xmlData := []byte(`<Root><Data>test</Data></Root>`)
	signedData, err := signer.Sign(xmlData)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(signedData); err != nil {
		t.Fatalf("Failed to parse signed XML: %v", err)
	}

	// Find SignedInfo
	si := doc.Root().FindElement("//ds:SignedInfo")
	if si == nil {
		si = doc.Root().FindElement("//SignedInfo")
	}
	if si == nil {
		t.Fatal("ds:SignedInfo not found")
	}

	// Count references - XAdES-B-B requires exactly 2
	refs := si.FindElements("ds:Reference")
	if len(refs) == 0 {
		refs = si.FindElements("Reference")
	}
	if len(refs) != 2 {
		t.Fatalf("Expected 2 ds:Reference elements in SignedInfo (document + SignedProperties), got %d", len(refs))
	}

	// One reference should have empty URI (document), other should have Type=SignedProperties
	var hasDocRef, hasPropsRef bool
	for _, ref := range refs {
		uri := ref.SelectAttrValue("URI", "none")
		refType := ref.SelectAttrValue("Type", "")
		if uri == "" {
			hasDocRef = true
		}
		if refType == signedPropertiesType {
			hasPropsRef = true
		}
	}

	if !hasDocRef {
		t.Error("Missing document reference (URI=\"\")")
	}
	if !hasPropsRef {
		t.Errorf("Missing SignedProperties reference (Type=%q)", signedPropertiesType)
	}
}

func TestFileSigner_XAdES_Disabled(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping XAdES test in CI environment")
	}

	certPath, keyPath := setupTestCert(t)

	signer := NewFileSigner(certPath, keyPath)
	signer.SetXAdES(false)

	xmlData := []byte(`<Root><Data>test</Data></Root>`)
	signedData, err := signer.Sign(xmlData)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(signedData); err != nil {
		t.Fatalf("Failed to parse signed XML: %v", err)
	}

	signedStr := string(signedData)

	// When XAdES is disabled, there should be no QualifyingProperties
	if strings.Contains(signedStr, "QualifyingProperties") {
		t.Error("XAdES disabled but QualifyingProperties found in output")
	}
	if strings.Contains(signedStr, "SignedProperties") {
		t.Error("XAdES disabled but SignedProperties found in output")
	}

	// But there should still be a Signature
	sig := doc.Root().FindElement("//Signature")
	if sig == nil {
		t.Fatal("No Signature found - plain XML-DSIG should still produce a signature")
	}
}

func TestFileSigner_XAdES_CertDigestMatchesCert(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping XAdES test in CI environment")
	}

	certPath, keyPath := setupTestCert(t)

	// Load the certificate to compute expected digest
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("Failed to read cert: %v", err)
	}
	block, _ := pem.Decode(certPEM)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse cert: %v", err)
	}
	expectedHash := sha256.Sum256(cert.Raw)
	expectedDigest := base64.StdEncoding.EncodeToString(expectedHash[:])

	signer := NewFileSigner(certPath, keyPath)

	xmlData := []byte(`<Root><Data>test</Data></Root>`)
	signedData, err := signer.Sign(xmlData)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(signedData); err != nil {
		t.Fatalf("Failed to parse signed XML: %v", err)
	}

	// Find CertDigest/DigestValue
	dv := doc.Root().FindElement("//xades:CertDigest/ds:DigestValue")
	if dv == nil {
		dv = doc.Root().FindElement("//CertDigest/DigestValue")
	}
	if dv == nil {
		t.Fatal("CertDigest/DigestValue not found")
	}

	actualDigest := dv.Text()
	if actualDigest != expectedDigest {
		t.Errorf("CertDigest mismatch:\n  got:  %s\n  want: %s", actualDigest, expectedDigest)
	}
}
