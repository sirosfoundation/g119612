package pipeline

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/sirosfoundation/g119612/pkg/etsi119612"
	"github.com/sirosfoundation/g119612/pkg/logging"
)

func TestPublishTSL_WithSignature(t *testing.T) {
	// Create a temporary directory for output files
	tempDir := t.TempDir()

	// Create a temporary directory for certificate and key
	certDir := t.TempDir()
	certFile := filepath.Join(certDir, "cert.pem")
	keyFile := filepath.Join(certDir, "key.pem")

	// Generate a test certificate and key
	if err := generateTestCertAndKey(certFile, keyFile); err != nil {
		t.Fatalf("Failed to generate test certificate and key: %v", err)
	}

	// Create a test pipeline context with TSLs
	ctx := &Context{}

	// Add a test TSL with a distribution point
	tsl := generateTSL("Test Service 1", "http://uri.etsi.org/TrstSvc/Svctype/CA/QC", []string{TestCertBase64})
	tsl.StatusList.TslSchemeInformation.TslDistributionPoints = &etsi119612.NonEmptyURIListType{
		URI: []string{"https://example.com/test-tsl.xml"},
	}
	ctx.EnsureTSLStack().TSLs.Push(tsl)

	// Run the PublishTSL function with signature
	pl := &Pipeline{
		Logger: logging.NewLogger(logging.DebugLevel),
	}
	result, err := PublishTSL(pl, ctx, tempDir, certFile, keyFile)
	if err != nil {
		t.Fatalf("PublishTSL failed: %v", err)
	}

	// Check the result
	if result == nil {
		t.Fatal("PublishTSL returned nil context")
	}

	// Verify files were created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}

	// Verify file content contains a signature
	filePath := filepath.Join(tempDir, files[0].Name())
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Parse XML and check for Signature element
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Find Signature element
	signature := doc.FindElement("//Signature")
	if signature == nil {
		t.Fatal("XML-DSIG Signature element not found in output XML")
	}
}

// generateTestCertAndKey creates a self-signed certificate and private key for testing
func generateTestCertAndKey(certFile, keyFile string) error {
	// Generate a private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create a certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   "Test Certificate",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create a self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// Write certificate to file
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return err
	}

	// Write private key to file
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}); err != nil {
		return err
	}

	return nil
}
