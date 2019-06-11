package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	chunkSize = 64 * 1024
)

var (
	defaultKeyStore = KeyStore{
		KDFSalt:  []byte{49, 226, 108, 3, 55, 36, 46, 86, 53, 219, 249, 67, 143, 113, 201, 210},
		KeyNonce: []byte{152, 162, 64, 109, 125, 231, 165, 166, 42, 144, 91, 97, 62, 118, 193, 196, 222, 101, 250, 60, 206, 64, 157, 55},
		Key:      []byte{150, 152, 73, 143, 185, 32, 106, 178, 67, 206, 106, 61, 13, 86, 146, 66, 87, 11, 117, 89, 225, 225, 136, 225, 111, 227, 158, 70, 255, 122, 72, 209, 15, 152, 67, 74, 194, 242, 175, 71, 28, 5, 32, 211, 140, 166, 29, 206},
	}
	defaultCrypto, _ = ExistingCrypto([]byte("foobarbaz"), &defaultKeyStore)
)

func TestNewCrypto(t *testing.T) {
	_, keyStore, err := NewCrypto([]byte("foobarbaz"))
	assert.NoError(t, err)
	assert.NotNil(t, keyStore.KDFSalt)
	assert.NotNil(t, keyStore.KeyNonce)
	assert.NotNil(t, keyStore.Key)
}

func TestExistingCryptoOK(t *testing.T) {
	_, err := ExistingCrypto([]byte("foobarbaz"), &defaultKeyStore)
	assert.NoError(t, err)
}

func TestExistingCryptoWrongPW(t *testing.T) {
	_, err := ExistingCrypto([]byte("lalala"), &defaultKeyStore)
	assert.Error(t, err)
}

func TestUpdatePassword(t *testing.T) {
	_, err := UpdatePassword([]byte("foobarbaz"), []byte("lalala"), &defaultKeyStore)
	assert.NoError(t, err)

	_, err = ExistingCrypto([]byte("lalala"), &defaultKeyStore)
	assert.NoError(t, err)

	_, err = ExistingCrypto([]byte("lalalaWRONG"), &defaultKeyStore)
	assert.Error(t, err)
}

func TestSealBytes(t *testing.T) {
	data := []byte("foo")
	aData := []byte{0, 1, 2}
	ciphertext := defaultCrypto.SealBytes(data, aData)
	plaintext, err := defaultCrypto.OpenBytes(ciphertext, aData)
	assert.NoError(t, err)
	assert.EqualValues(t, plaintext, data)
}
