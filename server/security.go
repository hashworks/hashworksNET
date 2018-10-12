package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure"
	"net/http"
	"strings"
)

func (s Server) getSecureMiddleware() *secure.Secure {
	secureMiddleware := secure.New(s.getSecureOptions())
	secureMiddleware.SetBadHostHandler(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	return secureMiddleware
}

func (s Server) secureHandler(secureMiddleware *secure.Secure) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := secureMiddleware.Process(c.Writer, c.Request)

		if err != nil {
			s.recoveryHandlerStatus(http.StatusBadRequest, c, err)
			return
		}
	}
}

func (s Server) getSecureOptions() secure.Options {
	options := secure.Options{
		STSSeconds:           315360000,
		STSIncludeSubdomains: true,
		STSPreload:           true,
		ForceSTSHeader:       s.config.TLSProxy,
		FrameDeny:            true,
		ContentTypeNosniff:   true,
		BrowserXssFilter:     true,
		ReferrerPolicy:       "no-referrer",
		FeaturePolicy: "geolocation 'none';" +
			"midi 'none';" +
			"notifications 'none';" +
			"push 'none';" +
			"sync-xhr 'none';" +
			"microphone 'none';" +
			"camera 'none';" +
			"magnetometer 'none';" +
			"gyroscope 'none';" +
			"speaker 'none';" +
			"vibrate 'none';" +
			"fullscreen 'none';" +
			"payment 'none'",
		ContentSecurityPolicy: s.getCSP(!s.config.Debug),
	}

	if s.config.Domain != "" {
		options.AllowedHosts = []string{s.config.Domain}
		if s.config.TLSProxy {
			options.SSLHost = s.config.Domain
		}
	}

	return options
}

func (s Server) getCSP(safeCSS bool) string {
	var styleSrc string
	if safeCSS {
		styleSrc = "'sha256-" + strings.Join(s.cssSha256, "' 'sha256-") + "'"
	} else {
		styleSrc = "'unsafe-inline'"
	}
	upgradeInSecureRequests := ""
	if s.config.TLSProxy {
		upgradeInSecureRequests = "upgrade-insecure-requests; "
	}
	return fmt.Sprintf("%s"+
		"default-src 'none';"+
		"script-src 'none';"+
		"style-src %s;"+
		"img-src 'self' data:;"+
		"connect-src 'none';"+
		"font-src 'none';"+
		"object-src 'none';"+
		"media-src 'none';"+
		"worker-src 'none';"+
		"frame-src 'none';"+
		"form-action 'none';"+
		"frame-ancestors 'none';"+
		"base-uri 'self'", upgradeInSecureRequests, styleSrc)
}
