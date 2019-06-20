package server

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/marcboeker/supertar/archive"
	"github.com/marcboeker/supertar/item"
)

const (
	bindAddr = "localhost:1337"
)

var (
	mutex = sync.Mutex{}
)

// Server holds the archive together with the item cache.
type Server struct {
	archive   *archive.Archive
	index     map[string][]*item.Item
	rootItems []*item.Item
}

type apiItem struct {
	Path  string    `json:"path"`
	Name  string    `json:"name"`
	IsDir bool      `json:"isDir"`
	Size  string    `json:"size"`
	MTime time.Time `json:"mtime"`
}

// Start starts the integrated webserver to enable browsing the archive.
func Start(arch *archive.Archive) (*Server, error) {
	s := Server{
		archive:   arch,
		index:     map[string][]*item.Item{},
		rootItems: []*item.Item{},
	}
	if err := s.buildIndex(); err != nil {
		return nil, err
	}
	r := s.setupRouter()
	if err := r.Run(bindAddr); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Server) buildIndex() error {
	wg := sync.WaitGroup{}
	wg.Add(1)

	ch := make(chan *item.Item)
	go func() {
		curLevel := -1
		for {
			i, more := <-ch
			if more {
				if curLevel == -1 {
					curLevel = strings.Count(i.Header.Path, "/")
				}

				dir := filepath.Dir(i.Header.Path)
				if _, ok := s.index[dir]; ok {
					s.index[dir] = append(s.index[dir], i)
				} else {
					s.index[dir] = []*item.Item{i}
				}

				level := strings.Count(i.Header.Path, "/")
				if level < curLevel {
					curLevel = level
					s.rootItems = append(s.rootItems, i)
				} else if level == curLevel {
					s.rootItems = append(s.rootItems, i)
				}
			} else {
				wg.Done()
				return
			}
		}
	}()

	err := s.archive.List(ch, "")
	wg.Wait()

	return err
}

func (s Server) setupRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.GET("/", s.serveStatic)
	r.GET("/js/:file", s.serveStatic)
	r.GET("/css/:file", s.serveStatic)

	api := r.Group("/api")
	{
		api.GET("/items/*path", s.listItems)
		api.GET("/stream/*path", s.streamItem)
	}

	return r
}

func (s Server) listItems(c *gin.Context) {
	var items []*item.Header

	path := strings.TrimPrefix(c.Param("path"), "/")
	if len(path) == 0 {
		for _, i := range s.rootItems {
			items = append(items, i.Header)
		}
	} else if _, ok := s.index[path]; ok {
		for _, i := range s.index[path] {
			items = append(items, i.Header)
		}
	} else {
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, toAPIItems(items))
}

func (s Server) streamItem(c *gin.Context) {
	mutex.Lock()
	defer mutex.Unlock()

	item, err := s.resolveItem(c.Param("path"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Header("Accept-Ranges", "bytes")

	ext := filepath.Ext(item.Header.Path)
	mt := mime.TypeByExtension(ext)
	c.Header("Content-Type", mt)

	start := 0
	end := int(item.Header.Size)

	rh := c.GetHeader("Range")
	if len(rh) > 0 {
		rh = strings.TrimPrefix(rh, "bytes=")
		r := strings.Split(rh, "-")
		start, _ = strconv.Atoi(r[0])
		if len(r[1]) > 0 {
			end, _ = strconv.Atoi(r[1])
		}

		if start > end {
			start = 0
		}

		c.Status(http.StatusPartialContent)
		if int64(end) == item.Header.Size {
			c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end-1, item.Header.Size))
			c.Header("Content-Length", strconv.FormatInt(int64(end-start), 10))
		} else {
			c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, item.Header.Size))
			c.Header("Content-Length", strconv.FormatInt(int64(end-start)+1, 10))
		}
	} else {
		c.Status(http.StatusOK)
		c.Header("Content-Length", strconv.FormatInt(item.Header.Size, 10))
	}

	if err := s.archive.Stream(item, c.Writer, start, end); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
}

func (s Server) serveStatic(c *gin.Context) {
	key := c.Request.URL.Path
	if c.Request.URL.Path == "/" {
		key = "/index.html"
	}
	if s, ok := fileMap[key]; ok {
		c.Status(http.StatusOK)

		if strings.HasSuffix(c.Request.URL.Path, ".css") {
			c.Header("Content-Type", "text/css")
		} else if strings.HasSuffix(c.Request.URL.Path, ".js") {
			c.Header("Content-Type", "text/javascript")
		}

		c.Writer.Write([]byte(s))
	}

	c.Status(http.StatusNotFound)
}

func toAPIItems(items []*item.Header) []*apiItem {
	var itms []*apiItem
	for _, i := range items {
		itms = append(itms, &apiItem{
			Path:  filepath.Dir(i.Path),
			Name:  filepath.Base(i.Path),
			IsDir: i.Type() == item.ModeDir,
			Size:  i.HumanSize(),
			MTime: i.MTime,
		})
	}
	return itms
}

func (s Server) resolveItem(path string) (*item.Item, error) {
	relPath := strings.TrimPrefix(path, "/")
	dir := filepath.Dir(relPath)

	for _, i := range s.index[dir] {
		if i.Header.Path == relPath {
			return i, nil
		}
	}

	return nil, errItemNotFound
}

var (
	errItemNotFound = errors.New("could not find item")
)
