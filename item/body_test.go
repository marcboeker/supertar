package item

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

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
func TestWriteReadPartialBody(t *testing.T) {
	c := config.Config{Crypto: defaultCrypto, ChunkSize: 1024 * 1024, Compression: true}
	data := []byte("eekeek")

	b := new(Body)

	mockFile := bytes.NewBuffer(data)

	path := fmt.Sprintf("/tmp/%s", time.Now().Format(time.RFC1123))
	fh, err := os.Create(path)
	assert.NoError(t, err)
	defer func() {
		fh.Close()
		os.Remove(path)
	}()
	err = b.Write(fh, mockFile, &c)
	assert.NoError(t, err)

	out := bytes.NewBuffer(nil)

	fh.Seek(0, io.SeekStart)

	err = b.ExtractRange(fh, io.Writer(out), 0, 3, 1, &c)
	assert.NoError(t, err)

	assert.EqualValues(t, "eek", out.Bytes())
}
