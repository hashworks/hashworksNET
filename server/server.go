package server

import (
	"github.com/ekyoung/gin-nice-recovery"
	"log"
	"strings"
	"time"

	// gin/logger.go might report undefined: isatty.IsCygwinTerminal
	// Fix: go get -u github.com/mattn/go-isatty
	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"html/template"
	"net/http"
)

// I'm using this struct to pass stuff around. Like database connnection
type Server struct {
	Router *gin.Engine
	Store  *persistence.InMemoryStore
}

func NewServer() Server {
	var err error
	//gin.SetMode(gin.ReleaseMode)

	s := Server{
		Router: gin.Default(),
		Store:  persistence.NewInMemoryStore(time.Minute),
	}

	s.Router.Use(nice.Recovery(recoveryHandler))

	// Load template file names from Asset
	templateNames, err := AssetDir("templates")
	if err != nil {
		panic(err)
	}

	// Create a base template where we add the template functions
	tmpl := template.New("")
	tmpl.Funcs(s.templateFunctionMap())

	// Iterate trough template files, load them into multitemplate
	multiT := multitemplate.New()
	for _, templateName := range templateNames {
		index := strings.Index(templateName, ".")
		basename := templateName[:index]
		tmpl := tmpl.New(basename)
		tmpl, err := tmpl.Parse(string(MustAsset("templates/" + templateName)))
		if err != nil {
			panic(err)
		}
		multiT.Add(basename, tmpl)
		log.Printf("Loaded templates/%s as %s\n", templateName, basename)
	}
	// multitemplate is our new HTML renderer
	s.Router.HTMLRender = multiT

	s.Router.StaticFS("/img", &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo, Prefix: "img"})
	s.Router.StaticFS("/static", &assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, AssetInfo: AssetInfo, Prefix: "static"})

	s.Router.GET("/robots.txt", func(c *gin.Context) {
		c.String(http.StatusOK, "User-agent: *\nDisallow: /status\nDisallow: /status.svg")
	})

	s.Router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/img/favicon.ico")
	})

	s.Router.GET("/", cache.CachePage(s.Store, 10*time.Minute, s.index))
	s.Router.GET("/status", cache.CachePage(s.Store, 10*time.Minute, s.status))
	s.Router.GET("/status.svg", cache.CachePage(s.Store, 10*time.Minute, s.statusSVG))

	s.Router.NoRoute(cache.CachePage(s.Store, 10*time.Minute, func(c *gin.Context) {
		c.Header("Cache-Control", "max-age=600")
		c.HTML(http.StatusNotFound, "error404", gin.H{
			"Title": "404",
		})
	}))

	return s
}

func recoveryHandler(c *gin.Context, err interface{}) {
	log.Printf("Error: %s", err)
	c.String(http.StatusInternalServerError, "There was an internal server error, please report this to mail@hashworks.net.")
}
