package archive

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

const (
	magicNumberLength = 4
	versionLength     = 1
	compressionLength = 1
	chunkSizeLength   = 8
	kdfSaltLength     = 16
	keyNonceLength    = 24
	keyLength         = 32
	tagLength         = 16

	headerLength = magicNumberLength + versionLength + compressionLength + chunkSizeLength + kdfSaltLength + keyNonceLength + keyLength + tagLength

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
	chunkSize   int    // chunkSizeLength
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

	buf := make([]byte, chunkSizeLength)
	binary.LittleEndian.PutUint64(buf, uint64(h.chunkSize))
	if _, err := w.Write(buf); err != nil {
		return err
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
	hdr := make([]byte, headerLength)
	if _, err := r.Read(hdr); err != nil {
		return err
	}

	buf := bytes.NewBuffer(hdr)
	if !bytes.Equal(h.readNBytes(buf, magicNumberLength), magicNumber) {
		return errInvalidMagicNumber
	}

	h.version, _ = buf.ReadByte()
	compression, _ := buf.ReadByte()
	h.compression = compression == compressionEnabled
	chunkSize := h.readNBytes(buf, chunkSizeLength)
	h.chunkSize = int(binary.LittleEndian.Uint64(chunkSize))
	h.kdfSalt = h.readNBytes(buf, kdfSaltLength)
	h.KeyNonce = h.readNBytes(buf, keyNonceLength)
	h.Key = h.readNBytes(buf, keyLength+tagLength)

	return nil
}

func (h *Header) readNBytes(r io.Reader, n int) []byte {
	buf := make([]byte, n)
	if _, err := r.Read(buf); err != nil {
		return nil
	}
	return buf
}

var (
	errInvalidMagicNumber = errors.New("invalid magic number")
)
