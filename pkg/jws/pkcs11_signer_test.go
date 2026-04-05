package jws

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/ThalesGroup/crypto11"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHexToBytes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{name: "simple", input: "48656c6c6f", want: []byte("Hello")},
		{name: "with 0x prefix", input: "0x48656c6c6f", want: []byte("Hello")},
		{name: "uppercase", input: "48454C4C4F", want: []byte("HELLO")},
		{name: "odd length", input: "123", want: []byte{0x01, 0x23}},
		{name: "empty", input: "", want: []byte{}},
		{name: "invalid char", input: "xyz", wantErr: true},
		{name: "mixed case invalid", input: "48eLLo", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hexToBytes(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUnhex(t *testing.T) {
	tests := []struct {
		input byte
		want  int
	}{
		{'0', 0}, {'1', 1}, {'9', 9},
		{'a', 10}, {'b', 11}, {'f', 15},
		{'A', 10}, {'B', 11}, {'F', 15},
		{'g', -1}, {'z', -1}, {'!', -1},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := unhex(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAlgorithmForPublicKey(t *testing.T) {
	t.Run("RSA", func(t *testing.T) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		alg, err := algorithmForPublicKey(&key.PublicKey)
		require.NoError(t, err)
		assert.Equal(t, "RS256", string(alg))
	})

	t.Run("EC P-256", func(t *testing.T) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		alg, err := algorithmForPublicKey(&key.PublicKey)
		require.NoError(t, err)
		assert.Equal(t, "ES256", string(alg))
	})

	t.Run("EC P-384", func(t *testing.T) {
		key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		require.NoError(t, err)
		alg, err := algorithmForPublicKey(&key.PublicKey)
		require.NoError(t, err)
		assert.Equal(t, "ES384", string(alg))
	})

	t.Run("EC P-521", func(t *testing.T) {
		key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		require.NoError(t, err)
		alg, err := algorithmForPublicKey(&key.PublicKey)
		require.NoError(t, err)
		assert.Equal(t, "ES512", string(alg))
	})

	t.Run("EC unsupported curve", func(t *testing.T) {
		key, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
		require.NoError(t, err)
		_, err = algorithmForPublicKey(&key.PublicKey)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported EC curve")
	})

	t.Run("unsupported key type", func(t *testing.T) {
		_, err := algorithmForPublicKey("not a key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported public key type")
	})
}

func TestNewPKCS11Signer_Constructor(t *testing.T) {
	// Test that constructor creates signer struct correctly
	// (doesn't actually connect to HSM until Sign is called)
	config := &crypto11.Config{
		Path: "/usr/lib/softhsm/libsofthsm2.so",
		Pin:  "1234",
	}
	signer := NewPKCS11Signer(config, "mykey", "mycert")
	assert.NotNil(t, signer)
	assert.Equal(t, "mykey", signer.keyLabel)
	assert.Equal(t, "mycert", signer.certLabel)
}

func TestPKCS11Signer_SetKeyID(t *testing.T) {
	config := &crypto11.Config{
		Path: "/usr/lib/softhsm/libsofthsm2.so",
	}
	signer := NewPKCS11Signer(config, "key", "cert")
	signer.SetKeyID("0102030405")
	assert.Equal(t, "0102030405", signer.keyID)
}
