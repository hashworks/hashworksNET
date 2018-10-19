package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func (s Server) handlerIndex(c *gin.Context) {
	pageStartTime := time.Now()

	c.Header("Cache-Control", "max-age=600")
	c.Header("Last-Modified", s.startTime.Format(time.RFC1123))
	c.HTML(http.StatusOK, "index", gin.H{
		"ContactTab":    true,
		"Description":   "Contact information.",
		"PageStartTime": pageStartTime,
	})
}
