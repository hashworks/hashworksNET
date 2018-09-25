package server

import (
	"crypto/sha256"
	"encoding/base64"
	"github.com/ekyoung/gin-nice-recovery"
	"github.com/unrolled/secure"
	"time"

	// gin/logger.go might report undefined: isatty.IsCygwinTerminal
	// Fix: go get -u github.com/mattn/go-isatty
	"github.com/elazarl/go-bindata-assetfs"
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
	TLS            bool
	TLSProxy       bool
	GZIPExtension  bool
	CacheDir       string
	Debug          bool
	Domain         string
	EMail          string
	RedditURL      string
	SteamURL       string
	GitHubURL      string
	InfluxHost     string
	InfluxAddress  string
	InfluxUsername string
	InfluxPassword string
}

func NewServer(config Config) Server {
	gin.SetMode(config.GinMode)

	cssBytes := MustAsset("css/main.css")
	cssSha256 := sha256.Sum256(cssBytes)

	s := Server{
		Router:    gin.Default(),
		store:     persistence.NewInMemoryStore(time.Minute),
		css:       template.CSS(cssBytes),
		cssSha256: base64.StdEncoding.EncodeToString(cssSha256[:]),
		config:    config,
	}

	s.Router.Use(nice.Recovery(s.recoveryHandler))

	s.Router.Use(s.secureHandler(secure.New(s.getSecureOptions())))
	s.Router.Use(s.preHandler())
	if config.GZIPExtension {
		s.Router.Use(gzip.Gzip(gzip.DefaultCompression))
	}

	s.loadTemplates()

	s.Router.StaticFS("/static", &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo, Prefix: "static"})
	s.Router.StaticFS("/img", &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo, Prefix: "img"})

	s.Router.GET("/robots.txt", func(c *gin.Context) {
		c.String(http.StatusOK, "User-agent: *\nDisallow: /status\nDisallow: /status.svg")
	})

	s.Router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/img/favicon.ico")
	})

	s.Router.GET("/", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerIndex))
	s.Router.GET("/status", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerStatus))
	s.Router.GET("/status-1940x1060.svg", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerStatusSVG1940x1060))
	s.Router.GET("/status-1700x700.svg", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerStatusSVG1700x700))
	s.Router.GET("/status-1380x520.svg", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerStatusSVG1380x520))
	s.Router.GET("/status-1145x385.svg", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerStatusSVG1145x385))
	s.Router.GET("/status-780x385.svg", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerStatusSVG780x385))
	s.Router.GET("/status-500x335.svg", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerStatusSVG500x335))
	s.Router.GET("/status-400x225.svg", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerStatusSVG400x225))
	s.Router.GET("/status-200x115.svg", s.cacheHandler(true, false, s.store, 10*time.Minute, s.handlerStatusSVG200x115))

	s.Router.NoRoute(s.cacheHandler(true, false, s.store, 10*time.Minute, func(c *gin.Context) {
		c.Header("Cache-Control", "max-age=600")
		c.HTML(http.StatusNotFound, "error404", gin.H{
			"Title": "404",
		})
	}))

	return s
}
