package item

import (
	"bytes"
	"io"
	"testing"

	"github.com/marcboeker/supertar/config"
	"github.com/stretchr/testify/assert"
)

func TestWriteReadBody(t *testing.T) {
	c := config.Config{Crypto: defaultCrypto, ChunkSize: 1024 * 1024, Compression: true}
	data := []byte("eekeek")

	buf := bytes.NewBuffer(nil)
	b := new(Body)

	mockFile := bytes.NewBuffer(data)
	err := b.Write(buf, mockFile, &c)
	assert.NoError(t, err)

	out := bytes.NewBuffer(nil)

	err = b.Extract(io.Reader(buf), io.Writer(out), 1, &c)
	assert.NoError(t, err)

	assert.Equal(t, data, out.Bytes())
}
