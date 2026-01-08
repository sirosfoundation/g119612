package dsig

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	xmldsig "github.com/russellhaering/goxmldsig"
)

// FileSigner implements XMLSigner using certificate and private key files.
// It uses file-based certificates and keys for signing XML documents.
// The certificate and key files should be in PEM format.
type FileSigner struct {
	// CertFile is the path to the X.509 certificate file in PEM format
	CertFile string

	// KeyFile is the path to the private key file in PEM format (PKCS#1 or PKCS#8)
	KeyFile string
}

// NewFileSigner creates a new FileSigner from certificate and key file paths.
// This is a convenience constructor for the FileSigner struct.
//
// Parameters:
//   - certFile: Path to the X.509 certificate file in PEM format
//   - keyFile: Path to the private key file in PEM format (PKCS#1 or PKCS#8)
//
// Returns:
//   - A new FileSigner instance configured with the provided files
func NewFileSigner(certFile, keyFile string) *FileSigner {
	return &FileSigner{
		CertFile: certFile,
		KeyFile:  keyFile,
	}
}

// Sign implements XMLSigner.Sign using certificate and key files.
// This method loads the certificate and private key from files,
// creates an XML digital signature, and returns the signed XML document.
//
// The method supports both PKCS#1 and PKCS#8 formatted private keys.
//
// Parameters:
//   - xmlData: Raw XML bytes to sign
//
// Returns:
//   - The signed XML document as bytes
//   - An error if reading files, parsing certificates/keys, or signing fails
func (fs *FileSigner) Sign(xmlData []byte) ([]byte, error) {
	// Load the certificate and private key
	certData, err := os.ReadFile(fs.CertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyData, err := os.ReadFile(fs.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Parse the certificate
	certBlock, _ := pem.Decode(certData)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Parse the private key
	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode key PEM")
	}

	// Try to parse the key in different formats
	var privateKey *rsa.PrivateKey
	var privateKeyAny interface{}

	// Try PKCS1 format
	privateKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try PKCS8 format
		privateKeyAny, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		var ok bool
		privateKey, ok = privateKeyAny.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
	}

	// Create a key store from the loaded certificate and private key
	keyStore := &fileKeyStore{
		cert: cert,
		key:  privateKey,
	}

	return SignXMLWithKeyStore(xmlData, keyStore)
}

// fileKeyStore implements the xmldsig.X509KeyStore interface.
// It provides access to an in-memory certificate and private key
// for XML digital signature operations.
type fileKeyStore struct {
	cert *x509.Certificate // The parsed X.509 certificate
	key  *rsa.PrivateKey   // The parsed RSA private key
}

// GetKeyPair returns the private key and certificate for signing.
// This method implements the xmldsig.X509KeyStore interface.
//
// Returns:
//   - The RSA private key for signing
//   - The raw X.509 certificate bytes for inclusion in the signature
//   - Always nil error as the keys are pre-loaded
func (ks *fileKeyStore) GetKeyPair() (*rsa.PrivateKey, []byte, error) {
	return ks.key, ks.cert.Raw, nil
}

// ToXMLDSigSigner converts a FileSigner to an xmldsig.Signer implementation.
// This method loads the certificate and private key from files and creates
// an xmldsig.Signer that can be used with the goxmldsig library directly.
//
// The method supports both PKCS#1 and PKCS#8 formatted private keys and
// configures the signer to use SHA-256 for signatures.
//
// Returns:
//   - An xmldsig.Signer implementation using the file-based certificate and key
//   - An error if reading files, parsing certificates/keys fails
func (fs *FileSigner) ToXMLDSigSigner() (xmldsig.Signer, error) {
	// Load the certificate and private key
	certData, err := os.ReadFile(fs.CertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyData, err := os.ReadFile(fs.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Parse the certificate
	certBlock, _ := pem.Decode(certData)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Parse the private key
	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode key PEM")
	}

	// Try to parse the key in different formats
	var privateKey *rsa.PrivateKey
	var privateKeyAny interface{}

	// Try PKCS1 format
	privateKey, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		// Try PKCS8 format
		privateKeyAny, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		var ok bool
		privateKey, ok = privateKeyAny.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
	}

	// Use the file private key with certificate to create a new xmldsig.Signer
	// Default to SHA256 for the signing algorithm
	return xmldsig.NewFileSigner(privateKey, cert.Raw, crypto.SHA256)
}
