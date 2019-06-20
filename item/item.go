package item

import (
	"io"

	"github.com/marcboeker/supertar/config"
)

// Item represents an item in an archive.
type Item struct {
	Header *Header
	Offset int64
}

// NewItem returns a new item in an archive.
func NewItem(header *Header) *Item {
	return &Item{Header: header}
}

// Read reads the header of an item from the archive file.
func Read(src io.Reader, config *config.Config) (*Item, error) {
	h := new(Header)
	if found, err := h.Read(src, config); err != nil || !found {
		return nil, err
	}

	i := Item{Header: h}

	return &i, nil
}

// Write serializes an item to the archive file.
func (i Item) Write(dest io.Writer, src io.Reader, config *config.Config) error {
	if err := i.Header.Write(dest, config); err != nil {
		return err
	}

	if i.Header.Type() == ModeRegular && i.Header.Size > 0 {
		body := new(Body)
		if err := body.Write(dest, src, config); err != nil {
			return err
		}
	}

	return nil
}

// Extract reads the body of an item and writes it to dest.
func (i Item) Extract(src io.Reader, dest io.Writer, config *config.Config) error {
	body := new(Body)
	if err := body.Extract(src, dest, i.Header.Chunks, config); err != nil {
		return err
	}
	return nil
}

// ExtractRange reads the given range from an item and writes it to dest.
func (i Item) ExtractRange(src io.ReadSeeker, dest io.Writer, start, end int, config *config.Config) error {
	body := new(Body)
	if err := body.ExtractRange(src, dest, start, end, i.Header.Chunks, config); err != nil {
		return err
	}
	return nil
}
