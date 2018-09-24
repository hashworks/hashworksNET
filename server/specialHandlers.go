package server

import (
	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
	"log"
	"strings"
	"time"
)

func recoveryHandler(c *gin.Context, err interface{}) {
	log.Printf("Error: %s", err)
	c.AbortWithStatus(500)
	// https://github.com/gin-contrib/cache/issues/35
	//c.String(http.StatusInternalServerError, "There was an internal server error, please report this to mail@hashworks.net.")
}

func (s Server) preHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.config.Debug {
			// Reload templates, they might have changed
			s.loadTemplates()
		}

		if strings.HasPrefix(c.Request.RequestURI, "/static/") {
			c.Header("Cache-Control", "max-age=604800")
			c.Header("Content-Description", "File Transfer")
			c.Header("Content-Disposition", "attachment")
			c.Header("Content-Type", "application/octet-stream")
			c.Header("Content-Transfer-Encoding", "binary")
		} else if strings.HasPrefix(c.Request.RequestURI, "/img/") {
			c.Header("Cache-Control", "max-age=31540000")
		}
	}
}

func (s Server) cacheHandler(store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	// No cache in debug mode
	if s.config.Debug {
		return handle
	}
	return cache.CachePage(store, expire, handle)
}
