package jws

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"math/big"

	"github.com/ThalesGroup/crypto11"
	jose "github.com/go-jose/go-jose/v4"
)

// PKCS11Signer signs JSON payloads using a PKCS#11 hardware token.
// By default it produces JAdES-B-B compliant signatures.
type PKCS11Signer struct {
	config    *crypto11.Config
	context   *crypto11.Context
	keyLabel  string
	certLabel string
	keyID     string
	jades     bool // JAdES-B-B compliance (default: true)
}

// NewPKCS11Signer creates a JWS signer backed by a PKCS#11 HSM.
// JAdES-B-B compliance is enabled by default.
func NewPKCS11Signer(config *crypto11.Config, keyLabel, certLabel string) *PKCS11Signer {
	return &PKCS11Signer{
		config:    config,
		keyLabel:  keyLabel,
		certLabel: certLabel,
		keyID:     "01",
		jades:     true,
	}
}

// SetKeyID sets the hex ID for key and certificate lookups.
func (s *PKCS11Signer) SetKeyID(id string) {
	s.keyID = id
}

// SetJAdES enables or disables JAdES-B-B compliant headers (sigT, x5t#S256).
func (s *PKCS11Signer) SetJAdES(enabled bool) {
	s.jades = enabled
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
	if privateKey == nil {
		return "", fmt.Errorf("no private key found with label %q and ID %q",
			s.keyLabel, s.keyID)
	}

	cert, err := s.context.FindCertificate(idBytes, []byte(s.certLabel), nil)
	if err != nil {
		return "", fmt.Errorf("failed to find certificate with label %q and ID %q: %w",
			s.certLabel, s.keyID, err)
	}
	if cert == nil {
		return "", fmt.Errorf("no certificate found with label %q and ID %q",
			s.certLabel, s.keyID)
	}

	alg, err := algorithmForPublicKey(cert.PublicKey)
	if err != nil {
		return "", err
	}

	// Wrap crypto11.Signer as jose.OpaqueSigner so go-jose can use it
	opaque := &pkcs11OpaqueSigner{
		signer: privateKey,
		cert:   cert,
		alg:    alg,
	}

	signingKey := jose.SigningKey{Algorithm: alg, Key: opaque}
	opts := &jose.SignerOptions{}
	opts.WithHeader("x5c", [][]byte{cert.Raw})

	if s.jades {
		addJAdESHeaders(opts, cert)
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

// pkcs11OpaqueSigner wraps a crypto11.Signer to implement jose.OpaqueSigner.
// go-jose does not accept crypto.Signer directly in its type switch;
// it requires either a concrete key type or an OpaqueSigner.
type pkcs11OpaqueSigner struct {
	signer crypto.Signer
	cert   *x509.Certificate
	alg    jose.SignatureAlgorithm
}

func (o *pkcs11OpaqueSigner) Public() *jose.JSONWebKey {
	return &jose.JSONWebKey{Key: o.signer.Public()}
}

func (o *pkcs11OpaqueSigner) Algs() []jose.SignatureAlgorithm {
	return []jose.SignatureAlgorithm{o.alg}
}

func (o *pkcs11OpaqueSigner) SignPayload(payload []byte, alg jose.SignatureAlgorithm) ([]byte, error) {
	// Determine hash from algorithm
	var hash crypto.Hash
	switch alg {
	case jose.ES256:
		hash = crypto.SHA256
	case jose.ES384:
		hash = crypto.SHA384
	case jose.ES512:
		hash = crypto.SHA512
	case jose.RS256, jose.PS256:
		hash = crypto.SHA256
	case jose.RS384, jose.PS384:
		hash = crypto.SHA384
	case jose.RS512, jose.PS512:
		hash = crypto.SHA512
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", alg)
	}

	hasher := hash.New()
	_, _ = hasher.Write(payload)
	digest := hasher.Sum(nil)

	var opts crypto.SignerOpts = hash
	// RSA-PSS requires special options
	switch alg {
	case jose.PS256, jose.PS384, jose.PS512:
		opts = &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash, Hash: hash}
	}

	sig, err := o.signer.Sign(rand.Reader, digest, opts)
	if err != nil {
		return nil, fmt.Errorf("PKCS#11 signing failed: %w", err)
	}

	// For ECDSA, crypto.Signer returns ASN.1 DER-encoded signature,
	// but JWS requires raw R||S format
	switch alg {
	case jose.ES256, jose.ES384, jose.ES512:
		return convertECDSASignature(sig, alg)
	}

	return sig, nil
}

// convertECDSASignature converts ASN.1 DER ECDSA signature to JWS R||S format.
func convertECDSASignature(derSig []byte, alg jose.SignatureAlgorithm) ([]byte, error) {
	var keySize int
	switch alg {
	case jose.ES256:
		keySize = 32
	case jose.ES384:
		keySize = 48
	case jose.ES512:
		keySize = 66
	default:
		return nil, fmt.Errorf("unsupported EC algorithm: %s", alg)
	}

	// Parse ASN.1 signature
	r, s, err := parseECDSASignature(derSig)
	if err != nil {
		return nil, err
	}

	// Pad R and S to key size
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	out := make([]byte, 2*keySize)
	copy(out[keySize-len(rBytes):keySize], rBytes)
	copy(out[2*keySize-len(sBytes):], sBytes)

	return out, nil
}

// parseECDSASignature parses an ASN.1 DER-encoded ECDSA signature into R and S.
func parseECDSASignature(derSig []byte) (*big.Int, *big.Int, error) {
	var sig struct {
		R, S *big.Int
	}
	if _, err := asn1.Unmarshal(derSig, &sig); err != nil {
		return nil, nil, fmt.Errorf("failed to parse ECDSA signature: %w", err)
	}
	return sig.R, sig.S, nil
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
