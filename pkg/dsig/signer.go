// Package dsig provides XML Digital Signature (XML-DSIG) functionality
// for signing Trust Status Lists (TSLs) and other XML documents.
// It supports multiple signing mechanisms including file-based keys
// and PKCS#11 hardware security modules.
package dsig

import (
	"crypto/rsa"

	"github.com/beevik/etree"
	xmldsig "github.com/russellhaering/goxmldsig"
)

// XMLSigner defines the interface for XML document signing operations.
// Implementations include FileSigner and PKCS11Signer for different
// key storage mechanisms.
type XMLSigner interface {
	// Sign takes XML data as bytes and returns the signed XML data.
	// The signature is added according to the XML-DSIG standard.
	//
	// Parameters:
	//   - xmlData: The raw XML data to sign
	//
	// Returns:
	//   - The signed XML data
	//   - An error if signing fails
	Sign(xmlData []byte) ([]byte, error)
}

// X509KeyStore defines an interface for accessing X.509 certificates and private keys.
// It's a wrapper around the goxmldsig X509KeyStore interface, providing access to
// key pairs needed for XML digital signatures.
type X509KeyStore interface {
	// GetKeyPair retrieves the private key and certificate for signing.
	//
	// Returns:
	//   - The RSA private key for signing
	//   - The X.509 certificate bytes to include in the signature
	//   - An error if the key pair cannot be retrieved
	GetKeyPair() (*rsa.PrivateKey, []byte, error)
}

// SignXML signs XML data using any implementation of the xmldsig.Signer interface.
// It applies XML Digital Signature standards to create a signed XML document.
//
// The function:
// 1. Sets up a signing context with exclusive canonicalization
// 2. Parses the input XML
// 3. Signs the document with an enveloped signature
// 4. Returns the signed document
//
// Parameters:
//   - xmlData: Raw XML bytes to sign
//   - signer: An implementation of xmldsig.Signer to perform the signing operation
//
// Returns:
//   - The signed XML document as bytes
//   - An error if parsing or signing fails
func SignXML(xmlData []byte, signer xmldsig.Signer) ([]byte, error) {
	// Create the signing context with our signer
	ctx := xmldsig.NewDefaultSigningContextWithSigner(signer)

	// Use exclusive canonicalization (C14N)
	ctx.Canonicalizer = xmldsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")

	// Parse the XML document
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlData); err != nil {
		return nil, err
	}

	// Sign the XML document
	signedDoc, err := ctx.SignEnveloped(doc.Root())
	if err != nil {
		return nil, err
	}

	// Return the signed XML
	doc2 := etree.NewDocument()
	doc2.SetRoot(signedDoc)
	return doc2.WriteToBytes()
}

// SignXMLWithKeyStore signs XML data using the provided X509KeyStore.
// This is a convenience function that creates a signing context and applies
// the same canonicalization and signing process as SignXML.
//
// Parameters:
//   - xmlData: Raw XML bytes to sign
//   - keyStore: An implementation of xmldsig.X509KeyStore that provides access to
//     the private key and certificate for signing
//
// Returns:
//   - The signed XML document as bytes
//   - An error if parsing or signing fails
func SignXMLWithKeyStore(xmlData []byte, keyStore xmldsig.X509KeyStore) ([]byte, error) {
	// Create the signature context
	ctx := xmldsig.NewDefaultSigningContext(keyStore)
	ctx.Canonicalizer = xmldsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")

	// Parse the XML document
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlData); err != nil {
		return nil, err
	}

	// Sign the XML document
	signedDoc, err := ctx.SignEnveloped(doc.Root())
	if err != nil {
		return nil, err
	}

	// Return the signed XML
	doc2 := etree.NewDocument()
	doc2.SetRoot(signedDoc)
	return doc2.WriteToBytes()
}

// GetSigningMethodName returns a string description of the default signing method.
// This function indicates which signature algorithm is used by the package
// for signing XML documents.
//
// Returns:
//   - A string identifying the algorithm, currently "rsa-sha256"
func GetSigningMethodName() string {
	return "rsa-sha256" // Default to SHA256
}
