package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/marcboeker/supertar/config"
	"github.com/marcboeker/supertar/item"
	"github.com/stretchr/testify/suite"
)

type ArchiveTestSuite struct {
	suite.Suite
	tmpDir string
	config *config.Config
	arch   *Archive
}

func (s *ArchiveTestSuite) SetupTest() {
	var err error
	s.tmpDir = os.TempDir()
	path := fmt.Sprintf("/tmp/%s.star", time.Now().Format(time.RFC1123))
	s.config = &config.Config{Path: path, Password: []byte("foobar"), Compression: true, ChunkSize: 1024 * 1024}
	s.arch, err = NewArchive(s.config)
	s.Assert().NoError(err)
}

func (s *ArchiveTestSuite) TearDownTest() {
	s.arch.Close()
	err := os.Remove(s.config.Path)
	s.Assert().NoError(err)
}

func (s *ArchiveTestSuite) TestArchiveExists() {
	stat, err := os.Stat(s.config.Path)
	s.Assert().NoError(err)
	if os.IsNotExist(err) {
		s.T().Error("archive file does not exist")
	}

	s.Assert().Equal(headerLength, int(stat.Size()))
}

func (s *ArchiveTestSuite) TestConfig() {
	s.arch.Close()

	arch, err := NewArchive(s.config)
	s.Assert().NoError(err)
	s.Assert().Equal(s.config, arch.config)

	arch.config.Compression = false
	s.Assert().False(arch.config.Compression)

	s.Assert().Equal(arch.Config(), arch.config)
}

func (s *ArchiveTestSuite) TestArchiveCompression() {
	s.arch.Add("", "../main.go")
	s.arch.Close()
	stat, _ := os.Stat(s.config.Path)
	os.Remove(s.config.Path)
	withCompression := stat.Size()

	arch, _ := NewArchive(&config.Config{Path: s.config.Path, Password: []byte("foobar"), Compression: false, ChunkSize: 1024 * 1024})
	arch.Add("", "../main.go")
	arch.Close()
	stat, _ = os.Stat(s.config.Path)
	withoutCompression := stat.Size()

	s.NotEqual(withCompression, withoutCompression)
}

func (s *ArchiveTestSuite) TestOpenExisting() {
	_, err := NewArchive(s.config)
	s.Assert().NoError(err)
}

func (s *ArchiveTestSuite) TestList() {
	err := s.arch.AddRecursive(".", "../main.go", nil)
	s.Assert().NoError(err)

	wg := sync.WaitGroup{}
	wg.Add(1)

	notFound := true

	ch := make(chan *item.Item)
	go func() {
		for {
			i, more := <-ch
			if i != nil && i.Header.Path == "../main.go" {
				notFound = false
			}
			if !more {
				wg.Done()
				return
			}
		}
	}()

	err = s.arch.List(ch, "*/main.go")
	s.Assert().NoError(err)

	wg.Wait()

	s.Assert().False(notFound, "could not find file main.go")
}

func (s *ArchiveTestSuite) TestExtract() {
	err := s.arch.AddRecursive(".", "archive.go", nil)
	s.Assert().NoError(err)

	path := filepath.Join(s.tmpDir, "archive-test")

	ch := make(chan *item.Item)

	go func() {
		for {
			<-ch
		}
	}()

	err = s.arch.Extract(ch, path)
	s.Assert().NoError(err)

	s.Assert().FileExists(filepath.Join(s.tmpDir, "archive-test", "archive.go"), "could not find main.go")

	os.RemoveAll(path)
}

func (s *ArchiveTestSuite) TestDelete() {
	err := s.arch.AddRecursive("../", "../main.go", nil)
	s.Assert().NoError(err)

	wg := sync.WaitGroup{}
	wg.Add(1)

	ch := make(chan *item.Item)
	go func() {
		for {
			_, more := <-ch
			if !more {
				wg.Done()
				return
			}
		}
	}()

	err = s.arch.Delete(ch, "main.go")
	s.Assert().NoError(err)
	wg.Wait()

	wg.Add(1)
	notFound := true

	ch = make(chan *item.Item)
	go func() {
		for {
			i, more := <-ch
			if i != nil && i.Header.Path == "main.go" && i.Header.Deleted == 1 {
				notFound = false
			}
			if !more {
				wg.Done()
				return
			}
		}
	}()

	err = s.arch.List(ch, "")
	s.Assert().NoError(err)

	wg.Wait()

	s.Assert().False(notFound)
}

func (s *ArchiveTestSuite) TestCompact() {
	err := s.arch.AddRecursive("../", "../archive", nil)
	s.Assert().NoError(err)

	// Delete file
	err = s.arch.Delete(nil, "*/header.go")
	s.Assert().NoError(err)

	// Delete directory
	err = s.arch.Delete(nil, "archive")
	s.Assert().NoError(err)

	err = s.arch.Compact()
	s.Assert().NoError(err)

	wg := sync.WaitGroup{}
	wg.Add(1)

	ch := make(chan *item.Item)
	go func() {
		for {
			i, more := <-ch
			if i != nil && i.Header.Path == "archive" {
				s.Assert().Fail("found archive")
			}
			if i != nil && i.Header.Path == "archive/header.go" {
				s.Assert().Fail("found archive/header.go")
			}

			if !more {
				wg.Done()
				return
			}
		}
	}()

	err = s.arch.List(ch, "")
	s.Assert().NoError(err)

	wg.Wait()
}

func (s *ArchiveTestSuite) TestInvalidPaths() {
	err := s.arch.Add("/tmp/", s.config.Path)
	s.Assert().NoError(err)

	err = s.arch.Add("/tmp", "/tmp/")
	s.Assert().NoError(err)

	wg := sync.WaitGroup{}
	wg.Add(1)

	ch := make(chan *item.Item)
	go func() {
		for {
			i, more := <-ch
			if i != nil && i.Header.Path == "foo.arch" {
				s.Fail("found foo.arch")
			}
			if i != nil && i.Header.Path == "." {
				s.Fail("found current dir '.'")
			}
			if !more {
				wg.Done()
				return
			}
		}
	}()

	err = s.arch.List(ch, "")
	s.Assert().NoError(err)

	wg.Wait()
}

func (s *ArchiveTestSuite) TestUpdatePassword() {
	err := s.arch.UpdatePassword([]byte("foo"))
	s.Assert().NoError(err)
	s.arch.Close()

	s.arch, err = NewArchive(s.config)
	s.Assert().Error(err)

	s.config.Password = []byte("foo")
	s.arch, err = NewArchive(s.config)
	s.Assert().NoError(err)
}

func TestArchiveTestSuite(t *testing.T) {
	suite.Run(t, new(ArchiveTestSuite))
}
