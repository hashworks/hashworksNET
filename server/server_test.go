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

var influxAddressBPMData,
	influxAddressBPMDataErrorHigh,
	influxAddressBPMDataWarningHigh,
	influxAddressBPMDataWarningLow,
	influxAddressBPMDataErrorLow,
	influxAddressBPMNotEnoughData,

	influxAddressLoadData,
	influxAddressLoadDataWarning,
	influxAddressLoadDataError,

	influxAddressStatusData,

	influxAddressNoData,
	influxAddressFailure,
	influxAddressUnauthorized string

func TestMain(m *testing.M) {
	emulateInflux()
	os.Exit(m.Run())
}

func emulateInflux() {
	http.HandleFunc("/bpmData/query", func(w http.ResponseWriter, _ *http.Request) {
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
	for name, avg := range map[string]int{"ErrorHigh": 130, "WarningHigh": 100, "WarningLow": 30, "ErrorLow": 20} {
		name, avg := name, avg // Copy value for closure
		http.HandleFunc(fmt.Sprintf("/bpmData%s/query", name), func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf(`{
   "results" : [
      {
         "statement_id" : 0,
         "series" : [
            {
               "name" : "bpm",
               "values" : [
                  [
                     1537844700,
                     %d
                  ],
                  [
                     1537845000,
                     %d
                  ],
                  [
                     1537845300,
                     %d
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
}`, avg, avg, avg)))
		})
	}
	http.HandleFunc("/bpmNotEnoughData/query", func(w http.ResponseWriter, _ *http.Request) {
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

	http.HandleFunc("/loadData/query", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
  "results": [
    {
      "statement_id": 0,
      "series": [
        {
          "name": "system",
          "columns": [
            "time",
            "load1"
          ],
          "values": [
            [
              1439837470,
              0.29
            ],
            [
              1439837480,
              0.25
            ],
            [
              1439837490,
              0.44
            ],
            [
              1439837500,
              0.52
            ],
            [
              1439837510,
              0.52
            ],
            [
              1439837520,
              0.44
            ],
            [
              1439837530,
              0.37
            ]
          ]
        }
      ]
    }
  ]
}`))
		if err != nil {
			panic(err)
		}
	})
	for name, avg := range map[string]int{"Error": 8, "Warning": 4} {
		name, avg := name, avg // Copy value for closure
		http.HandleFunc(fmt.Sprintf("/loadData%s/query", name), func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(fmt.Sprintf(`{
  "results": [
    {
      "statement_id": 0,
      "series": [
        {
          "name": "system",
          "columns": [
            "time",
            "load1"
          ],
          "values": [
            [
              1439837470,
              %d
            ],
            [
              1439837480,
              %d
            ],
            [
              1439837490,
              %d
            ]
          ]
        }
      ]
    }
  ]
}`, avg, avg, avg)))
			if err != nil {
				panic(err)
			}
		})
	}

	http.HandleFunc("/statusData/query", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`
{
  "results": [
    {
      "statement_id": 0,
      "series": [
        {
          "name": "net_response",
          "columns": [
            "time",
            "last_response_time",
            "last_result_code",
            "last_result_type"
          ],
          "values": [
            [
              "2018-10-18T05:06:07.287339614Z",
              30.022948892,
              1,
              "timeout"
            ]
          ]
        }
      ]
    },
    {
      "statement_id": 1,
      "series": [
        {
          "name": "net_response",
          "columns": [
            "time",
            "last_response_time",
            "last_result_code",
            "last_result_type"
          ],
          "values": [
            [
              "2018-10-18T05:06:07.287339614Z",
              0.022948892,
              0,
              "success"
            ]
          ]
        }
      ]
    },
    {
      "statement_id": 2,
      "series": [
        {
          "name": "system",
          "columns": [
            "time",
            "last",
            "last_1",
            "last_2"
          ],
          "values": [
            [
              "2018-10-18T05:11:30.280230906Z",
              2.2,
              5.01,
              9.34
            ]
          ]
        }
      ]
    }
  ]
}`))
		if err != nil {
			panic(err)
		}
	})

	http.HandleFunc("/failure/query", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`
{
   "results" : []
}`))
		if err != nil {
			panic(err)
		}
	})
	http.HandleFunc("/noData/query", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`
{
   "results" : [
      {
         "statement_id" : 0
      }
   ]
}`))
		if err != nil {
			panic(err)
		}
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

	influxAddress := "http://" + listener.Addr().String()
	influxAddressBPMData = influxAddress + "/bpmData"
	influxAddressBPMDataErrorHigh = influxAddress + "/bpmDataErrorHigh"
	influxAddressBPMDataWarningHigh = influxAddress + "/bpmDataWarningHigh"
	influxAddressBPMDataWarningLow = influxAddress + "/bpmDataWarningLow"
	influxAddressBPMDataErrorLow = influxAddress + "/bpmDataErrorLow"
	influxAddressBPMNotEnoughData = influxAddress + "/bpmNotEnoughData"

	influxAddressLoadData = influxAddress + "/loadData"
	influxAddressLoadDataWarning = influxAddress + "/loadDataWarning"
	influxAddressLoadDataError = influxAddress + "/loadDataError"

	influxAddressStatusData = influxAddress + "/statusData"

	influxAddressNoData = influxAddress + "/noData"
	influxAddressFailure = influxAddress + "/failure"
	influxAddressUnauthorized = influxAddress + "/unauthorized"
}

func TestBasicParallel(t *testing.T) {
	s, err := NewServer(Config{
		TLSProxy:       true,
		InfluxAddress:  influxAddressStatusData,
		InfluxLoadHost: "serverhostname",
		InfluxBPMHost:  "Max Mustermann",
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
		t.Run("redirect", s.redirectTest)
		t.Run("images", s.imagesTest)
		t.Run("css", s.cssTest)
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
		assert.True(t, strings.Contains(w.Body.String(), fmt.Sprintf(`<style rel="stylesheet" type="text/css">%s</style>`, s.css)))
		if path == "/status" {
			assert.True(t, strings.Contains(w.Body.String(), `<link rel="stylesheet" type="text/css" href="/css/status.css">`))
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

func (s *Server) statusHandlerTest(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/status", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	body := w.Body.String()
	assert.True(t, strings.Contains(body, s.config.InfluxLoadHost))
	assert.True(t, strings.Contains(body, `<td class="ok">2.20</td>`))
	assert.True(t, strings.Contains(body, `<td class="warning">5.01</td>`))
	assert.True(t, strings.Contains(body, `<td class="error">9.34</td>`))
	assert.True(t, strings.Contains(body, `<div class="status error">Timeout.</div>`))
	assert.True(t, strings.Contains(body, `<div class="status ok">Online. 0.02s latency.</div>`))
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

func TestBPMSVG(t *testing.T) {
	s, err := NewServer(Config{
		TLSProxy:       true,
		InfluxAddress:  influxAddressBPMData,
		InfluxBPMHost:  "Max Mustermann",
		InfluxUsername: "foo",
		InfluxPassword: "bar",
		GinMode:        gin.TestMode,
		GZIPExtension:  true,
		Debug:          true,
	})
	assert.NoError(t, err)
	for _, dimension := range svgBPMDimensions {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/bpm-%dx%d.svg", dimension[0], dimension[1]), nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
		assert.True(t, strings.HasPrefix(w.Body.String(), "<svg xmlns"))
		assert.True(t, strings.HasSuffix(w.Body.String(), "</svg>"))
	}
}

func TestBPMWarningErrorSVG(t *testing.T) {
	for address, status := range map[string]string{
		influxAddressBPMDataErrorHigh:   "error",
		influxAddressBPMDataWarningHigh: "warning",
		influxAddressBPMDataWarningLow:  "warning",
		influxAddressBPMDataErrorLow:    "error",
	} {
		s, err := NewServer(Config{
			TLSProxy:       true,
			InfluxAddress:  address,
			InfluxBPMHost:  "Max Mustermann",
			InfluxUsername: "foo",
			InfluxPassword: "bar",
			GinMode:        gin.TestMode,
			GZIPExtension:  true,
			Debug:          true,
		})
		assert.NoError(t, err)
		for _, dimension := range svgBPMDimensions {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/bpm-%dx%d.svg", dimension[0], dimension[1]), nil)
			s.Router.ServeHTTP(w, req)

			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
			assert.True(t, strings.HasPrefix(w.Body.String(), "<svg xmlns"))
			assert.True(t, strings.HasSuffix(w.Body.String(), "</svg>"))
			assert.True(t, strings.Contains(w.Body.String(), `class="series `+status))
		}
	}
}

func TestLoadSVG(t *testing.T) {
	s, err := NewServer(Config{
		TLSProxy:       true,
		InfluxAddress:  influxAddressLoadData,
		InfluxBPMHost:  "Max Mustermann",
		InfluxUsername: "foo",
		InfluxPassword: "bar",
		GinMode:        gin.TestMode,
		GZIPExtension:  true,
		Debug:          true,
	})
	assert.NoError(t, err)
	for _, dimension := range svgLoadDimensions {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/load-%dx%d.svg", dimension[0], dimension[1]), nil)
		s.Router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
		assert.True(t, strings.HasPrefix(w.Body.String(), "<svg xmlns"))
		assert.True(t, strings.HasSuffix(w.Body.String(), "</svg>"))
	}
}

func TestLoadWarningErrorSVG(t *testing.T) {
	for address, status := range map[string]string{
		influxAddressLoadDataError:   "error",
		influxAddressLoadDataWarning: "warning",
	} {
		s, err := NewServer(Config{
			TLSProxy:       true,
			InfluxAddress:  address,
			InfluxBPMHost:  "Max Mustermann",
			InfluxUsername: "foo",
			InfluxPassword: "bar",
			GinMode:        gin.TestMode,
			GZIPExtension:  true,
			Debug:          true,
		})
		assert.NoError(t, err)
		for _, dimension := range svgLoadDimensions {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", fmt.Sprintf("/load-%dx%d.svg", dimension[0], dimension[1]), nil)
			s.Router.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)
			assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
			assert.True(t, strings.HasPrefix(w.Body.String(), "<svg xmlns"))
			assert.True(t, strings.HasSuffix(w.Body.String(), "</svg>"))
			assert.True(t, strings.Contains(w.Body.String(), `class="series `+status))
		}
	}
}

func TestNoInfluxConnection(t *testing.T) {
	s, err := NewServer(Config{
		InfluxAddress: "http://127.0.0.1:1",
		InfluxBPMHost: "Max Mustermann",
		GinMode:       gin.TestMode,
	})
	assert.NoError(t, err)
	for svg, dimensions := range map[string][][]int{
		"bpm":  svgBPMDimensions,
		"load": svgLoadDimensions,
	} {
		for _, dimension := range dimensions {
			path := fmt.Sprintf("/%s-%dx%d.svg", svg, dimension[0], dimension[1])
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", path, nil)
			s.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadGateway, w.Code)
			assert.Contains(t, w.Body.String(), "mail@hashworks.net")
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/status", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "mail@hashworks.net")
}

func TestInfluxNotEnoughData(t *testing.T) {
	s, err := NewServer(Config{
		InfluxAddress: influxAddressBPMNotEnoughData,
		InfluxBPMHost: "Max Mustermann",
		GinMode:       gin.TestMode,
	})
	assert.NoError(t, err)
	for svg, dimensions := range map[string][][]int{
		"bpm":  svgBPMDimensions,
		"load": svgLoadDimensions,
	} {
		for _, dimension := range dimensions {
			path := fmt.Sprintf("/%s-%dx%d.svg", svg, dimension[0], dimension[1])
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", path, nil)
			s.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
			assert.True(t, strings.HasPrefix(w.Body.String(), "<svg xmlns"))
			assert.True(t, strings.HasSuffix(w.Body.String(), "</svg>"))
			assert.Contains(t, w.Body.String(), "Not enough")
		}
	}
}

func TestInfluxNoData(t *testing.T) {
	s, err := NewServer(Config{
		InfluxAddress: influxAddressNoData,
		InfluxBPMHost: "Max Mustermann",
		GinMode:       gin.TestMode,
	})
	assert.NoError(t, err)
	for svg, dimensions := range map[string][][]int{
		"bpm":  svgBPMDimensions,
		"load": svgLoadDimensions,
	} {
		for _, dimension := range dimensions {
			path := fmt.Sprintf("/%s-%dx%d.svg", svg, dimension[0], dimension[1])
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", path, nil)
			s.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
			assert.True(t, strings.HasPrefix(w.Body.String(), "<svg xmlns"))
			assert.True(t, strings.HasSuffix(w.Body.String(), "</svg>"))
			assert.Contains(t, w.Body.String(), "Not enough")
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/status", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `<div class="status error">No data!</div>`)
}

func TestInfluxFailure(t *testing.T) {
	s, err := NewServer(Config{
		InfluxAddress: influxAddressFailure,
		InfluxBPMHost: "Max Mustermann",
		GinMode:       gin.TestMode,
	})
	assert.NoError(t, err)
	for svg, dimensions := range map[string][][]int{
		"bpm":  svgBPMDimensions,
		"load": svgLoadDimensions,
	} {
		for _, dimension := range dimensions {
			path := fmt.Sprintf("/%s-%dx%d.svg", svg, dimension[0], dimension[1])
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", path, nil)
			s.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusInternalServerError, w.Code)
			assert.Contains(t, w.Body.String(), "mail@hashworks.net")
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/status", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "mail@hashworks.net")
}

func TestInfluxUnauthorized(t *testing.T) {
	s, err := NewServer(Config{
		InfluxAddress: influxAddressUnauthorized,
		InfluxBPMHost: "Max Mustermann",
		GinMode:       gin.TestMode,
	})
	assert.NoError(t, err)
	for svg, dimensions := range map[string][][]int{
		"bpm":  svgBPMDimensions,
		"load": svgLoadDimensions,
	} {
		for _, dimension := range dimensions {
			path := fmt.Sprintf("/%s-%dx%d.svg", svg, dimension[0], dimension[1])
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", path, nil)
			s.Router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadGateway, w.Code)
			assert.Contains(t, w.Body.String(), "mail@hashworks.net")
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/status", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "mail@hashworks.net")
}

func TestNoDebugCSS(t *testing.T) {
	s, err := NewServer(Config{
		InfluxAddress: "http://127.0.0.1:1",
		InfluxBPMHost: "Max Mustermann",
		GinMode:       gin.TestMode,
		Debug:         false,
	})
	assert.NoError(t, err)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	s.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "mail@hashworks.net")
	assert.True(t, strings.Contains(w.Body.String(), fmt.Sprintf("<style rel=\"stylesheet\" type=\"text/css\">%s</style>", s.css)))
}

func TestConfigError(t *testing.T) {
	_, err := NewServer(Config{
		InfluxBPMHost: "",
	})
	assert.EqualErrorf(t, err, "Influx host cannot be empty.", "")

	_, err = NewServer(Config{
		InfluxBPMHost: "Max Mustermann",
		InfluxAddress: "",
	})
	assert.EqualErrorf(t, err, "Influx address cannot be empty.", "")

	_, err = NewServer(Config{
		InfluxBPMHost: "Max Mustermann",
		InfluxAddress: "127.0.0.1:80",
	})
	assert.EqualErrorf(t, err, "Influx address must be a valid URI.", "")
}

func TestWrongHost(t *testing.T) {
	s, err := NewServer(Config{
		Domain:        "test.example.de",
		InfluxAddress: "http://127.0.0.1:1",
		InfluxBPMHost: "Max Mustermann",
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
