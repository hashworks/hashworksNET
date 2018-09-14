package server

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (s Server) handlerIndex(c *gin.Context) {
	c.Header("Cache-Control", "max-age=600")
	c.HTML(http.StatusOK, "index", gin.H{
		"ContactTab": true,
	})
}
