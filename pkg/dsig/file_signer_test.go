package dsig

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFileSigner(t *testing.T) {
	// Skip test if we're in CI
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping FileSigner test in CI environment")
	}

	// Check if we can run openssl to generate a test certificate
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("Skipping test: openssl not available")
	}

	// Create temp test directory
	tmpDir, err := os.MkdirTemp("", "dsig-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test certificate and key paths
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	// Generate self-signed certificate using openssl
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", keyPath, "-out", certPath, "-days", "1", "-nodes",
		"-subj", "/CN=Test Certificate")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("Failed to generate test certificate: %v, output: %s", err, output)
		return
	}

	// Create FileSigner
	signer := NewFileSigner(certPath, keyPath)

	// Test XML data
	xmlData := []byte(`<test>Test XML for signing</test>`)

	// Sign the XML
	signedData, err := signer.Sign(xmlData)
	if err != nil {
		t.Fatalf("Signing failed: %v", err)
	}

	// Verify that signature was added
	if len(signedData) <= len(xmlData) {
		t.Fatal("Signed data should be longer than original")
	}
}

func TestToXMLDSigSigner(t *testing.T) {
	// Skip test if we're in CI
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping ToXMLDSigSigner test in CI environment")
	}

	// Check if we can run openssl to generate a test certificate
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("Skipping test: openssl not available")
	}

	// Create temp test directory
	tmpDir, err := os.MkdirTemp("", "dsig-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("Success with PKCS1 key", func(t *testing.T) {
		// Create test certificate and key paths
		certPath := filepath.Join(tmpDir, "cert.pem")
		keyPath := filepath.Join(tmpDir, "key.pem")

		// Generate self-signed certificate using openssl (default PKCS1)
		cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
			"-keyout", keyPath, "-out", certPath, "-days", "1", "-nodes",
			"-subj", "/CN=Test Certificate")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Skipf("Failed to generate test certificate: %v, output: %s", err, output)
			return
		}

		// Create FileSigner
		signer := NewFileSigner(certPath, keyPath)

		// Convert to XMLDSig signer
		xmldsigSigner, err := signer.ToXMLDSigSigner()
		if err != nil {
			t.Fatalf("ToXMLDSigSigner() failed: %v", err)
		}

		if xmldsigSigner == nil {
			t.Fatal("ToXMLDSigSigner() returned nil signer")
		}
	})

	t.Run("Success with PKCS8 key", func(t *testing.T) {
		// Create test certificate and key paths
		certPath := filepath.Join(tmpDir, "cert_pkcs8.pem")
		keyPath := filepath.Join(tmpDir, "key_pkcs1.pem")
		keyPath8 := filepath.Join(tmpDir, "key_pkcs8.pem")

		// Generate self-signed certificate using openssl (default PKCS1)
		cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
			"-keyout", keyPath, "-out", certPath, "-days", "1", "-nodes",
			"-subj", "/CN=Test Certificate PKCS8")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Skipf("Failed to generate test certificate: %v, output: %s", err, output)
			return
		}

		// Convert key to PKCS8 format
		cmd = exec.Command("openssl", "pkcs8", "-topk8", "-nocrypt",
			"-in", keyPath, "-out", keyPath8)
		output, err = cmd.CombinedOutput()
		if err != nil {
			t.Skipf("Failed to convert key to PKCS8: %v, output: %s", err, output)
			return
		}

		// Create FileSigner with PKCS8 key
		signer := NewFileSigner(certPath, keyPath8)

		// Convert to XMLDSig signer
		xmldsigSigner, err := signer.ToXMLDSigSigner()
		if err != nil {
			t.Fatalf("ToXMLDSigSigner() with PKCS8 key failed: %v", err)
		}

		if xmldsigSigner == nil {
			t.Fatal("ToXMLDSigSigner() returned nil signer")
		}
	})

	t.Run("Missing certificate file", func(t *testing.T) {
		signer := NewFileSigner("/nonexistent/cert.pem", "/nonexistent/key.pem")

		_, err := signer.ToXMLDSigSigner()
		if err == nil {
			t.Fatal("ToXMLDSigSigner() should fail with missing certificate file")
		}
	})

	t.Run("Missing key file", func(t *testing.T) {
		// Create only certificate file
		certPath := filepath.Join(tmpDir, "cert_only.pem")
		keyPath := filepath.Join(tmpDir, "key_temp.pem")

		// Generate certificate
		cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
			"-keyout", keyPath, "-out", certPath, "-days", "1", "-nodes",
			"-subj", "/CN=Test Certificate")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Skipf("Failed to generate test certificate: %v, output: %s", err, output)
			return
		}

		// Remove key file
		os.Remove(keyPath)

		signer := NewFileSigner(certPath, keyPath)

		_, err = signer.ToXMLDSigSigner()
		if err == nil {
			t.Fatal("ToXMLDSigSigner() should fail with missing key file")
		}
	})

	t.Run("Invalid certificate PEM", func(t *testing.T) {
		// Create files with invalid PEM
		certPath := filepath.Join(tmpDir, "invalid_cert.pem")
		keyPath := filepath.Join(tmpDir, "valid_key.pem")

		// Write invalid cert
		if err := os.WriteFile(certPath, []byte("not a PEM file"), 0600); err != nil {
			t.Fatalf("Failed to write invalid cert: %v", err)
		}

		// Generate valid key
		cmd := exec.Command("openssl", "genrsa", "-out", keyPath, "2048")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Skipf("Failed to generate test key: %v, output: %s", err, output)
			return
		}

		signer := NewFileSigner(certPath, keyPath)

		_, err = signer.ToXMLDSigSigner()
		if err == nil {
			t.Fatal("ToXMLDSigSigner() should fail with invalid certificate PEM")
		}
	})

	t.Run("Invalid key PEM", func(t *testing.T) {
		// Create test certificate and invalid key
		certPath := filepath.Join(tmpDir, "valid_cert.pem")
		keyPath := filepath.Join(tmpDir, "valid_key_tmp.pem")
		invalidKeyPath := filepath.Join(tmpDir, "invalid_key.pem")

		// Generate valid certificate
		cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
			"-keyout", keyPath, "-out", certPath, "-days", "1", "-nodes",
			"-subj", "/CN=Test Certificate")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Skipf("Failed to generate test certificate: %v, output: %s", err, output)
			return
		}

		// Write invalid key
		if err := os.WriteFile(invalidKeyPath, []byte("not a PEM file"), 0600); err != nil {
			t.Fatalf("Failed to write invalid key: %v", err)
		}

		signer := NewFileSigner(certPath, invalidKeyPath)

		_, err = signer.ToXMLDSigSigner()
		if err == nil {
			t.Fatal("ToXMLDSigSigner() should fail with invalid key PEM")
		}
	})
}
