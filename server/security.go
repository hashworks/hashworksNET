package server

import (
	"crypto/tls"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"net/http"
)

func (s Server) RunTLS(address string) error {
	m := autocert.Manager{
		Email:      s.config.EMail,
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.config.Domain),
		Cache:      autocert.DirCache(s.config.CacheDir),
	}

	httpServer := &http.Server{
		Addr: address,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
			GetCertificate: m.GetCertificate,
			NextProtos: []string{
				"h2", "http/1.1",
				acme.ALPNProto,
			},
		},
		Handler: s.Router,
	}

	return httpServer.ListenAndServeTLS("", "")
}

func (s Server) secureHandler(secureMiddleware *secure.Secure) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := secureMiddleware.Process(c.Writer, c.Request)

		if err != nil {
			s.recoveryHandler(c, err)
			return
		}
	}
}

func (s Server) getSecureOptions() secure.Options {
	options := secure.Options{
		SSLRedirect:          s.config.TLS,
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
		if s.config.TLS || s.config.TLSProxy {
			options.SSLHost = s.config.Domain
		}
	}

	return options
}

func (s Server) getCSP(safeCSS bool) string {
	styleSrc := "'unsafe-inline'"
	if safeCSS {
		styleSrc = fmt.Sprintf("'sha256-%s'", s.cssSha256)
	}
	upgradeInSecureRequests := ""
	if s.config.TLS || s.config.TLSProxy {
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
