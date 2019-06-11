package item

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	defaultDirHeader    = Header{Path: "foo", Size: 0, MTime: time.Unix(0, 0), Mode: os.FileMode(0755) | os.ModeDir, Deleted: 0, Chunks: 0}
	defaultFileHeader   = Header{Path: "foo.txt", Size: 100, MTime: time.Unix(0, 0), Mode: os.FileMode(0777), Deleted: 0, Chunks: 1}
	serializedDirEntry  = []byte{62, 0, 178, 122, 242, 171, 143, 117, 24, 227, 143, 47, 153, 111, 13, 161, 110, 128, 253, 77, 210, 111, 131, 114, 130, 245, 68, 33, 61, 86, 156, 182, 139, 194, 220, 139, 42, 36, 191, 115, 49, 192, 202, 21, 254, 63, 36, 240, 208, 213, 49, 33, 179, 106, 198, 50, 192, 145, 3, 103, 7, 243, 139, 252}
	serializedFileEntry = []byte{63, 0, 33, 187, 186, 116, 71, 10, 30, 219, 87, 94, 104, 252, 39, 206, 246, 81, 246, 254, 252, 25, 8, 204, 180, 148, 176, 44, 223, 241, 235, 125, 111, 189, 195, 59, 69, 255, 130, 175, 41, 87, 46, 196, 74, 16, 27, 165, 3, 160, 75, 113, 32, 32, 35, 110, 124, 189, 170, 209, 176, 4, 170, 115, 130}
)

func TestReadDirHeader(t *testing.T) {
	src := bytes.NewBuffer(nil)
	err := defaultDirHeader.Write(src, &defaultConfig)
	assert.NoError(t, err)

	h := new(Header)
	found, err := h.Read(src, &defaultConfig)
	assert.NoError(t, err)
	assert.True(t, found)

	assert.Equal(t, h.Path, defaultDirHeader.Path)
	assert.Equal(t, h.Type(), defaultDirHeader.Type())
	assert.Equal(t, h.Size, defaultDirHeader.Size)
	assert.Equal(t, h.MTime, defaultDirHeader.MTime)
	assert.Equal(t, h.Mode, defaultDirHeader.Mode)
	assert.Equal(t, h.Chunks, defaultDirHeader.Chunks)
}

func TestReadFileHeader(t *testing.T) {
	src := bytes.NewBuffer(nil)
	err := defaultFileHeader.Write(src, &defaultConfig)
	assert.NoError(t, err)

	h := new(Header)
	found, err := h.Read(src, &defaultConfig)
	assert.NoError(t, err)
	assert.True(t, found)

	assert.Equal(t, h.Path, defaultFileHeader.Path)
	assert.Equal(t, h.Type(), defaultFileHeader.Type())
	assert.Equal(t, h.Size, defaultFileHeader.Size)
	assert.Equal(t, h.MTime, defaultFileHeader.MTime)
	assert.Equal(t, h.Mode, defaultFileHeader.Mode)
	assert.Equal(t, h.Chunks, defaultFileHeader.Chunks)
}

func TestSerializeToJSON(t *testing.T) {
	j := defaultFileHeader.ToJSON()
	assert.NotNil(t, j)
	assert.EqualValues(t, j, `{"path":"foo.txt","size":100,"chunks":1,"mtime":"1970-01-01T01:00:00+01:00","mode":511,"deleted":0}`)
}

func TestToString(t *testing.T) {
	sizes := map[string]int64{
		"----------         2B\t0001-01-01 00:00:00\tfoo.txt": 2,
		"----------     2.290K\t0001-01-01 00:00:00\tfoo.txt": 2345,
		"----------    11.772M\t0001-01-01 00:00:00\tfoo.txt": 12343456,
		"----------   142.897G\t0001-01-01 00:00:00\tfoo.txt": 153434569073,
		"----------   139.548T\t0001-01-01 00:00:00\tfoo.txt": 153434569078399,
	}

	for k, v := range sizes {
		nh := Header{Path: "foo.txt", Size: v}
		assert.EqualValues(t, nh.ToString(), k)
	}

	j := defaultDirHeader.ToString()
	assert.NotNil(t, j)
	assert.EqualValues(t, j, "drwxr-xr-x         0B\t1970-01-01 01:00:00\tfoo")

}
