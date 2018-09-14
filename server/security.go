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
		Email:      "mail@hashworks.net",
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.domain),
		Cache:      autocert.DirCache(s.cacheDir),
	}

	httpServer := &http.Server{
		Addr: address,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,

				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, // Old Safaris (<= Safari 8 / OS X 10.10)
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

func secureHandler(secureMiddleware *secure.Secure) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := secureMiddleware.Process(c.Writer, c.Request)

		if err != nil {
			recoveryHandler(c, err)
			return
		}

		// Avoid header rewrite if response is a redirection.
		if status := c.Writer.Status(); status > 300 && status < 399 {
			c.Abort()
		}
	}
}

func (s Server) getSecureOptions() secure.Options {
	options := secure.Options{
		SSLRedirect:          s.tls,
		STSSeconds:           315360000,
		STSIncludeSubdomains: true,
		STSPreload:           true,
		ForceSTSHeader:       false,
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
		ContentSecurityPolicy: s.getCSP(!s.debug),
	}

	if s.domain != "" {
		options.AllowedHosts = []string{s.domain}
		options.SSLHost = s.domain
	}

	return options
}

func (s Server) getCSP(safeCSS bool) string {
	styleSrc := "'unsafe-inline'"
	if safeCSS {
		styleSrc = fmt.Sprintf("'sha256-%s'", s.cssSha256)
	}
	upgradeInSecureRequests := ""
	if s.tls {
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
