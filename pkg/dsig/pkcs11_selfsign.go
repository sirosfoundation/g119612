package dsig

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/ThalesGroup/crypto11"
)

// SelfSignedCertOptions configures SelfSignCertificate.
type SelfSignedCertOptions struct {
	// KeyLabel and KeyID identify the existing key pair in the token. The
	// certificate is issued for whatever public key that pair holds.
	KeyLabel string
	KeyID    string

	// Subject is the distinguished name for both subject and issuer.
	Subject pkix.Name

	// Validity is how long the certificate is valid from now.
	Validity time.Duration

	// CertLabel, when non-empty, is the label to store the certificate under
	// in the token. Any existing certificate with that label and the same ID
	// is removed first, so the token never ends up with a stale one alongside.
	CertLabel string
}

// SelfSignCertificate issues a self-signed CA certificate for a key pair that
// already exists in a PKCS#11 token, using that token to produce the signature.
// The private key never leaves the token.
//
// This exists because the obvious shell recipe is wrong in a way that is hard
// to notice. Generating a key in a token with pkcs11-tool and then minting a
// certificate with "openssl req -x509 -newkey ..." produces a certificate over
// a brand new, unrelated key pair; loading it into the token leaves the
// certificate and the signing key referring to different keys. Signatures then
// verify against nothing, and the failure only shows up in a relying party,
// never at signing time.
//
// Returns the certificate and its PEM encoding.
func SelfSignCertificate(config *crypto11.Config, opts SelfSignedCertOptions) (*x509.Certificate, []byte, error) {
	if config == nil {
		return nil, nil, fmt.Errorf("nil PKCS#11 config")
	}
	if opts.KeyLabel == "" {
		return nil, nil, fmt.Errorf("key label is required")
	}
	if opts.Validity <= 0 {
		return nil, nil, fmt.Errorf("validity must be positive, got %s", opts.Validity)
	}

	idBytes, err := hexToBytes(opts.KeyID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert key ID to bytes: %w", err)
	}

	ctx, err := crypto11.Configure(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to configure PKCS#11 context: %w", err)
	}

	signer, err := ctx.FindKeyPair(idBytes, []byte(opts.KeyLabel))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find key pair with label %q and ID %q: %w",
			opts.KeyLabel, opts.KeyID, err)
	}
	if signer == nil {
		return nil, nil, fmt.Errorf("no key pair with label %q and ID %q in token",
			opts.KeyLabel, opts.KeyID)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now().UTC()
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               opts.Subject,
		Issuer:                opts.Subject,
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(opts.Validity),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	// The public key comes from the token's key pair, and the token signs.
	// That is what ties the certificate to the key that will actually sign.
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, signer.Public(), signer)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse generated certificate: %w", err)
	}

	// Refuse to hand back a certificate that does not match its own key. This
	// is the invariant the whole function exists to guarantee.
	if err := cert.CheckSignatureFrom(cert); err != nil {
		return nil, nil, fmt.Errorf("generated certificate does not verify against its own key: %w", err)
	}

	if opts.CertLabel != "" {
		// Drop any prior certificate under this label/ID so a stale one cannot
		// be picked up instead. Absence is fine.
		if err := ctx.DeleteCertificate(idBytes, []byte(opts.CertLabel), nil); err != nil {
			return nil, nil, fmt.Errorf("failed to remove existing certificate %q: %w", opts.CertLabel, err)
		}
		if err := ctx.ImportCertificateWithLabel(idBytes, []byte(opts.CertLabel), cert); err != nil {
			return nil, nil, fmt.Errorf("failed to import certificate %q into token: %w", opts.CertLabel, err)
		}
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return cert, pemBytes, nil
}
