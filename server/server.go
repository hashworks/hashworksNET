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
	cssSha256 string
	config    Config
}

type Config struct {
	Version        string
	GinMode        string
	TLSProxy       bool
	GZIPExtension  bool
	Debug          bool
	Domain         string
	InfluxHost     string
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

	s := Server{
		Router:    gin.Default(),
		store:     persistence.NewInMemoryStore(time.Minute),
		css:       template.CSS(cssBytes),
		cssSha256: base64.StdEncoding.EncodeToString(cssSha256[:]),
		config:    config,
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
	s.Router.GET("/status", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerStatus))
	for _, dimension := range svgDimensions {
		s.Router.GET(fmt.Sprintf("/status-%dx%d.svg", dimension[0], dimension[1]), s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerStatusSVG(dimension[0], dimension[1])))

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
	if c.InfluxHost == "" {
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
