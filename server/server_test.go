package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/marcboeker/supertar/archive"
	"github.com/marcboeker/supertar/config"

	"github.com/marcboeker/supertar/item"
	"github.com/stretchr/testify/suite"
)

var (
	itm = item.Item{Header: &item.Header{Path: "test/foo.txt"}}
)

type ServerTestSuite struct {
	suite.Suite
	server  *Server
	archive *archive.Archive
}

func (s *ServerTestSuite) SetupTest() {
	config := &config.Config{Path: "/tmp/foo.star", Password: []byte("foobar"), Compression: true, ChunkSize: 1024 * 1024}
	s.archive, _ = archive.NewArchive(config)
	s.archive.AddRecursive("../", "../archive", nil)

	s.server = &Server{
		archive:   s.archive,
		index:     map[string][]*item.Item{},
		rootItems: []*item.Item{},
	}

	err := s.server.buildIndex()
	s.NoError(err)
}

func (s ServerTestSuite) TearDownTest() {
	s.archive.Close()
	os.Remove("/tmp/foo.star")
}

func (s ServerTestSuite) TestBuildIndex() {
	s.Len(s.server.rootItems, 1)
	s.Len(s.server.index, 2)

	s.Equal(s.server.rootItems[0].Header.Path, "archive")
}

func (s ServerTestSuite) TestListItems() {
	router := s.server.setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/items/", nil)
	router.ServeHTTP(w, req)

	s.Equal(200, w.Code)
	s.Contains(w.Body.String(), `"name":"archive"`)
}

func (s ServerTestSuite) TestListItemsWithPath() {
	router := s.server.setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/items/archive", nil)
	router.ServeHTTP(w, req)

	s.Equal(200, w.Code)
	s.Contains(w.Body.String(), "archive.go")
}

func (s ServerTestSuite) TestListItemsWithInvalidPath() {
	router := s.server.setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/items/foo", nil)
	router.ServeHTTP(w, req)

	s.Equal(404, w.Code)
}

func (s ServerTestSuite) TestStreamFound() {

	router := s.server.setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/stream/archive/archive.go", nil)
	router.ServeHTTP(w, req)

	s.Equal(200, w.Code)
	s.Contains(w.Body.String(), "package archive")
}
func (s ServerTestSuite) TestStreamNotFound() {
	router := s.server.setupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/stream/test/notfound.txt", nil)
	router.ServeHTTP(w, req)

	s.Equal(404, w.Code)
}

func (s ServerTestSuite) TestServeStaticIndex() {
	router := s.server.setupRouter()

	for k := range fileMap {
		if k == "/index.html" {
			k = "/"
		}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", k, nil)
		router.ServeHTTP(w, req)

		s.Equal(200, w.Code)
		s.NotZero(w.Body.String())
	}
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
