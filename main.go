package main

import (
	"flag"
	"fmt"
	"github.com/hashworks/hashworksNET/server"
	"os"
)

var (
	// Set the following uppercase three with -ldflags "-X main.VERSION=v1.2.3 [...]"
	VERSION      string = "unknown"
	BUILD_COMMIT string = "unknown"
	BUILD_DATE   string = "unknown"
	versionFlag  bool
	address      string
	port         int
)

func main() {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	flagSet.BoolVar(&versionFlag, "version", false, "Displays the version and license information.")
	flagSet.StringVar(&address, "address", "", "The address to listen on.")
	flagSet.IntVar(&port, "port", 65431, "The port to listen on.")

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
		s := server.NewServer()
		s.Router.Run(fmt.Sprintf("%s:%d", address, port))
	}
}
