package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func emulateInflux() string {
	http.HandleFunc("/query", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
{
   "results" : [
      {
         "statement_id" : 0,
         "series" : [
            {
               "name" : "bpm",
               "values" : [
                  [
                     1537844700,
                     85
                  ],
                  [
                     1537845000,
                     78.25
                  ],
                  [
                     1537845300,
                     73
                  ],
                  [
                     1537845600,
                     71.8
                  ],
                  [
                     1537845900,
                     84.1666666666667
                  ],
                  [
                     1537846200,
                     68.5
                  ],
                  [
                     1537846500,
                     73.8
                  ],
                  [
                     1537846800,
                     70.8
                  ],
                  [
                     1537847100,
                     78.3333333333333
                  ],
                  [
                     1537847400,
                     74.25
                  ],
                  [
                     1537847700,
                     82.6666666666667
                  ]
               ],
               "columns" : [
                  "time",
                  "mean"
               ]
            }
         ]
      }
   ]
}`))
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	go func() {
		err := http.Serve(listener, nil)
		if err != nil {
			panic(err)
		}
	}()

	return "http://" + listener.Addr().String()
}

func TestBasicParallel(t *testing.T) {
	s, err := NewServer(Config{
		TLSProxy:       true,
		InfluxAddress:  emulateInflux(),
		InfluxHost:     "Max Mustermann",
		InfluxUsername: "foo",
		InfluxPassword: "bar",
		GinMode:        gin.TestMode,
		Debug:          true,
	})
	if err != nil {
		panic(err)
	}

	t.Run("basicTests", func(t *testing.T) {
		t.Run("header", s.headerTest)
		t.Run("notFoundHandler", s.notFoundHandlerTest)
		t.Run("indexHandler", s.indexHandlerTest)
		t.Run("statusHandler", s.statusHandlerTest)
		t.Run("svgHandler", s.svgHandlerTest)
		t.Run("redirect", s.redirectTest)
		t.Run("images", s.imagesTest)
		t.Run("statics", s.staticsTest)
		t.Run("robots", s.robotsTest)
	})
}

func (s *Server) headerTest(t *testing.T) {
	for _, path := range []string{"/", "/status"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.True(t, strings.Contains(w.Body.String(), fmt.Sprintf("<style type=\"text/css\" rel=\"stylesheet\">%s</style>", s.css)))
	}
}

func (s *Server) notFoundHandlerTest(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/not-existing-sub-page", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "not found"))
}

func (s *Server) indexHandlerTest(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "mail@hashworks.net"))
}

func (s *Server) statusHandlerTest(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/status", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "status-svg"))
}

func (s *Server) svgHandlerTest(t *testing.T) {
	for _, dimension := range svgDimensions {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/status-%dx%d.svg", dimension[0], dimension[1]), nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
		assert.True(t, strings.HasPrefix(w.Body.String(), "<svg xmlns"))
		assert.True(t, strings.HasSuffix(w.Body.String(), "</svg>"))
	}
}

func (s *Server) redirectTest(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/favicon.ico", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, 301, w.Code)
	assert.Equal(t, "/img/favicon.ico", w.Header().Get("Location"))
}

func getFiles(path string) []string {
	var fileList []string
	err := filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return fileList
}

func (s *Server) imagesTest(t *testing.T) {
	fileList := getFiles("img")

	for _, image := range fileList {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/"+image, nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "image"))
	}
}

func (s *Server) staticsTest(t *testing.T) {
	fileList := getFiles("static")

	for _, static := range fileList {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/"+static, nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.NotEmpty(t, w.Body)
	}
}

func (s *Server) robotsTest(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/robots.txt", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "User-agent")
}

func TestNoInfluxConnection(t *testing.T) {
	s, err := NewServer(Config{
		InfluxAddress: "http://127.0.0.1:1",
		InfluxHost:    "Max Mustermann",
		GinMode:       gin.TestMode,
	})
	if err != nil {
		panic(err)
	}
	for _, dimension := range svgDimensions {
		path := fmt.Sprintf("/status-%dx%d.svg", dimension[0], dimension[1])
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadGateway, w.Code)
		assert.Contains(t, w.Body.String(), "mail@hashworks.net")
	}
}

func TestNoDebugCSS(t *testing.T) {
	s, err := NewServer(Config{
		InfluxAddress: "http://127.0.0.1:1",
		InfluxHost:    "Max Mustermann",
		GinMode:       gin.TestMode,
		Debug:         false,
	})
	if err != nil {
		panic(err)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "mail@hashworks.net")
	assert.True(t, strings.Contains(w.Body.String(), fmt.Sprintf("<style type=\"text/css\" rel=\"stylesheet\">%s</style>", s.css)))
}

func TestConfigError(t *testing.T) {
	_, err := NewServer(Config{
		InfluxHost: "",
	})
	assert.EqualErrorf(t, err, "Influx host cannot be empty.", "")

	_, err = NewServer(Config{
		InfluxHost:    "Max Mustermann",
		InfluxAddress: "",
	})
	assert.EqualErrorf(t, err, "Influx address cannot be empty.", "")

	_, err = NewServer(Config{
		InfluxHost:    "Max Mustermann",
		InfluxAddress: "127.0.0.1:80",
	})
	assert.EqualErrorf(t, err, "Influx address must be a valid URI.", "")
}

func TestWrongHost(t *testing.T) {
	s, err := NewServer(Config{
		Domain:        "test.example.de",
		InfluxAddress: "http://127.0.0.1:1",
		InfluxHost:    "Max Mustermann",
		TLSProxy:      true,
		GinMode:       gin.TestMode,
		Debug:         true,
	})
	if err != nil {
		panic(err)
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bad host name")
}
