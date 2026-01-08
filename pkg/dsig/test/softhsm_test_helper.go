package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// SoftHSMTestHelper provides utilities for testing with SoftHSM
type SoftHSMTestHelper struct {
	TokenDir    string
	TokenName   string
	SlotID      int
	UserPIN     string
	SOUserPIN   string
	LibPath     string
	initialized bool
}

// NewSoftHSMTestHelper creates a new SoftHSM test helper
func NewSoftHSMTestHelper() *SoftHSMTestHelper {
	// Create a unique directory name using timestamp and a random component
	uuid := make([]byte, 8)
	rand.Read(uuid)
	dirName := fmt.Sprintf("go-trust-softhsm-test-%d-%x", time.Now().UnixNano(), uuid)

	return &SoftHSMTestHelper{
		TokenDir:  filepath.Join(os.TempDir(), dirName),
		TokenName: "go-trust-test-token",
		SlotID:    0,
		UserPIN:   "1234",
		SOUserPIN: "5678",
	}
}

// IsSoftHSMAvailable checks if SoftHSM is available on the system
func (h *SoftHSMTestHelper) IsSoftHSMAvailable() bool {
	// Try common paths for libsofthsm2.so
	commonPaths := []string{
		"/usr/lib/softhsm/libsofthsm2.so",
		"/usr/lib/x86_64-linux-gnu/softhsm/libsofthsm2.so",
		"/usr/local/lib/softhsm/libsofthsm2.so",
		"/usr/lib64/softhsm/libsofthsm2.so",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			h.LibPath = path
			return true
		}
	}

	// Check if softhsm2-util is available in PATH
	_, err := exec.LookPath("softhsm2-util")
	return err == nil
}

// Setup creates a new SoftHSM token for testing
func (h *SoftHSMTestHelper) Setup() error {
	// Create token directory
	if err := os.MkdirAll(h.TokenDir, 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	// Configure SoftHSM to use this directory
	confPath := filepath.Join(h.TokenDir, "softhsm2.conf")
	confContent := fmt.Sprintf(`
directories.tokendir = %s
objectstore.backend = file
log.level = INFO
slots.removable = true
`, h.TokenDir)

	if err := os.WriteFile(confPath, []byte(confContent), 0600); err != nil {
		return fmt.Errorf("failed to create SoftHSM configuration file: %w", err)
	}

	// Set environment variable for SoftHSM configuration
	os.Setenv("SOFTHSM2_CONF", confPath)

	// Initialize token in the first free slot
	cmd := exec.Command("softhsm2-util", "--init-token", "--free",
		"--label", h.TokenName,
		"--so-pin", h.SOUserPIN,
		"--pin", h.UserPIN)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to initialize token: %w, output: %s", err, output)
	}

	// Extract the slot ID from the output
	outputStr := string(output)
	for _, line := range strings.Split(outputStr, "\n") {
		if strings.Contains(line, "Slot ") {
			var slotID int
			if _, err := fmt.Sscanf(line, "Slot %d", &slotID); err == nil {
				h.SlotID = slotID
				break
			}
		}
	}

	if strings.Contains(outputStr, "The token has been initialized") {
		h.initialized = true
		return nil
	}

	return fmt.Errorf("failed to initialize token, unexpected output: %s", outputStr)
}

// GenerateAndImportTestCert generates a test certificate and imports it into the token
func (h *SoftHSMTestHelper) GenerateAndImportTestCert(keyLabel, certLabel, keyID string) error {
	if !h.initialized {
		return fmt.Errorf("SoftHSM token not initialized")
	}

	// Check if pkcs11-tool is available
	_, err := exec.LookPath("pkcs11-tool")
	if err != nil {
		return fmt.Errorf("pkcs11-tool not found in PATH, required for testing: %w", err)
	}

	// First, verify the token is accessible
	cmd := exec.Command("softhsm2-util", "--show-slots")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list SoftHSM slots: %w, output: %s", err, output)
	}

	// Check if our token is visible
	outputStr := string(output)
	if !strings.Contains(outputStr, h.TokenName) {
		return fmt.Errorf("token '%s' not found in slot list: %s", h.TokenName, outputStr)
	}

	// If no key ID is provided, generate a unique one
	if keyID == "" {
		keyID = fmt.Sprintf("%02x", time.Now().UnixNano()%0x100)
	}

	// Generate RSA key pair directly in the token using pkcs11-tool
	cmd = exec.Command("pkcs11-tool", "--module", h.LibPath,
		"--token-label", h.TokenName,
		"--login", "--pin", h.UserPIN,
		"--keypairgen", "--key-type", "rsa:2048",
		"--id", keyID,
		"--label", keyLabel)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate keypair in token: %w, output: %s", err, output)
	}

	// Debug: List objects to verify key was created
	cmdList1 := exec.Command("pkcs11-tool", "--module", h.LibPath,
		"--token-label", h.TokenName,
		"--login", "--pin", h.UserPIN,
		"--list-objects")
	debugOutput1, _ := cmdList1.CombinedOutput()
	fmt.Printf("Objects in token after key generation:\n%s\n", string(debugOutput1))

	// Create temporary certificate file
	certFile := filepath.Join(h.TokenDir, "cert.pem")

	// Generate a temporary key for certificate creation
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate temporary private key: %w", err)
	}

	// Create self-signed certificate
	notBefore := time.Now()
	notAfter := notBefore.Add(24 * time.Hour)

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "Go-Trust Test Certificate",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Save certificate to PEM file
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("failed to write certificate to file: %w", err)
	}

	// Import certificate into token
	cmd = exec.Command("pkcs11-tool", "--module", h.LibPath,
		"--token-label", h.TokenName,
		"--login", "--pin", h.UserPIN,
		"--write-object", certFile, "--type", "cert",
		"--id", keyID,
		"--label", certLabel)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to import certificate: %w, output: %s", err, output)
	}

	// Debug: List objects to verify certificate was imported
	cmdList2 := exec.Command("pkcs11-tool", "--module", h.LibPath,
		"--token-label", h.TokenName,
		"--login", "--pin", h.UserPIN,
		"--list-objects")
	debugOutput2, _ := cmdList2.CombinedOutput()
	fmt.Printf("Objects in token after certificate import:\n%s\n", string(debugOutput2))

	// List objects to verify
	cmd = exec.Command("pkcs11-tool", "--module", h.LibPath,
		"--token-label", h.TokenName,
		"--login", "--pin", h.UserPIN,
		"--list-objects")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list objects: %w, output: %s", err, output)
	}

	outputStr = string(output)
	if !strings.Contains(outputStr, keyLabel) || !strings.Contains(outputStr, certLabel) {
		return fmt.Errorf("key or certificate not found in token after import: %s", outputStr)
	}

	return nil
}

// Cleanup removes the temporary directory and token
func (h *SoftHSMTestHelper) Cleanup() error {
	if h.TokenDir != "" {
		fmt.Printf("Cleaning up SoftHSM test environment at: %s\n", h.TokenDir)
		return os.RemoveAll(h.TokenDir)
	}
	return nil
}

// GetPKCS11URI returns the PKCS11 URI for this token
func (h *SoftHSMTestHelper) GetPKCS11URI() string {
	// Get the actual slot information using pkcs11-tool
	cmd := exec.Command("pkcs11-tool", "--module", h.LibPath, "--list-token-slots")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fall back to simple token-based URI if pkcs11-tool fails
		return fmt.Sprintf("pkcs11:module=%s;pin=%s;token=%s", h.LibPath, h.UserPIN, h.TokenName)
	}

	// Parse the output to find the correct slot
	lines := strings.Split(string(output), "\n")
	var slotID string
	for i, line := range lines {
		if strings.Contains(line, "token label") && strings.Contains(line, h.TokenName) {
			// Look for the slot ID in the previous line
			if i > 0 && strings.Contains(lines[i-1], "Slot ") {
				slotParts := strings.Split(lines[i-1], " ")
				for _, part := range slotParts {
					// Look for a hex number in parentheses like (0x1234)
					if strings.HasPrefix(part, "(0x") && strings.HasSuffix(part, ")") {
						slotID = part[1 : len(part)-1] // Remove the parentheses
						break
					}
				}
			}
		}
	}

	// If we found a hex slot ID, use it in the URI
	if slotID != "" {
		return fmt.Sprintf("pkcs11:module=%s;pin=%s;slot=%s", h.LibPath, h.UserPIN, slotID)
	}

	// Fall back to token name
	return fmt.Sprintf("pkcs11:module=%s;pin=%s;token=%s", h.LibPath, h.UserPIN, h.TokenName)
}

// SkipIfSoftHSMUnavailable skips the test if SoftHSM is not available
func SkipIfSoftHSMUnavailable(t *testing.T) *SoftHSMTestHelper {
	helper := NewSoftHSMTestHelper()
	if !helper.IsSoftHSMAvailable() {
		t.Skip("Skipping test: SoftHSM not available")
		return nil
	}
	return helper
}
