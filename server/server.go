package server

import (
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"fmt"
	"io/fs"
	"time"

	_ "embed"

	"html/template"
	"net/http"

	nice "github.com/ekyoung/gin-nice-recovery"

	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// Server passes stuff around. Like database connections etc
type Server struct {
	Router    *gin.Engine
	store     *persistence.InMemoryStore
	css       template.CSS
	chartCSS  string
	cssSha256 []string
	config    Config
	startTime time.Time
}

type Config struct {
	Version       string
	BuildDate     string
	GinMode       string
	TLSProxy      bool
	GZIPExtension bool
	Debug         bool
	Domain        string
	TrustedProxy  string
	StaticContent embed.FS
}

func NewServer(config Config) (Server, error) {
	gin.SetMode(config.GinMode)

	css, err := config.StaticContent.ReadFile("css/main.css")
	if err != nil {
		panic(err)
	}

	chartCSS, err := config.StaticContent.ReadFile("css/chart.css")
	if err != nil {
		panic(err)
	}

	s := Server{
		Router:    gin.Default(),
		store:     persistence.NewInMemoryStore(time.Minute),
		css:       template.CSS(css),
		chartCSS:  string(chartCSS),
		config:    config,
		startTime: time.Now(),
	}

	err = s.Router.SetTrustedProxies([]string{config.TrustedProxy})
	if err != nil {
		panic(err)
	}

	cssSha256 := sha256.Sum256(css)
	chartCSSSha256 := sha256.Sum256(chartCSS)
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

	cssRoot, err := fs.Sub(s.config.StaticContent, "css")
	if err != nil {
		panic(err)
	}

	imgRoot, err := fs.Sub(s.config.StaticContent, "img")
	if err != nil {
		panic(err)
	}

	s.Router.StaticFS("/css", http.FS(cssRoot))
	s.Router.StaticFS("/img", http.FS(imgRoot))

	s.Router.GET("/robots.txt", func(c *gin.Context) {
		c.String(http.StatusOK, "User-agent: *\nDisallow: /status\nDisallow: /status-*.svg")
	})

	s.Router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/img/favicon.ico")
	})

	s.Router.GET("/", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerIndex))
	s.Router.GET("/status", s.cacheHandler(true, false, s.store, time.Minute, s.handlerStatus))

	for _, node := range [][2]string{{"hive", "hive.hashworks.net"}, {"helios", "helios.kromlinger.eu"}} {
		for _, dimension := range svgLoadDimensions {
			s.Router.GET(fmt.Sprintf("/load-%s-%dx%d.svg", node[0], dimension[0], dimension[1]), s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerLoadSVG(node[1], dimension[0], dimension[1])))
		}
	}

	s.Router.NoRoute(s.cacheHandler(true, false, s.store, 10*time.Minute, func(c *gin.Context) {
		c.Header("Cache-Control", "max-age=600")
		c.HTML(http.StatusNotFound, "error404", gin.H{
			"Title": "404",
		})
	}))

	return s, nil
}
