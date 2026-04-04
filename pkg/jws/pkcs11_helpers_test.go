package jws

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/ThalesGroup/crypto11"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlgorithmForPublicKey_RSA(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	alg, err := algorithmForPublicKey(&key.PublicKey)
	assert.NoError(t, err)
	assert.Equal(t, "RS256", string(alg))
}

func TestAlgorithmForPublicKey_P256(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	alg, err := algorithmForPublicKey(&key.PublicKey)
	assert.NoError(t, err)
	assert.Equal(t, "ES256", string(alg))
}

func TestAlgorithmForPublicKey_P384(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	alg, err := algorithmForPublicKey(&key.PublicKey)
	assert.NoError(t, err)
	assert.Equal(t, "ES384", string(alg))
}

func TestAlgorithmForPublicKey_P521(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	require.NoError(t, err)
	alg, err := algorithmForPublicKey(&key.PublicKey)
	assert.NoError(t, err)
	assert.Equal(t, "ES512", string(alg))
}

func TestAlgorithmForPublicKey_UnsupportedType(t *testing.T) {
	_, pub, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	_, err = algorithmForPublicKey(pub)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported public key type")
}

func TestHexToBytes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{"simple", "01", []byte{0x01}, false},
		{"multi-byte", "01ff", []byte{0x01, 0xff}, false},
		{"with 0x prefix", "0x0a", []byte{0x0a}, false},
		{"odd length padded", "f", []byte{0x0f}, false},
		{"uppercase", "ABCD", []byte{0xab, 0xcd}, false},
		{"mixed case", "aBcD", []byte{0xab, 0xcd}, false},
		{"invalid chars", "zz", nil, true},
		{"empty with prefix", "0x", []byte{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hexToBytes(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUnhex(t *testing.T) {
	assert.Equal(t, 0, unhex('0'))
	assert.Equal(t, 9, unhex('9'))
	assert.Equal(t, 10, unhex('a'))
	assert.Equal(t, 15, unhex('f'))
	assert.Equal(t, 10, unhex('A'))
	assert.Equal(t, 15, unhex('F'))
	assert.Equal(t, -1, unhex('g'))
	assert.Equal(t, -1, unhex(' '))
	assert.Equal(t, -1, unhex('/'))
}

func TestNewPKCS11Signer(t *testing.T) {
	signer := NewPKCS11Signer(nil, "my-key", "my-cert")
	assert.NotNil(t, signer)
	assert.Equal(t, "my-key", signer.keyLabel)
	assert.Equal(t, "my-cert", signer.certLabel)
	assert.Equal(t, "01", signer.keyID) // default key ID
}

func TestPKCS11Signer_SetKeyID(t *testing.T) {
	signer := NewPKCS11Signer(nil, "k", "c")
	signer.SetKeyID("ff")
	assert.Equal(t, "ff", signer.keyID)
}

func TestPKCS11Signer_Close(t *testing.T) {
	signer := NewPKCS11Signer(nil, "k", "c")
	err := signer.Close()
	assert.NoError(t, err)
	assert.Nil(t, signer.context)
}

func TestPKCS11Signer_Sign_EmptyConfig(t *testing.T) {
	// Use an empty (but non-nil) config to avoid a nil-pointer panic
	// inside crypto11.Configure. The module path is intentionally invalid.
	signer := NewPKCS11Signer(&crypto11.Config{Path: "/nonexistent/pkcs11.so"}, "k", "c")
	_, err := signer.Sign([]byte("test"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to configure PKCS#11 context")
}
