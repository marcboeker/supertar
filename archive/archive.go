package archive

import (
	"encoding/binary"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/marcboeker/supertar/config"
	"github.com/marcboeker/supertar/crypto"
	"github.com/marcboeker/supertar/item"
)

// Archive represents an archive.
type Archive struct {
	header *Header
	path   string
	file   *os.File
	config *config.Config
}

// NewArchive opens or creates a new archive. If an archive already
// exists, the archive is opened and the keystore is read.
func NewArchive(c *config.Config) (*Archive, error) {
	exists := true
	if _, err := os.Stat(c.Path); os.IsNotExist(err) {
		exists = false
	}

	fh, err := os.OpenFile(c.Path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	arch := Archive{path: c.Path, file: fh, header: &Header{}}
	if exists {
		if _, err := arch.file.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}

		if err := arch.header.Read(fh); err != nil {
			return nil, err
		}

		c.Compression = arch.header.compression
		c.ChunkSize = arch.header.chunkSize

		ks := crypto.KeyStore{
			KDFSalt:  arch.header.kdfSalt,
			KeyNonce: arch.header.KeyNonce,
			Key:      arch.header.Key,
		}

		c.Crypto, err = crypto.ExistingCrypto(c.Password, &ks)
		if err != nil {
			return nil, err
		}
	} else {
		var (
			ks  *crypto.KeyStore
			err error
		)
		c.Crypto, ks, err = crypto.NewCrypto(c.Password)
		if err != nil {
			return nil, err
		}

		arch.header.compression = c.Compression
		arch.header.chunkSize = c.ChunkSize
		arch.header.kdfSalt = ks.KDFSalt
		arch.header.KeyNonce = ks.KeyNonce
		arch.header.Key = ks.Key

		if _, err := arch.file.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		arch.header.Write(fh)
	}

	arch.config = c
	arch.header.version = supertarVersion

	if err != nil {
		return nil, err
	}

	return &arch, nil
}

// Close closes the file handler of the archive.
// After an archive is closed, it is unusable.
func (a Archive) Close() {
	a.file.Close()
}

// Config returns the archive's config.
func (a Archive) Config() *config.Config {
	return a.config
}

// Add adds a new file or directory to the archive.
// It strips the base path from the file path to make the
// file path relative.
func (a Archive) Add(basePath, path string) error {
	if path == basePath || path == a.path {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	size := int64(stat.Size())
	chunks := math.Ceil(float64(size) / float64(a.config.ChunkSize))
	if stat.IsDir() {
		size = 0
		chunks = 0
	}

	path, err = filepath.Rel(basePath, path)
	if err != nil {
		return err
	}

	b := filepath.Base(path)
	if b == "." || b == ".." {
		return nil
	}

	if strings.HasPrefix(path, "/") {
		path = strings.TrimPrefix(path, "/")
	}

	hdr := item.Header{
		Path:   path,
		Size:   size,
		MTime:  stat.ModTime(),
		Mode:   stat.Mode(),
		Chunks: int64(chunks),
	}

	e := item.NewItem(&hdr)
	if _, err := a.file.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	return e.Write(a.file, file, a.config)
}

// AddRecursive adds a directory and all its children to
// an archive. All path names are made relative.
func (a Archive) AddRecursive(basePath, path string, ch chan string) error {
	walkFnc := func(path string, info os.FileInfo, err error) error {
		if ch != nil {
			ch <- path
		}
		return a.Add(basePath, path)
	}

	return filepath.Walk(path, walkFnc)
}

func (a Archive) iterateItems(cb func(*item.Item) error) error {
	if _, err := a.file.Seek(headerLength, io.SeekStart); err != nil {
		return err
	}

	for {
		i, err := item.Read(a.file, a.config)
		if err != nil {
			return err
		}
		if i == nil {
			return nil
		}

		if i.Offset, err = a.file.Seek(0, io.SeekCurrent); err != nil {
			return err
		}

		if err := cb(i); err != nil {
			return err
		}
	}
}

// List lists all files and directories of an archive.
func (a Archive) List(ch chan *item.Item, pattern string) error {
	defer func() {
		close(ch)
	}()
	return a.iterateItems(func(i *item.Item) error {
		if len(pattern) > 0 {
			matched, err := filepath.Match(pattern, i.Header.Path)
			if err != nil {
				return err
			}
			if matched {
				ch <- i
			}
		} else {
			ch <- i
		}

		if i.Header.Type() == item.ModeRegular {
			a.skipChunks(i.Header.Chunks)
		}

		return nil
	})
}

// Delete searches for the given glob and marks the entry as deleted.
func (a Archive) Delete(ch chan *item.Item, pattern string) error {
	if ch != nil {
		defer func() {
			close(ch)
		}()
	}
	return a.iterateItems(func(i *item.Item) error {
		if ch != nil {
			ch <- i
		}

		matched, err := filepath.Match(pattern, i.Header.Path)
		if err != nil {
			return err
		}
		if matched {
			i.Header.Deleted = 1
			a.file.Seek(-i.Header.Len(), io.SeekCurrent)
			i.Header.Write(a.file, a.config)
		}

		if i.Header.Type() == item.ModeRegular {
			if _, err := a.skipChunks(i.Header.Chunks); err != nil {
				return err
			}
		}

		return nil
	})
}

// Move moves items matched by the given pattern to its new destination.
func (a Archive) Move(ch chan *item.Item, src, target string) error {
	if ch != nil {
		defer func() {
			close(ch)
		}()
	}

	type matchedItem struct {
		item  *item.Item
		start int64
		end   int64
	}

	var matchedItems []*matchedItem
	toIsFile := false
	err := a.iterateItems(func(i *item.Item) error {
		if toIsFile && target == i.Header.Path && i.Header.Type() == item.ModeRegular {
			toIsFile = false
		}

		matched, err := filepath.Match(src, i.Header.Path)
		if err != nil {
			return err
		}

		start, err := a.file.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		var end int64
		if i.Header.Type() == item.ModeRegular {
			end, err = a.skipChunks(i.Header.Chunks)
			if err != nil {
				return err
			}
		}

		if matched {
			if ch != nil {
				ch <- i
			}

			matchedItems = append(matchedItems, &matchedItem{i, start, end})
		}

		return nil
	})
	if err != nil {
		return err
	}

	writeFile, err := os.OpenFile(a.path, os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer func() {
		writeFile.Sync()
		writeFile.Close()
	}()

	if _, err := writeFile.Seek(0, io.SeekEnd); err != nil {
		return err
	}

	// If moving multiple items the destination path cannot be a file.
	if len(matchedItems) > 1 && toIsFile {
		return errors.New("destination path is an existing file, should be missing or directory")
	}

	for _, mi := range matchedItems {
		if _, err := a.file.Seek(mi.item.Offset-mi.item.Header.Len(), io.SeekStart); err != nil {
			return err
		}

		// Make copy of header
		hdr := *mi.item.Header

		mi.item.Header.Deleted = 1
		if err := mi.item.Header.Write(a.file, a.config); err != nil {
			return err
		}

		if len(matchedItems) > 1 {
			// Multiple items are prefixed with the target path.
			hdr.Path = filepath.Join(target, mi.item.Header.Path)
		} else {
			hdr.Path = target
		}
		if err := hdr.Write(writeFile, a.config); err != nil {
			return nil
		}

		if _, err := io.CopyN(writeFile, a.file, mi.end-mi.start); err != nil {
			return err
		}
	}

	return nil
}

// Compact removes all entries that are marked as deleted.
func (a Archive) Compact() error {
	if _, err := a.file.Seek(headerLength, io.SeekStart); err != nil {
		return err
	}

	slices := [][2]int64{}
	curOffset := int64(headerLength)

	a.iterateItems(func(i *item.Item) error {
		lastOffset := curOffset
		curOffset += i.Header.Len()

		if i.Header.Type() == item.ModeRegular && i.Header.Size > 0 {
			pos, err := a.skipChunks(i.Header.Chunks)
			if err != nil {
				return err
			}

			curOffset = pos
		}

		if i.Header.Deleted == 1 {
			slices = append(slices, [2]int64{lastOffset, curOffset})
		}

		return nil
	})

	stat, err := a.file.Stat()
	if err != nil {
		return err
	}
	slices = append(slices, [2]int64{stat.Size(), 0})

	writeFile, err := os.OpenFile(a.path, os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	if _, err := a.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	newSize := int64(0)
	offset := int64(0)
	for _, slice := range slices {
		n, err := io.CopyN(writeFile, a.file, slice[0]-offset)
		if err != nil {
			return err
		}
		newSize += n

		offset = slice[1]
		if offset > 0 {
			if _, err := a.file.Seek(offset, io.SeekStart); err != nil {
				return err
			}
		}
	}

	writeFile.Sync()
	writeFile.Truncate(newSize)
	writeFile.Close()

	return nil
}

// Extract extracts the archive to the give base path.
func (a Archive) Extract(ch chan *item.Item, dest string) error {
	defer func() {
		close(ch)
	}()
	return a.iterateItems(func(i *item.Item) error {
		ch <- i

		path := filepath.Join(dest, i.Header.Path)
		if i.Header.Type() == item.ModeRegular {
			if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
				return err
			}

			dest, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
			if err != nil {
				return err
			}

			if i.Header.Size > 0 {
				if err := i.Extract(a.file, dest, a.config); err != nil {
					dest.Close()
					return err
				}
			}

			dest.Close()
		} else if i.Header.Type() == item.ModeDir {
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				return err
			}
		}

		return os.Chtimes(path, i.Header.MTime, i.Header.MTime)
	})
}

// Stream streams an item from the archive.
func (a Archive) Stream(item *item.Item, dest io.Writer, start, end int) error {
	if _, err := a.file.Seek(item.Offset, io.SeekStart); err != nil {
		return err
	}

	if item.Header.Size > 0 {
		if err := item.ExtractRange(a.file, dest, start, end, a.config); err != nil {
			return err
		}
	}

	return nil
}

// UpdatePassword updates the password of the archive.
func (a *Archive) UpdatePassword(newPassword []byte) error {
	ks := crypto.KeyStore{
		KDFSalt:  a.header.kdfSalt,
		KeyNonce: a.header.KeyNonce,
		Key:      a.header.Key,
	}
	newKS, err := crypto.UpdatePassword(a.config.Password, newPassword, &ks)
	if err != nil {
		return err
	}
	a.header.kdfSalt = newKS.KDFSalt
	a.header.KeyNonce = newKS.KeyNonce
	a.header.Key = newKS.Key

	if _, err := a.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return a.header.Write(a.file)
}

func (a Archive) skipChunks(n int64) (int64, error) {
	var pos int64
	for i := int64(0); i < n; i++ {
		hdr := make([]byte, 8)
		_, err := a.file.Read(hdr)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}

		size := binary.LittleEndian.Uint32(hdr[4:])
		pos, err = a.file.Seek(int64(size), io.SeekCurrent)
		if err != nil {
			return 0, err
		}
	}

	return pos, nil
}
