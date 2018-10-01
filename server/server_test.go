package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var influxAddressData,
	influxAddressNotEnoughData,
	influxAddressNoData,
	influxAddressUnauthorized string

func TestMain(m *testing.M) {
	emulateInflux()
	os.Exit(m.Run())
}

func emulateInflux() {
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
	http.HandleFunc("/notEnoughData/query", func(w http.ResponseWriter, _ *http.Request) {
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
	http.HandleFunc("/noData/query", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
{
   "results" : [
      {
         "statement_id" : 0,
         "series" : [
            
         ]
      }
   ]
}`))
	})
	http.HandleFunc("/unauthorized/query", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
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

	influxAddressData = "http://" + listener.Addr().String()
	influxAddressNotEnoughData = influxAddressData + "/notEnoughData"
	influxAddressNoData = influxAddressData + "/noData"
	influxAddressUnauthorized = influxAddressData + "/unauthorized"
}

func TestBasicParallel(t *testing.T) {
	s, err := NewServer(Config{
		TLSProxy:       true,
		InfluxAddress:  influxAddressData,
		InfluxHost:     "Max Mustermann",
		InfluxUsername: "foo",
		InfluxPassword: "bar",
		GinMode:        gin.TestMode,
		Debug:          true,
		GZIPExtension:  true,
	})
	assert.NoError(t, err)

	t.Run("basicTests", func(t *testing.T) {
		t.Run("header", s.headerTest)
		t.Run("notFoundHandler", s.notFoundHandlerTest)
		t.Run("indexHandler", s.indexHandlerTest)
		t.Run("statusHandler", s.statusHandlerTest)
		t.Run("svgHandler", s.svgHandlerTest)
		t.Run("redirect", s.redirectTest)
		t.Run("images", s.imagesTest)
		t.Run("statics", s.staticsTest)
		t.Run("notFound", s.notFoundTest)
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

func (s *Server) imagesTest(t *testing.T) {
	for _, image := range []string{"favicon.ico", "favicon-16x16.png", "favicon-32x32.png", "favicon-96x96.png", "favicon-194x194.png"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/img/"+image, nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "image"))
	}
}

func (s *Server) staticsTest(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/static/pgp_public_key.asc", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "File Transfer", w.Header().Get("Content-Description"))
	assert.Equal(t, "attachment", w.Header().Get("Content-Disposition"))
	assert.Equal(t, "application/octet-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "binary", w.Header().Get("Content-Transfer-Encoding"))
	assert.NotEmpty(t, w.Body)
}

func (s *Server) notFoundTest(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/static/notExistingFile", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
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
	assert.NoError(t, err)
	for _, dimension := range svgDimensions {
		path := fmt.Sprintf("/status-%dx%d.svg", dimension[0], dimension[1])
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadGateway, w.Code)
		assert.Contains(t, w.Body.String(), "mail@hashworks.net")
	}
}

func TestInfluxNotEnoughData(t *testing.T) {
	s, err := NewServer(Config{
		InfluxAddress: influxAddressNotEnoughData,
		InfluxHost:    "Max Mustermann",
		GinMode:       gin.TestMode,
	})
	assert.NoError(t, err)
	for _, dimension := range svgDimensions {
		path := fmt.Sprintf("/status-%dx%d.svg", dimension[0], dimension[1])
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
		assert.True(t, strings.HasPrefix(w.Body.String(), "<svg xmlns"))
		assert.True(t, strings.HasSuffix(w.Body.String(), "</svg>"))
		assert.Contains(t, w.Body.String(), "heart-rate")
	}
}

func TestInfluxNoData(t *testing.T) {
	s, err := NewServer(Config{
		InfluxAddress: influxAddressNoData,
		InfluxHost:    "Max Mustermann",
		GinMode:       gin.TestMode,
	})
	assert.NoError(t, err)
	for _, dimension := range svgDimensions {
		path := fmt.Sprintf("/status-%dx%d.svg", dimension[0], dimension[1])
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", path, nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "mail@hashworks.net")
	}
}

func TestInfluxUnauthorized(t *testing.T) {
	s, err := NewServer(Config{
		InfluxAddress: influxAddressUnauthorized,
		InfluxHost:    "Max Mustermann",
		GinMode:       gin.TestMode,
	})
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bad host name")
}
