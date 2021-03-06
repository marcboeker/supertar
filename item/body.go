package item

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/marcboeker/supertar/compress"
	"github.com/marcboeker/supertar/config"
	"github.com/marcboeker/supertar/crypto"
)

// Body wraps all functions to write and extract the body of an item.
type Body struct{}

func (b Body) Write(dest io.Writer, src io.Reader, c *config.Config) error {
	seq := 0
	for {
		buf := make([]byte, c.ChunkSize)
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		seqB := make([]byte, 4)
		binary.LittleEndian.PutUint32(seqB, uint32(seq))

		cb := buf[:n]
		if c.Compression {
			cb = compress.Compress(cb)
		}

		sizeB := make([]byte, 4)
		size := len(cb) + crypto.Overhead
		binary.LittleEndian.PutUint32(sizeB, uint32(size))

		hdr := append(seqB, sizeB...)

		res := c.Crypto.SealBytes(cb, hdr)

		dest.Write(hdr)
		dest.Write(res)

		if n < c.ChunkSize {
			break
		}

		seq++
	}

	return nil
}

// Extract extracts the body to the destination file.
func (b Body) Extract(src io.Reader, dest io.Writer, chunks int64, c *config.Config) error {
	for i := int64(0); i < chunks; i++ {
		hdr := make([]byte, 8)
		if _, err := src.Read(hdr); err != nil {
			return err
		}

		seq := binary.LittleEndian.Uint32(hdr[:4])
		size := binary.LittleEndian.Uint32(hdr[4:])

		if int64(seq) != i {
			return fmt.Errorf("chunk order incorrect: expected %d, got %d", i, seq)
		}

		buf := make([]byte, size)
		_, err := src.Read(buf)
		if err != nil {
			return err
		}

		plaintext, err := c.Crypto.OpenBytes(buf, hdr)
		if err != nil {
			return err
		}

		if c.Compression {
			data, err := compress.Decompress(plaintext)
			if err != nil {
				return err
			}
			if _, err := dest.Write(data); err != nil {
				return err
			}
		} else {
			if _, err := dest.Write(plaintext); err != nil {
				return err
			}
		}
	}

	return nil
}

// ExtractRange extracts the given range to the destination file.
func (b Body) ExtractRange(src io.ReadSeeker, dest io.Writer, start, end int, chunks int64, c *config.Config) error {
	counter := 0
	for i := int64(0); i < chunks; i++ {
		hdr := make([]byte, 8)
		if _, err := src.Read(hdr); err != nil {
			return err
		}

		seq := binary.LittleEndian.Uint32(hdr[:4])
		size := binary.LittleEndian.Uint32(hdr[4:8])

		if int64(seq) != i {
			return fmt.Errorf("chunk order incorrect: expected %d, got %d", i, seq)
		}

		if counter+c.ChunkSize >= start && counter < end {
			buf := make([]byte, size)
			_, err := src.Read(buf)
			if err != nil {
				return err
			}

			plaintext, err := c.Crypto.OpenBytes(buf, hdr)
			if err != nil {
				return err
			}

			startOffset := 0
			if start > counter && start < counter+c.ChunkSize {
				startOffset = start - counter
			}

			endOffset := c.ChunkSize
			if counter+c.ChunkSize > end {
				endOffset = c.ChunkSize - ((counter + c.ChunkSize) - end)
			}

			if c.Compression {
				data, err := compress.Decompress(plaintext)
				if err != nil {
					return err
				}
				if endOffset < len(data) {
					endOffset++
				}
				if _, err := dest.Write(data[startOffset:endOffset]); err != nil {
					return err
				}
			} else {
				if endOffset < len(plaintext) {
					endOffset++
				}
				if _, err := dest.Write(plaintext[startOffset:endOffset]); err != nil {
					return err
				}
			}
		} else {
			src.Seek(int64(size), io.SeekCurrent)
		}

		counter += c.ChunkSize
	}

	return nil
}
