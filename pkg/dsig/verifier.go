// Package dsig provides XML Digital Signature (XML-DSIG) functionality
// for signing and verifying Trust Status Lists (TSLs) and other XML documents.
package dsig

import (
	"crypto/x509"
	"fmt"

	"github.com/beevik/etree"
	xmldsig "github.com/russellhaering/goxmldsig"
)

// XMLVerifier defines the interface for XML signature verification operations.
type XMLVerifier interface {
	// Verify validates the XML signature and returns the verified element.
	// Returns an error if verification fails.
	Verify(xmlData []byte) (*etree.Element, error)
}

// CertPoolStore implements xmldsig.X509CertificateStore using a [crypto/x509.CertPool].
// This allows using a standard Go certificate pool for signature verification.
type CertPoolStore struct {
	pool  *x509.CertPool
	certs []*x509.Certificate
}

// NewCertPoolStore creates a new CertPoolStore from a certificate pool.
// The pool should contain the trusted root certificates for signature verification.
func NewCertPoolStore(pool *x509.CertPool, certs []*x509.Certificate) *CertPoolStore {
	return &CertPoolStore{
		pool:  pool,
		certs: certs,
	}
}

// Certificates returns the list of certificates in the store.
// Required by xmldsig.X509CertificateStore interface.
func (s *CertPoolStore) Certificates() ([]*x509.Certificate, error) {
	return s.certs, nil
}

// VerifyXMLSignature verifies an XML digital signature against a certificate pool.
// It parses the XML, finds the enveloped signature, and validates it against
// the certificates in the provided pool.
//
// Parameters:
//   - xmlData: The signed XML document as bytes
//   - trustedCerts: The certificates to validate the signature against
//
// Returns:
//   - The verified XML element (with signature removed)
//   - An error if verification fails
func VerifyXMLSignature(xmlData []byte, trustedCerts []*x509.Certificate) (*etree.Element, error) {
	if len(trustedCerts) == 0 {
		return nil, fmt.Errorf("no trusted certificates provided for signature verification")
	}

	// Parse the XML document
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlData); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	root := doc.Root()
	if root == nil {
		return nil, fmt.Errorf("XML document has no root element")
	}

	// Create certificate pool from trusted certs
	pool := x509.NewCertPool()
	for _, cert := range trustedCerts {
		pool.AddCert(cert)
	}

	// Create validation context with the certificate store
	store := NewCertPoolStore(pool, trustedCerts)
	ctx := xmldsig.NewDefaultValidationContext(store)

	// Validate the signature
	verified, err := ctx.Validate(root)
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	return verified, nil
}

// VerifyXMLSignatureWithPool verifies an XML digital signature using certificates
// from a CertPool. This is a convenience function for cases where you have a
// CertPool but need to extract certificates for verification.
//
// Note: This function requires the actual certificate slice since CertPool
// doesn't expose its certificates directly.
//
// Parameters:
//   - xmlData: The signed XML document as bytes
//   - pool: The certificate pool (for future compatibility)
//   - certs: The certificates to validate against
//
// Returns:
//   - The verified XML element
//   - An error if verification fails
func VerifyXMLSignatureWithPool(xmlData []byte, pool *x509.CertPool, certs []*x509.Certificate) (*etree.Element, error) {
	return VerifyXMLSignature(xmlData, certs)
}

// TSLSignatureVerifier verifies XML signatures on ETSI Trust Status Lists.
// It implements the XMLVerifier interface.
type TSLSignatureVerifier struct {
	trustedCerts []*x509.Certificate
}

// NewTSLSignatureVerifier creates a verifier for TSL signatures.
// The trustedCerts should contain the certificates that are authorized
// to sign Trust Status Lists (typically EU or national scheme operators).
func NewTSLSignatureVerifier(trustedCerts []*x509.Certificate) *TSLSignatureVerifier {
	return &TSLSignatureVerifier{
		trustedCerts: trustedCerts,
	}
}

// Verify validates the XML signature on a TSL document.
// Returns the verified element (signature removed) or an error.
func (v *TSLSignatureVerifier) Verify(xmlData []byte) (*etree.Element, error) {
	return VerifyXMLSignature(xmlData, v.trustedCerts)
}

// AddTrustedCertificate adds a certificate to the list of trusted signers.
func (v *TSLSignatureVerifier) AddTrustedCertificate(cert *x509.Certificate) {
	v.trustedCerts = append(v.trustedCerts, cert)
}

// TrustedCertificates returns the list of trusted signer certificates.
func (v *TSLSignatureVerifier) TrustedCertificates() []*x509.Certificate {
	return v.trustedCerts
}
