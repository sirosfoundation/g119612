// Package dsig provides XML Digital Signature (XML-DSIG) functionality.
// This file implements XAdES-B-B (ETSI TS 101 903 / ETSI EN 319 132-1)
// qualified electronic signatures for Trust Status Lists.

package dsig

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/beevik/etree"
	xmldsig "github.com/russellhaering/goxmldsig"
)

const (
	xadesNamespace      = "http://uri.etsi.org/01903/v1.3.2#"
	dsigNamespace       = "http://www.w3.org/2000/09/xmldsig#"
	excC14NAlgorithm    = "http://www.w3.org/2001/10/xml-exc-c14n#"
	sha256DigestAlg     = "http://www.w3.org/2001/04/xmlenc#sha256"
	envelopedSigAlg     = "http://www.w3.org/2000/09/xmldsig#enveloped-signature"
	rsaSHA256SigAlg     = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	signedPropertiesType = "http://uri.etsi.org/01903#SignedProperties"
)

// SignXMLWithXAdES signs XML data with an enveloped XAdES-B-B signature.
// This creates a full XAdES-B-B signature including:
//   - ds:Signature with two ds:Reference elements (document + SignedProperties)
//   - ds:Object containing xades:QualifyingProperties
//   - xades:SignedProperties with SigningTime, SigningCertificate, and DataObjectFormat
//
// Parameters:
//   - xmlData: Raw XML bytes to sign
//   - signer: crypto.Signer for signing (e.g., *rsa.PrivateKey or PKCS#11 opaque signer)
//   - cert: X.509 certificate of the signer
func SignXMLWithXAdES(xmlData []byte, signer crypto.Signer, cert *x509.Certificate) ([]byte, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(xmlData); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	root := doc.Root()

	// Generate a unique signature ID
	sigID := generateSignatureID()
	signedPropsID := "xades-" + sigID

	// Build the QualifyingProperties / SignedProperties element
	object := buildXAdESObject(sigID, signedPropsID, cert)

	// Canonicalize the SignedProperties for the second reference digest
	signedPropsDigest, err := digestSignedProperties(object, signedPropsID)
	if err != nil {
		return nil, fmt.Errorf("failed to digest SignedProperties: %w", err)
	}

	// Compute digest of the document (with enveloped-signature transform)
	docDigest, err := digestDocument(root)
	if err != nil {
		return nil, fmt.Errorf("failed to digest document: %w", err)
	}

	// Build SignedInfo with two references
	signedInfo := buildSignedInfo(docDigest, signedPropsDigest, signedPropsID)

	// Canonicalize SignedInfo and sign it
	signedInfoDigest, err := canonicalizeAndHash(signedInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize SignedInfo: %w", err)
	}

	rawSignature, err := signer.Sign(rand.Reader, signedInfoDigest, crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	// Build the complete Signature element
	sig := buildSignatureElement(signedInfo, rawSignature, cert, object, sigID)

	// Append signature to a copy of the root
	result := root.Copy()
	result.AddChild(sig)

	outDoc := etree.NewDocument()
	outDoc.SetRoot(result)
	return outDoc.WriteToBytes()
}

// generateSignatureID generates a deterministic-format signature ID.
func generateSignatureID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// buildXAdESObject constructs the ds:Object element containing xades:QualifyingProperties.
func buildXAdESObject(sigID, signedPropsID string, cert *x509.Certificate) *etree.Element {
	object := etree.NewElement("ds:Object")

	qp := object.CreateElement("xades:QualifyingProperties")
	qp.CreateAttr("xmlns:xades", xadesNamespace)
	qp.CreateAttr("Target", "#"+sigID)

	sp := qp.CreateElement("xades:SignedProperties")
	sp.CreateAttr("Id", signedPropsID)

	// SignedSignatureProperties
	ssp := sp.CreateElement("xades:SignedSignatureProperties")

	// SigningTime
	signingTime := ssp.CreateElement("xades:SigningTime")
	signingTime.SetText(time.Now().UTC().Format("2006-01-02T15:04:05Z"))

	// SigningCertificate
	sigCert := ssp.CreateElement("xades:SigningCertificate")
	certElem := sigCert.CreateElement("xades:Cert")

	// CertDigest (SHA-256)
	certDigest := certElem.CreateElement("xades:CertDigest")
	digestMethod := certDigest.CreateElement("ds:DigestMethod")
	digestMethod.CreateAttr("Algorithm", sha256DigestAlg)
	hash := sha256.Sum256(cert.Raw)
	digestValue := certDigest.CreateElement("ds:DigestValue")
	digestValue.SetText(base64.StdEncoding.EncodeToString(hash[:]))

	// IssuerSerial
	issuerSerial := certElem.CreateElement("xades:IssuerSerial")
	issuerName := issuerSerial.CreateElement("ds:X509IssuerName")
	issuerName.SetText(cert.Issuer.String())
	serialNumber := issuerSerial.CreateElement("ds:X509SerialNumber")
	serialNumber.SetText(cert.SerialNumber.String())

	// SignedDataObjectProperties
	sdop := sp.CreateElement("xades:SignedDataObjectProperties")
	dof := sdop.CreateElement("xades:DataObjectFormat")
	dof.CreateAttr("ObjectReference", "#r-doc-"+sigID)
	mimeType := dof.CreateElement("xades:MimeType")
	mimeType.SetText("text/xml")

	return object
}

// buildSignedInfo constructs the ds:SignedInfo element with two references.
func buildSignedInfo(docDigest, signedPropsDigest []byte, signedPropsID string) *etree.Element {
	signedInfo := etree.NewElement("ds:SignedInfo")
	signedInfo.CreateAttr("xmlns:ds", dsigNamespace)

	// CanonicalizationMethod
	c14n := signedInfo.CreateElement("ds:CanonicalizationMethod")
	c14n.CreateAttr("Algorithm", excC14NAlgorithm)

	// SignatureMethod
	sigMethod := signedInfo.CreateElement("ds:SignatureMethod")
	sigMethod.CreateAttr("Algorithm", rsaSHA256SigAlg)

	// Reference 1: the document
	ref1 := signedInfo.CreateElement("ds:Reference")
	ref1.CreateAttr("URI", "")
	transforms := ref1.CreateElement("ds:Transforms")
	t1 := transforms.CreateElement("ds:Transform")
	t1.CreateAttr("Algorithm", envelopedSigAlg)
	t2 := transforms.CreateElement("ds:Transform")
	t2.CreateAttr("Algorithm", excC14NAlgorithm)
	dm1 := ref1.CreateElement("ds:DigestMethod")
	dm1.CreateAttr("Algorithm", sha256DigestAlg)
	dv1 := ref1.CreateElement("ds:DigestValue")
	dv1.SetText(base64.StdEncoding.EncodeToString(docDigest))

	// Reference 2: SignedProperties
	ref2 := signedInfo.CreateElement("ds:Reference")
	ref2.CreateAttr("Type", signedPropertiesType)
	ref2.CreateAttr("URI", "#"+signedPropsID)
	transforms2 := ref2.CreateElement("ds:Transforms")
	t3 := transforms2.CreateElement("ds:Transform")
	t3.CreateAttr("Algorithm", excC14NAlgorithm)
	dm2 := ref2.CreateElement("ds:DigestMethod")
	dm2.CreateAttr("Algorithm", sha256DigestAlg)
	dv2 := ref2.CreateElement("ds:DigestValue")
	dv2.SetText(base64.StdEncoding.EncodeToString(signedPropsDigest))

	return signedInfo
}

// buildSignatureElement assembles the complete ds:Signature element.
func buildSignatureElement(signedInfo *etree.Element, rawSignature []byte, cert *x509.Certificate, object *etree.Element, sigID string) *etree.Element {
	sig := etree.NewElement("ds:Signature")
	sig.CreateAttr("xmlns:ds", dsigNamespace)
	sig.CreateAttr("Id", sigID)

	// Remove the temporary xmlns:ds from SignedInfo (it's inherited from Signature)
	signedInfo.RemoveAttr("xmlns:ds")
	sig.AddChild(signedInfo)

	// SignatureValue
	sigValue := sig.CreateElement("ds:SignatureValue")
	sigValue.SetText(base64.StdEncoding.EncodeToString(rawSignature))

	// KeyInfo
	keyInfo := sig.CreateElement("ds:KeyInfo")
	x509Data := keyInfo.CreateElement("ds:X509Data")
	x509Cert := x509Data.CreateElement("ds:X509Certificate")
	x509Cert.SetText(base64.StdEncoding.EncodeToString(cert.Raw))

	// Object (containing QualifyingProperties)
	sig.AddChild(object)

	return sig
}

// digestDocument computes the SHA-256 digest of the document root after
// applying exclusive canonicalization (simulating the enveloped-signature transform).
func digestDocument(root *etree.Element) ([]byte, error) {
	canonicalizer := xmldsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")
	canonical, err := canonicalizer.Canonicalize(root)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(canonical)
	return hash[:], nil
}

// digestSignedProperties computes the SHA-256 digest of the SignedProperties element
// after exclusive canonicalization.
func digestSignedProperties(object *etree.Element, signedPropsID string) ([]byte, error) {
	// Find the SignedProperties element inside the Object
	qp := object.FindElement("xades:QualifyingProperties")
	if qp == nil {
		return nil, fmt.Errorf("QualifyingProperties not found")
	}
	sp := qp.FindElement("xades:SignedProperties")
	if sp == nil {
		return nil, fmt.Errorf("SignedProperties not found")
	}

	// For exclusive canonicalization, we need to create a detached copy
	// with the necessary namespace declarations visible on the element itself,
	// since exclusive C14N requires namespace declarations to be visibly utilized.
	spCopy := sp.Copy()
	spCopy.CreateAttr("xmlns:xades", xadesNamespace)
	spCopy.CreateAttr("xmlns:ds", dsigNamespace)

	canonicalizer := xmldsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")
	canonical, err := canonicalizer.Canonicalize(spCopy)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(canonical)
	return hash[:], nil
}

// canonicalizeAndHash canonicalizes an element with exclusive C14N and returns its SHA-256 hash.
func canonicalizeAndHash(el *etree.Element) ([]byte, error) {
	canonicalizer := xmldsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")
	canonical, err := canonicalizer.Canonicalize(el)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(canonical)
	return hash[:], nil
}
