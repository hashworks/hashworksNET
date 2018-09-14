//go:generate make generate
package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hashworks/hashworksNET/server"
	"os"
	"path"
)

var (
	// Set the following uppercase three with -ldflags "-X main.VERSION=v1.2.3 [...]"
	VERSION      string = "unknown"
	BUILD_COMMIT string = "unknown"
	BUILD_DATE   string = "unknown"
	GIN_MODE     string = gin.DebugMode
	versionFlag  bool
	address      string
	port         int
	https        bool
	domain       string
	cacheDir     string
	debug        bool
)

func main() {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flagSet.BoolVar(&versionFlag, "version", false, "Displays the version and license information.")
	flagSet.StringVar(&address, "address", "", "The address to listen on.")
	flagSet.IntVar(&port, "port", 65432, "The port to listen on.")
	flagSet.BoolVar(&https, "https", false, "Provide HTTPS. Requires a domain.")
	flagSet.StringVar(&domain, "domain", "", "The domain required by HTTPS.")
	flagSet.StringVar(&cacheDir, "cacheDir", getDefaultCacheDir(), "Cache directory, f.e. for certificates.")
	flagSet.BoolVar(&debug, "debug", false, "debug mode.")

	flagSet.Parse(os.Args[1:])

	switch {
	case versionFlag:
		fmt.Println("hashworksNET")
		fmt.Println("https://github.com/hashworks/hashworksNET")
		fmt.Println("Version: " + VERSION)
		fmt.Println("Commit: " + BUILD_COMMIT)
		fmt.Println("Build date: " + BUILD_DATE)
		fmt.Println()
		fmt.Println("Published under the GNU General Public License v3.0.")
	default:
		s := server.NewServer(GIN_MODE, https, domain, cacheDir, debug)
		if https {
			if domain != "" {
				if err := s.RunTLS(fmt.Sprintf("%s:%d", address, port)); err != nil {
					fmt.Printf("Failed to start the https server: %s\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Println("Error: TLS requires a domain.")
				os.Exit(2)
			}
		} else {
			if err := s.Router.Run(fmt.Sprintf("%s:%d", address, port)); err != nil {
				fmt.Printf("Error: Failed to start the http server: %s\n", err)
				os.Exit(1)
			}
		}
	}
}

func getDefaultCacheDir() string {
	if userCacheDir, err := os.UserCacheDir(); err == nil {
		return path.Join(userCacheDir, "hashworksNET")
	}
	return "cache"
}
