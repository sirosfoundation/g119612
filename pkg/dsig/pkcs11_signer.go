package dsig

import (
	"crypto"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ThalesGroup/crypto11"
	xmldsig "github.com/russellhaering/goxmldsig"
)

// PKCS11Signer implements XMLSigner using a PKCS#11 hardware token.
// This type provides XML digital signature functionality using keys stored in
// Hardware Security Modules (HSMs) or other PKCS#11-compatible devices.
type PKCS11Signer struct {
	// Config contains the PKCS#11 module configuration (path, PIN, etc.)
	Config *crypto11.Config

	// context is the initialized crypto11 context for the PKCS#11 module
	context *crypto11.Context

	// keyLabel is the label used to identify the private key in the HSM
	keyLabel string

	// certLabel is the label used to identify the certificate in the HSM
	certLabel string

	// keyID is the ID for the key and certificate (usually same for both)
	keyID string

	// initialized indicates if the PKCS#11 context has been initialized
	initialized bool
}

// NewPKCS11Signer creates a new PKCS11Signer from a PKCS#11 configuration and key/cert labels.
// This constructor initializes a signer with the provided configuration but does not
// connect to the PKCS#11 module until signing operations are performed.
//
// Parameters:
//   - config: PKCS#11 module configuration (path, PIN, token label, etc.)
//   - keyLabel: Label used to identify the private key in the HSM
//   - certLabel: Label used to identify the certificate in the HSM
//
// Returns:
//   - A new PKCS11Signer with default key ID "01"
func NewPKCS11Signer(config *crypto11.Config, keyLabel, certLabel string) *PKCS11Signer {
	return &PKCS11Signer{
		Config:    config,
		keyLabel:  keyLabel,
		certLabel: certLabel,
		keyID:     "01", // Default ID, can be set with SetKeyID
	}
}

// NewPKCS11SignerFromURI creates a new PKCS11Signer from a PKCS#11 URI.
// This convenience constructor parses a PKCS#11 URI (RFC 7512) to extract
// the configuration parameters for the PKCS#11 module.
//
// Parameters:
//   - pkcs11URI: A PKCS#11 URI string in the format "pkcs11:module=/path/to/module;pin=1234;..."
//   - keyLabel: Label used to identify the private key in the HSM
//   - certLabel: Label used to identify the certificate in the HSM
//
// Returns:
//   - A new PKCS11Signer configured based on the URI parameters
//   - An error if the URI cannot be parsed or is invalid
func NewPKCS11SignerFromURI(pkcs11URI, keyLabel, certLabel string) (*PKCS11Signer, error) {
	config := ExtractPKCS11Config(pkcs11URI)
	if config == nil {
		return nil, fmt.Errorf("invalid PKCS#11 URI: %s", pkcs11URI)
	}
	return NewPKCS11Signer(config, keyLabel, certLabel), nil
}

// initialize ensures the PKCS#11 context is created.
// This method lazy-loads the PKCS#11 module and initializes the connection
// to the HSM on first use. It caches the context for subsequent operations.
//
// Returns:
//   - An error if the PKCS#11 context could not be configured
func (ps *PKCS11Signer) initialize() error {
	if ps.initialized {
		return nil
	}

	context, err := crypto11.Configure(ps.Config)
	if err != nil {
		return fmt.Errorf("failed to configure PKCS#11 context: %w", err)
	}

	ps.context = context
	ps.initialized = true
	return nil
}

// Close cleans up any resources associated with the signer.
// This method prepares the signer for garbage collection by resetting
// its internal state. Note that the crypto11 context doesn't currently
// have an explicit Close method, but this function is provided for
// future-proofing.
//
// Returns:
//   - Always returns nil error (reserved for future implementations)
func (ps *PKCS11Signer) Close() error {
	if ps.context != nil {
		// Context doesn't have a Close method in crypto11,
		// but we can add it here for future-proofing
		ps.initialized = false
		ps.context = nil
	}
	return nil
}

// SetKeyID sets the ID to use for key and certificate lookups.
// The key ID is typically a hex string (with or without '0x' prefix)
// that identifies both the private key and certificate in the HSM.
//
// Parameter:
//   - id: Hex string ID to identify the key and certificate in the HSM
func (ps *PKCS11Signer) SetKeyID(id string) {
	ps.keyID = id
}

// hexToBytes converts a hex string to bytes (handling both with and without '0x' prefix).
// This helper function normalizes hex strings for use as PKCS#11 object IDs.
//
// Parameters:
//   - hexStr: A hex string, with or without '0x' prefix
//
// Returns:
//   - The decoded byte array
//   - An error if the hex string cannot be decoded
func hexToBytes(hexStr string) ([]byte, error) {
	// Remove 0x prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")

	// Handle odd-length hex strings by prepending a 0
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}

	return hex.DecodeString(hexStr)
}

// Sign implements XMLSigner.Sign using PKCS#11 hardware token with goxmldsig's Signer interface.
// This method connects to the HSM, retrieves the private key and certificate,
// and uses them to create an XML digital signature.
//
// Parameters:
//   - xmlData: Raw XML bytes to sign
//
// Returns:
//   - The signed XML document as bytes
//   - An error if HSM connection, key/cert retrieval, or signing fails
func (ps *PKCS11Signer) Sign(xmlData []byte) ([]byte, error) {
	if err := ps.initialize(); err != nil {
		return nil, err
	}

	// Convert ID to bytes
	idBytes, err := hexToBytes(ps.keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert key ID to bytes: %w", err)
	}

	// Get the private key by ID and label
	// The crypto11 FindKeyPair function takes (id, label) parameters
	privateKey, err := ps.context.FindKeyPair(idBytes, []byte(ps.keyLabel))
	if err != nil {
		return nil, fmt.Errorf("failed to find private key with label '%s' and ID '%s': %w",
			ps.keyLabel, ps.keyID, err)
	}

	// Get the certificate by ID and label
	// The crypto11 FindCertificate function takes (id, label, serial) parameters
	cert, err := ps.context.FindCertificate(idBytes, []byte(ps.certLabel), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to find certificate with label '%s' and ID '%s': %w",
			ps.certLabel, ps.keyID, err)
	}

	// Create a goxmldsig PKCS11Signer that implements the Signer interface
	// Using SHA256 as the default hash algorithm
	pkcs11Signer, err := xmldsig.NewPKCS11Signer(privateKey, cert.Raw, crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("failed to create PKCS11Signer: %w", err)
	}

	return SignXML(xmlData, pkcs11Signer)
}

// ExtractPKCS11Config extracts a PKCS#11 configuration from a URI.
// This function parses a PKCS#11 URI according to RFC 7512 and extracts
// the configuration parameters for initializing a PKCS#11 module connection.
//
// Parameters:
//   - pkcs11URI: A PKCS#11 URI string in the format "pkcs11:module=/path/to/module;pin=1234;..."
//
// Returns:
//   - A crypto11.Config populated with parameters from the URI, or nil if parsing fails
func ExtractPKCS11Config(pkcs11URI string) *crypto11.Config {
	// Parse the PKCS#11 URI
	u, err := url.Parse(pkcs11URI)
	if err != nil || u.Scheme != "pkcs11" {
		return nil
	}

	// Parse according to RFC 7512 PKCS#11 URI scheme
	// Format is pkcs11:module=/path/to/module;pin=1234;...

	if u.Opaque == "" {
		return nil
	}

	// Split parameters (separated by semicolons)
	params := strings.Split(u.Opaque, ";")

	config := &crypto11.Config{}

	// Parse each parameter
	for _, param := range params {
		if param == "" {
			continue
		}

		// Split key-value pair
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := kv[0]
		value := kv[1]

		// Set config based on key
		switch key {
		case "module":
			config.Path = value
		case "pin":
			config.Pin = value
		case "token":
			config.TokenLabel = value
		case "slot-id":
			slotID, err := strconv.Atoi(value)
			if err == nil {
				config.SlotNumber = &slotID
			}
		}
	}

	// Module path is required
	if config.Path == "" {
		return nil
	}

	return config
}
