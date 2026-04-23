package x509util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestCertDER(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	return der
}

func TestParseX5CFromArray(t *testing.T) {
	der := generateTestCertDER(t)
	b64 := base64.StdEncoding.EncodeToString(der)

	certs, err := ParseX5CFromArray([]interface{}{b64})
	require.NoError(t, err)
	require.Len(t, certs, 1)
	assert.Equal(t, "test", certs[0].Subject.CommonName)
}

func TestParseX5CFromArray_Empty(t *testing.T) {
	_, err := ParseX5CFromArray([]interface{}{})
	assert.Error(t, err)
}

func TestParseX5CFromArray_NotString(t *testing.T) {
	_, err := ParseX5CFromArray([]interface{}{42})
	assert.Error(t, err)
}

func TestParseX5CFromArray_BadBase64(t *testing.T) {
	_, err := ParseX5CFromArray([]interface{}{"not-valid-base64!!!"})
	assert.Error(t, err)
}

func TestParseX5CFromJWK(t *testing.T) {
	der := generateTestCertDER(t)
	b64 := base64.StdEncoding.EncodeToString(der)

	jwk := map[string]interface{}{
		"kty": "EC",
		"x5c": []interface{}{b64},
	}
	certs, err := ParseX5CFromJWK([]interface{}{jwk})
	require.NoError(t, err)
	require.Len(t, certs, 1)
	assert.Equal(t, "test", certs[0].Subject.CommonName)
}

func TestParseX5CFromJWK_Empty(t *testing.T) {
	_, err := ParseX5CFromJWK([]interface{}{})
	assert.Error(t, err)
}

func TestParseX5CFromJWK_NotMap(t *testing.T) {
	_, err := ParseX5CFromJWK([]interface{}{"not-a-map"})
	assert.Error(t, err)
}

func TestParseX5CFromJWK_NoX5C(t *testing.T) {
	jwk := map[string]interface{}{"kty": "EC"}
	_, err := ParseX5CFromJWK([]interface{}{jwk})
	assert.Error(t, err)
}

func TestParseX5CFromJWK_X5CNotArray(t *testing.T) {
	jwk := map[string]interface{}{"kty": "EC", "x5c": "not-array"}
	_, err := ParseX5CFromJWK([]interface{}{jwk})
	assert.Error(t, err)
}
