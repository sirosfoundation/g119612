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
