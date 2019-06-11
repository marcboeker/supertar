package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/poly1305"
)

const (
	kdfTime    = 1
	kdfMemory  = 64 * 1024
	kdfThreads = 4

	keyLength  = 32
	saltLength = 16

	// Overhead includes the nonce size and the auth tag size.
	Overhead = chacha20poly1305.NonceSizeX + poly1305.TagSize
	// ChunkOverhead is the normal overhead plus the header for each chunk.
	ChunkOverhead = Overhead + 8
)

// Crypto represents a wrapper for AES de- and encryption.
type Crypto struct {
	aead cipher.AEAD
}

// KeyStore holds all information necessary to derive a key from the
// users password and decrypt the data key.
type KeyStore struct {
	KDFSalt  []byte // 16 byte
	KeyNonce []byte // 12 byte
	Key      []byte // 48 byte (32 bytes key, 16 bytes auth)
}

// ExistingCrypto returns a crypto wrapper for the given key.
func ExistingCrypto(password []byte, ks *KeyStore) (*Crypto, error) {
	key := argon2.IDKey(password, ks.KDFSalt, kdfTime, kdfMemory, kdfThreads, keyLength)

	dataKey, err := decryptKey(key, ks.KeyNonce, ks.Key)
	if err != nil {
		return nil, err
	}

	aead, _ := chacha20poly1305.NewX(dataKey)

	dataKey = nil

	return &Crypto{aead: aead}, nil
}

// NewCrypto returns a crypto wrapper for the given key.
func NewCrypto(password []byte) (*Crypto, *KeyStore, error) {
	salt := make([]byte, saltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, err
	}

	dataNonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := io.ReadFull(rand.Reader, dataNonce); err != nil {
		return nil, nil, err
	}

	dataKey := make([]byte, keyLength)
	if _, err := io.ReadFull(rand.Reader, dataKey); err != nil {
		return nil, nil, err
	}

	derivedKey := argon2.IDKey(password, salt, kdfTime, kdfMemory, kdfThreads, keyLength)

	aead, _ := chacha20poly1305.NewX(dataKey)

	ks := KeyStore{
		KDFSalt:  salt,
		KeyNonce: dataNonce,
		Key:      encryptKey(derivedKey, dataNonce, dataKey),
	}

	dataKey = nil
	derivedKey = nil

	return &Crypto{aead: aead}, &ks, nil
}

// UpdatePassword re-encrypts the archive key with the new password.
func UpdatePassword(oldPwd, newPwd []byte, ks *KeyStore) (*KeyStore, error) {
	key := argon2.IDKey(oldPwd, ks.KDFSalt, kdfTime, kdfMemory, kdfThreads, keyLength)

	dataKey, err := decryptKey(key, ks.KeyNonce, ks.Key)
	if err != nil {
		return nil, err
	}

	if _, err := io.ReadFull(rand.Reader, ks.KDFSalt); err != nil {
		return nil, err
	}

	if _, err := io.ReadFull(rand.Reader, ks.KeyNonce); err != nil {
		return nil, err
	}

	derivedKey := argon2.IDKey(newPwd, ks.KDFSalt, kdfTime, kdfMemory, kdfThreads, keyLength)

	ks.Key = encryptKey(derivedKey, ks.KeyNonce, dataKey)

	derivedKey = nil
	dataKey = nil

	return ks, nil
}

func decryptKey(key, nonce, encryptedKey []byte) ([]byte, error) {
	aead, _ := chacha20poly1305.NewX(key)
	return aead.Open(nil, nonce, encryptedKey, nil)
}

func encryptKey(key, nonce, decryptedKey []byte) []byte {
	aead, _ := chacha20poly1305.NewX(key)
	return aead.Seal(nil, nonce, decryptedKey, nil)
}

// SealBytes takes the plaintext and encrypts its contents and the ciphertext.
func (c Crypto) SealBytes(plaintext, data []byte) []byte {
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil
	}

	ciphertext := c.aead.Seal(nil, nonce, plaintext, data)

	return append(nonce, ciphertext...)
}

// OpenBytes decrypts the contents of an io.Reader to an io.Writer.
func (c Crypto) OpenBytes(ciphertext, data []byte) ([]byte, error) {
	nonce := ciphertext[0:chacha20poly1305.NonceSizeX]

	return c.aead.Open(nil, nonce, ciphertext[chacha20poly1305.NonceSizeX:], data)
}
