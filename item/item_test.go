package item

import (
	"bytes"
	"testing"

	"github.com/marcboeker/supertar/config"
	"github.com/marcboeker/supertar/crypto"
	"github.com/stretchr/testify/assert"
)

const (
	chunkSize = 64 * 1024
)

var (
	defaultKeyStore = crypto.KeyStore{
		KDFSalt:  []byte{49, 226, 108, 3, 55, 36, 46, 86, 53, 219, 249, 67, 143, 113, 201, 210},
		KeyNonce: []byte{152, 162, 64, 109, 125, 231, 165, 166, 42, 144, 91, 97, 62, 118, 193, 196, 222, 101, 250, 60, 206, 64, 157, 55},
		Key:      []byte{150, 152, 73, 143, 185, 32, 106, 178, 67, 206, 106, 61, 13, 86, 146, 66, 87, 11, 117, 89, 225, 225, 136, 225, 111, 227, 158, 70, 255, 122, 72, 209, 15, 152, 67, 74, 194, 242, 175, 71, 28, 5, 32, 211, 140, 166, 29, 206}}
	defaultCrypto, e = crypto.ExistingCrypto([]byte("foobarbaz"), &defaultKeyStore)
	defaultConfig    = config.Config{Crypto: defaultCrypto, ChunkSize: 1024 * 1024}
)

func TestSerializeDirItem(t *testing.T) {
	i := NewItem(&defaultDirHeader)
	assert.Equal(t, i.Header.Path, defaultDirHeader.Path)
	assert.Equal(t, i.Header.Type(), defaultDirHeader.Type())

	buf := bytes.NewBuffer(nil)
	err := i.Write(buf, nil, &defaultConfig)
	assert.NoError(t, err)

	assert.Equal(t, buf.Bytes()[:2], []byte{0x4a, 0x0})
}

func TestSerializeFileItem(t *testing.T) {
	mockFile := bytes.NewBufferString("eekeek")
	i := NewItem(&defaultFileHeader)
	assert.Equal(t, i.Header.Path, defaultFileHeader.Path)
	assert.Equal(t, i.Header.Type(), defaultFileHeader.Type())

	buf := bytes.NewBuffer(nil)
	err := i.Write(buf, mockFile, &defaultConfig)
	assert.NoError(t, err)

	assert.Equal(t, buf.Bytes()[:2], []byte{0x4e, 0x0})
}
