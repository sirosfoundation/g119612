# Digital Signature Package

This package provides a comprehensive framework for handling XML Digital Signatures, especially for Trust Status Lists (TSLs). It supports both file-based keys and PKCS#11 hardware security modules.

## Interfaces

The package provides a simple, consistent interface for signing XML documents:

```go
// XMLSigner represents an interface for signing XML documents with XML-DSIG
type XMLSigner interface {
	// Sign takes XML data and returns signed XML data
	Sign(xmlData []byte) ([]byte, error)
}
```

## Available Signers

### FileSigner

`FileSigner` implements XML signing using certificate and private key files:

```go
// Create a new file signer
signer := dsig.NewFileSigner("path/to/cert.pem", "path/to/key.pem")

// Sign XML data
signedXML, err := signer.Sign(xmlData)
```

### PKCS11Signer

`PKCS11Signer` implements XML signing using a PKCS#11 hardware token:

```go
// Create a PKCS11 signer from URI
signer, err := dsig.NewPKCS11SignerFromURI(
    "pkcs11:module=/usr/lib/softhsm/libsofthsm2.so;pin=1234;slot-id=0", 
    "key-label", 
    "cert-label"
)
if err != nil {
    // Handle error
}
defer signer.Close()

// Set key ID (optional)
signer.SetKeyID("01")

// Sign XML data
signedXML, err := signer.Sign(xmlData)
```

## Testing Utilities

The package includes testing utilities in the `dsig/test` subpackage to assist with testing PKCS#11 functionality using SoftHSM:

```go
// In your test function
func TestPKCS11Signing(t *testing.T) {
    // Skip if SoftHSM is unavailable
    helper := test.SkipIfSoftHSMUnavailable(t)
    
    // Set up SoftHSM token
    err := helper.Setup()
    if err != nil {
        t.Skip("Could not set up SoftHSM token")
    }
    defer helper.Cleanup()
    
    // Generate and import test key pair
    err = helper.GenerateAndImportTestCert("test-key", "test-cert", "01")
    if err != nil {
        t.Skip("Could not import test certificate")
    }
    
    // Get PKCS11 URI for testing
    pkcs11URI := helper.GetPKCS11URI()
    
    // Create signer and run tests...
}
```