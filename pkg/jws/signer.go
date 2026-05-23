// Package jws provides JSON Web Signature signing and verification for LoTE documents.
//
// This complements the XML-DSIG signing in pkg/dsig by providing JWS-based
// integrity protection as required by ETSI TS 119 602.
//
// By default, signatures are produced in JAdES-B-B profile (ETSI TS 119 182-1),
// which adds iat (signing time) and x5t#S256 (certificate thumbprint) headers
// to the standard JWS protected header.
package jws

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	jose "github.com/go-jose/go-jose/v4"
)

// JSONSigner signs JSON payloads and produces compact JWS strings.
type JSONSigner interface {
	Sign(payload []byte) (string, error)
}

// JAdESConfigurable allows configuring JAdES compliance on a signer.
type JAdESConfigurable interface {
	SetJAdES(enabled bool)
}

// JSONVerifier verifies JWS compact serializations.
type JSONVerifier interface {
	Verify(compact string) ([]byte, error)
}

// FileSigner signs using a PEM-encoded private key and certificate file.
// By default it produces JAdES-B-B compliant signatures.
type FileSigner struct {
	key   crypto.PrivateKey
	certs []*x509.Certificate
	alg   jose.SignatureAlgorithm
	jades bool // JAdES-B-B compliance (default: true)
}

// NewFileSigner creates a JWS signer from PEM certificate and private key files.
// JAdES-B-B compliance is enabled by default.
func NewFileSigner(certFile, keyFile string) (*FileSigner, error) {
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %s: %w", keyFile, err)
	}

	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert file %s: %w", certFile, err)
	}

	key, err := parsePrivateKey(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	certs, err := parseCertificates(certPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificates: %w", err)
	}

	alg, err := algorithmForKey(key)
	if err != nil {
		return nil, err
	}

	return &FileSigner{
		key:   key,
		certs: certs,
		alg:   alg,
		jades: true,
	}, nil
}

// SetJAdES enables or disables JAdES-B-B compliant headers (iat, x5t#S256).
func (s *FileSigner) SetJAdES(enabled bool) {
	s.jades = enabled
}

// Sign produces a compact JWS serialization of the payload.
func (s *FileSigner) Sign(payload []byte) (string, error) {
	signingKey := jose.SigningKey{Algorithm: s.alg, Key: s.key}
	opts := &jose.SignerOptions{}
	opts.WithHeader("x5c", certsToX5C(s.certs))

	if s.jades {
		addJAdESHeaders(opts, s.certs[0])
	}

	signer, err := jose.NewSigner(signingKey, opts)
	if err != nil {
		return "", fmt.Errorf("failed to create JWS signer: %w", err)
	}

	obj, err := signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("JWS signing failed: %w", err)
	}
	return obj.CompactSerialize()
}

// addJAdESHeaders adds ETSI TS 119 182-1 JAdES-B-B required headers to the signer options.
func addJAdESHeaders(opts *jose.SignerOptions, cert *x509.Certificate) {
	// iat: signing time as NumericDate (Unix time) per RFC 7519 / ETSI TS 119 182-1
	opts.WithHeader("iat", time.Now().UTC().Unix())

	// x5t#S256: SHA-256 thumbprint of the signing certificate (ETSI TS 119 182-1 §5.2.2)
	thumbprint := sha256.Sum256(cert.Raw)
	opts.WithHeader("x5t#S256", base64.RawURLEncoding.EncodeToString(thumbprint[:]))
}

// KeyVerifier verifies JWS using a set of trusted public keys.
type KeyVerifier struct {
	keys []crypto.PublicKey
}

// NewKeyVerifier creates a verifier from trusted public keys.
func NewKeyVerifier(keys ...crypto.PublicKey) *KeyVerifier {
	return &KeyVerifier{keys: keys}
}

// NewCertVerifier creates a verifier from trusted X.509 certificates.
func NewCertVerifier(certs ...*x509.Certificate) *KeyVerifier {
	keys := make([]crypto.PublicKey, len(certs))
	for i, c := range certs {
		keys[i] = c.PublicKey
	}
	return &KeyVerifier{keys: keys}
}

// NewCertFileVerifier creates a verifier from a PEM certificate file.
func NewCertFileVerifier(certFile string) (*KeyVerifier, error) {
	pemData, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert file %s: %w", certFile, err)
	}
	certs, err := parseCertificates(pemData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificates: %w", err)
	}
	return NewCertVerifier(certs...), nil
}

// Verify verifies a compact JWS and returns the payload.
func (v *KeyVerifier) Verify(compact string) ([]byte, error) {
	obj, err := jose.ParseSigned(compact, []jose.SignatureAlgorithm{
		jose.RS256, jose.RS384, jose.RS512,
		jose.ES256, jose.ES384, jose.ES512,
		jose.PS256, jose.PS384, jose.PS512,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWS: %w", err)
	}

	for _, key := range v.keys {
		payload, err := obj.Verify(key)
		if err == nil {
			return payload, nil
		}
	}
	return nil, fmt.Errorf("JWS signature verification failed: no trusted key matched")
}

// --- helpers ---

func parsePrivateKey(pemData []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		return x509.ParsePKCS8PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
}

func parseCertificates(pemData []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	rest := pemData
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in PEM data")
	}
	return certs, nil
}

func algorithmForKey(key crypto.PrivateKey) (jose.SignatureAlgorithm, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return jose.RS256, nil
	case *ecdsa.PrivateKey:
		switch k.Curve {
		case elliptic.P256():
			return jose.ES256, nil
		case elliptic.P384():
			return jose.ES384, nil
		case elliptic.P521():
			return jose.ES512, nil
		default:
			return "", fmt.Errorf("unsupported EC curve: %v", k.Curve.Params().Name)
		}
	default:
		return "", fmt.Errorf("unsupported key type: %T", key)
	}
}

func certsToX5C(certs []*x509.Certificate) [][]byte {
	x5c := make([][]byte, len(certs))
	for i, c := range certs {
		x5c[i] = c.Raw
	}
	return x5c
}
