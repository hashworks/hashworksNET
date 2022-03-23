package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var influxAddressLoadData,
	influxAddressLoadDataWarning,
	influxAddressLoadDataError,

	influxAddressStatusData,

	influxAddressNoData,
	influxAddressFailure,
	influxAddressUnauthorized string

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestBasicParallel(t *testing.T) {
	s, err := NewServer(Config{
		TLSProxy:      true,
		GinMode:       gin.TestMode,
		Debug:         true,
		GZIPExtension: true,
	})
	assert.NoError(t, err)

	t.Run("basicTests", func(t *testing.T) {
		t.Run("header", s.headerTest)
		t.Run("notFoundHandler", s.notFoundHandlerTest)
		t.Run("indexHandler", s.indexHandlerTest)
		t.Run("redirect", s.redirectTest)
		t.Run("images", s.imagesTest)
		t.Run("css", s.cssTest)
		t.Run("robots", s.robotsTest)
	})
}

func (s *Server) headerTest(t *testing.T) {
	for _, path := range []string{"/", "/status"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.True(t, strings.Contains(w.Body.String(), fmt.Sprintf(`<style rel=stylesheet type="text/css">%s</style>`, s.css)))
		if path == "/status" {
			assert.True(t, strings.Contains(w.Body.String(), `<link rel=stylesheet type="text/css" href="/css/status.css">`))
		}
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

func (s *Server) redirectTest(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/favicon.ico", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, 301, w.Code)
	assert.Equal(t, "/img/favicon.ico", w.Header().Get("Location"))
}

func (s *Server) imagesTest(t *testing.T) {
	for _, image := range []string{"favicon.ico", "favicon-16x16.png", "favicon-32x32.png", "favicon-96x96.png", "favicon-194x194.png"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/img/"+image, nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "image"))
	}
}

func (s *Server) cssTest(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/css/chart.css", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "text/css", w.Header().Get("Content-Type"))
}

func (s *Server) robotsTest(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/robots.txt", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "User-agent")
}

func TestNoDebugCSS(t *testing.T) {
	s, err := NewServer(Config{
		GinMode: gin.TestMode,
		Debug:   false,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "mail@hashworks.net")
	assert.True(t, strings.Contains(w.Body.String(), fmt.Sprintf("<style rel=stylesheet type=\"text/css\">%s</style>", s.css)))
}

func TestWrongHost(t *testing.T) {
	s, err := NewServer(Config{
		Domain:   "test.example.de",
		TLSProxy: true,
		GinMode:  gin.TestMode,
		Debug:    true,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bad host name")
}
