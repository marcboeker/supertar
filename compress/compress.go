package compress

import (
	"github.com/DataDog/zstd"
)

// Decompress decompresses a byte stream to the given destination.
func Decompress(src []byte) ([]byte, error) {
	return zstd.Decompress(nil, src)
}

// Compress compresses a byte stream to the given destination.
func Compress(src []byte) []byte {
	buf, _ := zstd.Compress(nil, src)
	return buf
}
