package dsig

import (
	"testing"

	"github.com/ThalesGroup/crypto11"
	"github.com/sirosfoundation/g119612/pkg/dsig/test"
)

func TestNewPKCS11Signer(t *testing.T) {
	// Create a minimal configuration for testing creation (not connection)
	config := &crypto11.Config{
		Path:       "/path/to/module",
		TokenLabel: "test-token",
		Pin:        "1234",
	}

	// Test basic creation
	signer := NewPKCS11Signer(config, "key-label", "cert-label")
	if signer == nil {
		t.Fatal("Failed to create PKCS11Signer")
	}

	// Check default values
	if signer.keyLabel != "key-label" {
		t.Errorf("Expected key label to be 'key-label', got '%s'", signer.keyLabel)
	}
	if signer.certLabel != "cert-label" {
		t.Errorf("Expected cert label to be 'cert-label', got '%s'", signer.certLabel)
	}
	if signer.keyID != "01" {
		t.Errorf("Expected default key ID to be '01', got '%s'", signer.keyID)
	}

	// Test SetKeyID
	signer.SetKeyID("42")
	if signer.keyID != "42" {
		t.Errorf("Expected key ID to be '42' after SetKeyID, got '%s'", signer.keyID)
	}
}

func TestNewPKCS11SignerFromURI(t *testing.T) {
	// Test with valid URI
	signer, err := NewPKCS11SignerFromURI(
		"pkcs11:module=/usr/lib/softhsm/libsofthsm2.so;pin=1234;slot-id=0",
		"key-label",
		"cert-label",
	)
	if err != nil {
		t.Fatalf("Failed to create PKCS11Signer from URI: %v", err)
	}
	if signer == nil {
		t.Fatal("PKCS11Signer from URI is nil despite no error")
	}

	// Test with invalid URI
	_, err = NewPKCS11SignerFromURI("invalid-uri", "key-label", "cert-label")
	if err == nil {
		t.Fatal("Expected error for invalid URI, got nil")
	}
}

func TestPKCS11SignerWithSoftHSM(t *testing.T) {
	// Skip if CI or SoftHSM not available
	if helper := test.SkipIfSoftHSMUnavailable(t); helper != nil {
		// Set up SoftHSM token
		if err := helper.Setup(); err != nil {
			t.Skipf("Could not set up SoftHSM token: %v", err)
		}
		defer helper.Cleanup()

		// Generate and import test key pair
		keyLabel := "test-key"
		certLabel := "test-cert"
		keyID := "01"
		if err := helper.GenerateAndImportTestCert(keyLabel, certLabel, keyID); err != nil {
			t.Skipf("Could not import test certificate: %v", err)
		}

		// Get PKCS11 URI and create signer
		uri := helper.GetPKCS11URI()
		signer, err := NewPKCS11SignerFromURI(uri, keyLabel, certLabel)
		if err != nil {
			t.Fatalf("Failed to create PKCS11Signer: %v", err)
		}
		defer signer.Close()

		// Test XML signing
		xmlData := []byte("<test>Test XML for PKCS11 signing</test>")
		signedData, err := signer.Sign(xmlData)
		if err != nil {
			t.Fatalf("Failed to sign XML: %v", err)
		}
		if len(signedData) <= len(xmlData) {
			t.Fatal("Signed data should be longer than original")
		}
	}
}
