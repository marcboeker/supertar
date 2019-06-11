package compress

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompress(t *testing.T) {
	text := []byte("hello")
	out := Compress(text)
	assert.Equal(t, []byte{0x28, 0xb5, 0x2f, 0xfd, 0x20, 0x5, 0x29, 0x0, 0x0, 0x68, 0x65, 0x6c, 0x6c, 0x6f}, out)

	data, err := Decompress(out)
	assert.NoError(t, err)

	assert.Equal(t, data, text)
}
