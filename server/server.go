package server

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/ekyoung/gin-nice-recovery"
	"github.com/hashworks/hashworksNET/server/bindata"
	"regexp"
	"time"

	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"html/template"
	"net/http"
)

// I'm using this struct to pass stuff around. Like database connections etc
type Server struct {
	Router    *gin.Engine
	store     *persistence.InMemoryStore
	css       template.CSS
	chartCSS  string
	cssSha256 []string
	config    Config
}

type Config struct {
	Version        string
	BuildDate      string
	GinMode        string
	TLSProxy       bool
	GZIPExtension  bool
	Debug          bool
	Domain         string
	InfluxBPMHost  string
	InfluxLoadHost string
	InfluxAddress  string
	InfluxUsername string
	InfluxPassword string
}

func NewServer(config Config) (Server, error) {
	err := testConfig(&config)
	if err != nil {
		return Server{}, err
	}
	gin.SetMode(config.GinMode)

	cssBytes := bindata.FileSassMainCSS
	cssSha256 := sha256.Sum256(cssBytes)

	chartCSSBytes := bindata.FileSassChartCSS
	chartCSSSha256 := sha256.Sum256(chartCSSBytes)

	s := Server{
		Router:   gin.Default(),
		store:    persistence.NewInMemoryStore(time.Minute),
		css:      template.CSS(cssBytes),
		chartCSS: string(chartCSSBytes),
		config:   config,
	}

	s.cssSha256 = []string{
		base64.StdEncoding.EncodeToString(cssSha256[:]),
		base64.StdEncoding.EncodeToString(chartCSSSha256[:]),
	}

	s.Router.Use(nice.Recovery(s.recoveryHandler))

	s.Router.Use(s.secureHandler(s.getSecureMiddleware()))
	s.Router.Use(s.preHandler())
	if config.GZIPExtension {
		s.Router.Use(gzip.Gzip(gzip.DefaultCompression))
	}

	s.loadTemplates()

	s.Router.StaticFS("/static", &prefixHTTPFS{prefix: "static"})
	s.Router.StaticFS("/img", &prefixHTTPFS{prefix: "img"})

	s.Router.GET("/robots.txt", func(c *gin.Context) {
		c.String(http.StatusOK, "User-agent: *\nDisallow: /status\nDisallow: /status-*.svg")
	})

	s.Router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/img/favicon.ico")
	})

	s.Router.GET("/", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerIndex))
	s.Router.GET("/status", s.cacheHandler(true, false, s.store, time.Minute, s.handlerStatus))
	for _, dimension := range svgBPMDimensions {
		s.Router.GET(fmt.Sprintf("/bpm-%dx%d.svg", dimension[0], dimension[1]), s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerBPMSVG(dimension[0], dimension[1])))

	}
	for _, dimension := range svgLoadDimensions {
		s.Router.GET(fmt.Sprintf("/load-%dx%d.svg", dimension[0], dimension[1]), s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerLoadSVG(dimension[0], dimension[1])))

	}
	s.Router.NoRoute(s.cacheHandler(true, false, s.store, 10*time.Minute, func(c *gin.Context) {
		c.Header("Cache-Control", "max-age=600")
		c.HTML(http.StatusNotFound, "error404", gin.H{
			"Title": "404",
		})
	}))

	return s, nil
}

func testConfig(c *Config) error {
	if c.InfluxBPMHost == "" {
		return errors.New("Influx host cannot be empty.")
	}
	if c.InfluxAddress == "" {
		return errors.New("Influx address cannot be empty.")
	} else {
		if regexp.MustCompile(`^http(?:s)?:\/\/[\S^:]+(?::[0-9]+)?(?:\S+)?$`).FindStringIndex(c.InfluxAddress) == nil {
			return errors.New("Influx address must be a valid URI.")
		}
	}
	return nil
}
