package jws

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"fmt"

	"github.com/ThalesGroup/crypto11"
	jose "github.com/go-jose/go-jose/v4"
)

// PKCS11Signer signs JSON payloads using a PKCS#11 hardware token.
type PKCS11Signer struct {
	config    *crypto11.Config
	context   *crypto11.Context
	keyLabel  string
	certLabel string
	keyID     string
}

// NewPKCS11Signer creates a JWS signer backed by a PKCS#11 HSM.
func NewPKCS11Signer(config *crypto11.Config, keyLabel, certLabel string) *PKCS11Signer {
	return &PKCS11Signer{
		config:    config,
		keyLabel:  keyLabel,
		certLabel: certLabel,
		keyID:     "01",
	}
}

// SetKeyID sets the hex ID for key and certificate lookups.
func (s *PKCS11Signer) SetKeyID(id string) {
	s.keyID = id
}

// Sign produces a compact JWS serialization of the payload using the HSM key.
func (s *PKCS11Signer) Sign(payload []byte) (string, error) {
	if err := s.initialize(); err != nil {
		return "", err
	}

	idBytes, err := hexToBytes(s.keyID)
	if err != nil {
		return "", fmt.Errorf("failed to convert key ID to bytes: %w", err)
	}

	privateKey, err := s.context.FindKeyPair(idBytes, []byte(s.keyLabel))
	if err != nil {
		return "", fmt.Errorf("failed to find private key with label %q and ID %q: %w",
			s.keyLabel, s.keyID, err)
	}

	cert, err := s.context.FindCertificate(idBytes, []byte(s.certLabel), nil)
	if err != nil {
		return "", fmt.Errorf("failed to find certificate with label %q and ID %q: %w",
			s.certLabel, s.keyID, err)
	}

	alg, err := algorithmForPublicKey(cert.PublicKey)
	if err != nil {
		return "", err
	}

	signingKey := jose.SigningKey{Algorithm: alg, Key: privateKey}
	opts := &jose.SignerOptions{}
	opts.WithHeader("x5c", [][]byte{cert.Raw})

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

// Close releases any PKCS#11 resources.
func (s *PKCS11Signer) Close() error {
	if s.context != nil {
		s.context = nil
	}
	return nil
}

func (s *PKCS11Signer) initialize() error {
	if s.context != nil {
		return nil
	}
	ctx, err := crypto11.Configure(s.config)
	if err != nil {
		return fmt.Errorf("failed to configure PKCS#11 context: %w", err)
	}
	s.context = ctx
	return nil
}

// algorithmForPublicKey determines the JWS signing algorithm from a public key.
func algorithmForPublicKey(pub crypto.PublicKey) (jose.SignatureAlgorithm, error) {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		return jose.RS256, nil
	case *ecdsa.PublicKey:
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
		return "", fmt.Errorf("unsupported public key type: %T", pub)
	}
}

// hexToBytes converts a hex string (with optional "0x" prefix) to bytes.
func hexToBytes(hexStr string) ([]byte, error) {
	if len(hexStr) >= 2 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	b := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		hi := unhex(hexStr[i])
		lo := unhex(hexStr[i+1])
		if hi < 0 || lo < 0 {
			return nil, fmt.Errorf("invalid hex string: %s", hexStr)
		}
		b[i/2] = byte(hi<<4 | lo)
	}
	return b, nil
}

func unhex(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c - 'a' + 10)
	case 'A' <= c && c <= 'F':
		return int(c - 'A' + 10)
	default:
		return -1
	}
}
