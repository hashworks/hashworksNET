package server

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cache"
	"github.com/gin-contrib/cache/persistence"
	"github.com/gin-gonic/gin"
)

func (s Server) recoveryHandlerStatus(statusCode int, c *gin.Context, err interface{}) {
	timeString := time.Now().Format(time.RFC3339)
	var message string

	switch err.(type) {
	case error:
		message = err.(error).Error()
	default:
		message = "Unknown"
	}

	log.Printf("%s - Error: %s", timeString, message)

	if !s.config.Debug {
		message = "There was an error, please report this to mail@hashworks.net."
	}

	c.AbortWithStatusJSON(statusCode, map[string]interface{}{
		"time":   timeString,
		"error":  message,
		"status": statusCode,
	})
}

func (s Server) recoveryHandler(c *gin.Context, err interface{}) {
	s.recoveryHandlerStatus(http.StatusInternalServerError, c, err)
}

func (s Server) preHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.config.Debug {
			// Reload templates, they might have changed
			s.loadTemplates()
		}

		if strings.HasPrefix(c.Request.URL.Path, "/img/") {
			c.Header("Cache-Control", "max-age=31540000")
			c.Header("Last-Modified", s.startTime.Format(time.RFC1123))
		} else if strings.HasPrefix(c.Request.URL.Path, "/css/") {
			c.Header("Content-Type", "text/css")
			c.Header("Cache-Control", "max-age=604800")
			c.Header("Last-Modified", s.startTime.Format(time.RFC1123))
		} else if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Header("Cache-Control", "max-age=604800")
			c.Header("Last-Modified", s.startTime.Format(time.RFC1123))
			c.Header("Content-Description", "File Transfer")
			c.Header("Content-Disposition", "attachment")
			c.Header("Content-Type", "application/octet-stream")
			c.Header("Content-Transfer-Encoding", "binary")
		}
	}
}

func (s Server) cacheHandler(withoutQuery bool, withoutHeader bool, store persistence.CacheStore, expire time.Duration, handle gin.HandlerFunc) gin.HandlerFunc {
	// No cache in debug mode
	if s.config.Debug {
		return handle
	}
	if withoutQuery {
		return cache.CachePageWithoutQuery(store, expire, handle)
	}
	return cache.CachePage(store, expire, handle)
}
