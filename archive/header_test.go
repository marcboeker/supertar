package archive

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	defaultHeader = Header{
		version:     supertarVersion,
		compression: true,
		kdfSalt:     []byte("deadbeeffoodbabe"),
		KeyNonce:    []byte("012345678912012345678912"),
		Key:         []byte("deadbeeffoodbabedeadbeeffoodbabe"),
	}
)

func TestHeaderOK(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	if err := defaultHeader.Write(buf); err != nil {
		t.Error(err)
	}

	assert.Equal(t, magicNumber, buf.Bytes()[0:4])

	hdr := new(Header)
	if err := hdr.Read(buf); err != nil {
		t.Error(err)
	}

	assert.Equal(t, defaultHeader.version, hdr.version)
	assert.Equal(t, defaultHeader.compression, hdr.compression)
	assert.Equal(t, defaultHeader.kdfSalt, hdr.kdfSalt)
	assert.Equal(t, defaultHeader.KeyNonce, hdr.KeyNonce)
	assert.Equal(t, defaultHeader.Key, hdr.Key[:32])
}

func TestHeaderCorrupted(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	err := defaultHeader.Write(buf)
	assert.NoError(t, err)

	tBuf := buf.Bytes()
	tBuf[4] = 254
	tBuf[7] = 254

	cBuf := bytes.NewBuffer(tBuf)

	hdr := new(Header)
	err = hdr.Read(cBuf)
	assert.NoError(t, err)

	assert.NotEqual(t, defaultHeader.version, hdr.version)
	assert.NotEqual(t, defaultHeader.kdfSalt, hdr.kdfSalt)
}

func TestMagicNumberCorrupted(t *testing.T) {
	buf := bytes.NewBuffer(nil)

	err := defaultHeader.Write(buf)
	assert.NoError(t, err)

	tBuf := buf.Bytes()
	tBuf[0] = 254

	assert.NotEqual(t, magicNumber, buf.Bytes()[0:4])

	cBuf := bytes.NewBuffer(tBuf)

	hdr := new(Header)
	err = hdr.Read(cBuf)
	assert.Error(t, err)
}
