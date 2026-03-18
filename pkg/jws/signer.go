// Package jws provides JSON Web Signature signing and verification for LoTE documents.
//
// This complements the XML-DSIG signing in pkg/dsig by providing JWS-based
// integrity protection as required by ETSI TS 119 602.
package jws

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	jose "github.com/go-jose/go-jose/v4"
)

// JSONSigner signs JSON payloads and produces compact JWS strings.
type JSONSigner interface {
	Sign(payload []byte) (string, error)
}

// JSONVerifier verifies JWS compact serializations.
type JSONVerifier interface {
	Verify(compact string) ([]byte, error)
}

// FileSigner signs using a PEM-encoded private key and certificate file.
type FileSigner struct {
	signer jose.Signer
}

// NewFileSigner creates a JWS signer from PEM certificate and private key files.
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

	signingKey := jose.SigningKey{Algorithm: alg, Key: key}
	opts := &jose.SignerOptions{}
	opts.WithHeader("x5c", certsToX5C(certs))

	signer, err := jose.NewSigner(signingKey, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWS signer: %w", err)
	}

	return &FileSigner{signer: signer}, nil
}

// Sign produces a compact JWS serialization of the payload.
func (s *FileSigner) Sign(payload []byte) (string, error) {
	obj, err := s.signer.Sign(payload)
	if err != nil {
		return "", fmt.Errorf("JWS signing failed: %w", err)
	}
	return obj.CompactSerialize()
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
