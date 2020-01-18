package item

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/marcboeker/supertar/config"
	"github.com/marcboeker/supertar/crypto"
)

const (
	pathLength    = 2
	sizeLength    = 8
	deletedLength = 1
	chunksLength  = 8
	timeLength    = 8
	modeLength    = 4

	headerSizeLength = 2
	minHeaderLength  = pathLength + timeLength + modeLength

	kb = 1024
	mb = kb * 1024
	gb = mb * 1024
	tb = gb * 1024
)

// Header represents a file or directory of an entry.
type Header struct {
	Path    string      `json:"path"`    // 2 bytes + x bytes
	Size    int64       `json:"size"`    // 8 bytes
	Chunks  int64       `json:"chunks"`  // 8 bytes
	MTime   time.Time   `json:"mtime"`   // 8 bytes
	Mode    os.FileMode `json:"mode"`    // 4 bytes
	Deleted int         `json:"deleted"` // 1 byte

	serializedLength uint16
}

// Type returns the entry's type.
func (h Header) Type() Mode {
	if h.Mode.IsRegular() {
		return ModeRegular
	} else if h.Mode.IsDir() {
		return ModeDir
	}
	return 0
}

// Read reads an header from a file handler and parses it.
func (h *Header) Read(src io.Reader, config *config.Config) (bool, error) {
	sizeBuf := make([]byte, headerSizeLength)
	_, err := src.Read(sizeBuf)
	if err == io.EOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	hdrLen := binary.LittleEndian.Uint16(sizeBuf)

	if hdrLen < minHeaderLength {
		return false, errors.New("header is invalid as it is too short")
	}

	hdrBuf := make([]byte, hdrLen)
	if _, err = src.Read(hdrBuf); err != nil {
		return false, err
	}

	hdrBuf, err = config.Crypto.OpenBytes(hdrBuf, sizeBuf)
	if err != nil {
		return false, err
	}

	offset := 0
	pathLen := binary.LittleEndian.Uint16(hdrBuf[:pathLength])
	offset += pathLength + int(pathLen)
	h.Path = string(hdrBuf[pathLength:offset])

	h.Size = int64(binary.LittleEndian.Uint64(hdrBuf[offset : offset+sizeLength]))

	offset += sizeLength
	h.Chunks = int64(binary.LittleEndian.Uint64(hdrBuf[offset : offset+chunksLength]))
	offset += chunksLength

	mtime := int64(binary.LittleEndian.Uint64(hdrBuf[offset : offset+timeLength]))
	h.MTime = time.Unix(mtime, 0)
	offset += timeLength

	h.Mode = os.FileMode(binary.LittleEndian.Uint32(hdrBuf[offset : offset+modeLength]))
	offset += modeLength

	if hdrBuf[offset] == 0 {
		h.Deleted = 0
	} else {
		h.Deleted = 1
	}

	h.serializedLength = hdrLen

	return true, nil
}

// Write serializes an header and writes it to a file handler.
func (h *Header) Write(dest io.Writer, config *config.Config) error {
	hdr := bytes.NewBuffer(nil)

	pathSizeBuf := make([]byte, pathLength)
	binary.LittleEndian.PutUint16(pathSizeBuf, uint16(len(h.Path)))

	hdr.Write(pathSizeBuf)
	hdr.Write([]byte(h.Path))

	sizeBuf := make([]byte, sizeLength)
	binary.LittleEndian.PutUint64(sizeBuf, uint64(h.Size))
	hdr.Write(sizeBuf)

	chunksBuf := make([]byte, chunksLength)
	binary.LittleEndian.PutUint64(chunksBuf, uint64(h.Chunks))
	hdr.Write(chunksBuf)

	timeBuf := make([]byte, timeLength)
	binary.LittleEndian.PutUint64(timeBuf, uint64(h.MTime.Unix()))
	hdr.Write(timeBuf)

	modeBuf := make([]byte, modeLength)
	binary.LittleEndian.PutUint32(modeBuf, uint32(h.Mode))
	hdr.Write(modeBuf)

	if h.Deleted == 1 {
		hdr.Write([]byte{1})
	} else {
		hdr.Write([]byte{0})
	}

	hdrLenBuf := make([]byte, headerSizeLength)
	overhead := hdr.Len() + crypto.Overhead

	binary.LittleEndian.PutUint16(hdrLenBuf, uint16(overhead))

	buf := config.Crypto.SealBytes(hdr.Bytes(), hdrLenBuf)

	if _, err := dest.Write(hdrLenBuf); err != nil {
		return err
	}

	_, err := dest.Write(buf)

	h.serializedLength = uint16(overhead)

	return err
}

// Len returns the serialized length of the header.
func (h Header) Len() int64 {
	return headerSizeLength + int64(h.serializedLength)
}

// ToJSON serializes the header to JSON.
func (h Header) ToJSON() []byte {
	b, err := json.Marshal(h)
	if err != nil {
		return nil
	}
	return b
}

// ToString formats the header to a string.
func (h Header) ToString() string {
	return fmt.Sprintf("%s %s%s\t%s\t%s", os.FileMode(h.Mode).String(), h.IsDeleted(), h.HumanSize(), h.MTime.Format("2006-01-02 15:04:05"), h.Path)
}

// IsDeleted returns a human readable flag if the item is marked for deletion.
func (h Header) IsDeleted() string {
	if h.Deleted == 1 {
		return "(del)"
	}
	return "\t"
}

// HumanSize returns the file size in a human readable format.
func (h Header) HumanSize() string {
	if h.Size < kb {
		return fmt.Sprintf("%10dB", h.Size)
	} else if h.Size >= kb && h.Size < mb {
		return fmt.Sprintf("%10.3fK", float64(h.Size)/float64(kb))
	} else if h.Size >= mb && h.Size < gb {
		return fmt.Sprintf("%10.3fM", float64(h.Size)/float64(mb))
	} else if h.Size >= gb && h.Size < tb {
		return fmt.Sprintf("%10.3fG", float64(h.Size)/float64(gb))
	}

	return fmt.Sprintf("%10.3fT", float64(h.Size)/float64(tb))
}
