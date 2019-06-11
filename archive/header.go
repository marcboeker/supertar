package archive

import (
	"bytes"
	"errors"
	"io"
)

const (
	magicNumberLength = 4
	versionLength     = 1
	compressionLength = 1
	kdfSaltLength     = 16
	keyNonceLength    = 24
	keyLength         = 32
	tagLength         = 16

	headerLength = magicNumberLength + versionLength + compressionLength + kdfSaltLength + keyNonceLength + keyLength + tagLength

	compressionDisabled = 0
	compressionEnabled  = 1

	supertarVersion = 1
)

var (
	magicNumber = []byte{1, 3, 3, 7}
)

// Header contains all necessary fields of an archive.
type Header struct {
	version     uint8  // versionLength
	compression bool   // compressionLength
	kdfSalt     []byte // kdfSaltLength
	KeyNonce    []byte // keyNonceLength
	Key         []byte // keyLength + tagLength
}

// Write serializes and writes the header to given file handler.
func (h Header) Write(w io.Writer) error {
	if _, err := w.Write(magicNumber); err != nil {
		return err
	}

	if _, err := w.Write([]byte{h.version}); err != nil {
		return err
	}

	if h.compression {
		if _, err := w.Write([]byte{compressionEnabled}); err != nil {
			return err
		}
	} else {
		if _, err := w.Write([]byte{compressionDisabled}); err != nil {
			return err
		}
	}

	if _, err := w.Write(h.kdfSalt); err != nil {
		return err
	}

	if _, err := w.Write(h.KeyNonce); err != nil {
		return err
	}

	if _, err := w.Write(h.Key); err != nil {
		return err
	}

	return nil
}

// Read reads the header from the given fikle handler.
func (h *Header) Read(r io.Reader) error {
	buf := make([]byte, headerLength)
	if _, err := r.Read(buf); err != nil {
		return err
	}

	if !bytes.Equal(buf[0:4], magicNumber) {
		return errInvalidMagicNumber
	}

	h.version = buf[4]
	afterCompression := magicNumberLength + versionLength + compressionLength
	h.compression = buf[5] == compressionEnabled
	afterKDFSalt := afterCompression + kdfSaltLength
	h.kdfSalt = buf[afterCompression:afterKDFSalt]
	afterKeyNonce := afterKDFSalt + keyNonceLength
	h.KeyNonce = buf[afterKDFSalt:afterKeyNonce]
	h.Key = buf[afterKeyNonce:]

	return nil
}

var (
	errInvalidMagicNumber = errors.New("invalid magic number")
)
