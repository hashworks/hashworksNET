// Any changes to the sassc rules need to be added to the CI configuration as well
//go:generate sassc -p 2 -t compressed sass/main.scss sass/main.css
//go:generate sassc -p 2 -t compressed sass/status.scss sass/status.css
//go:generate sassc -p 2 -t compressed sass/chart.scss sass/chart.css
//go:generate go run github.com/UnnoTed/fileb0x b0x.yaml
package main

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/hashworks/hashworksNET/server"
	"github.com/urfave/cli"
)

var (
	// Set the following uppercase three with -ldflags "-X main.VERSION=v1.2.3 [...]"
	VERSION    = "?.?.?"
	BUILD_DATE = "unknown date"
	GIN_MODE   = gin.DebugMode
)

func main() {
	app := cli.NewApp()
	app.Name = "hashworksNET"
	app.Description = "The server of hashworks.net"
	app.UsageText = "hashworksNET [global options]"
	app.Version = fmt.Sprintf("%s (%s)", VERSION, BUILD_DATE)
	app.Copyright = "GNU General Public License v3.0"

	config := server.Config{GinMode: GIN_MODE, Version: VERSION, BuildDate: BUILD_DATE}

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			EnvVar:      "HWNET_DEBUG",
			Name:        "debug",
			Usage:       "enables debug mode",
			Destination: &config.Debug,
		},
		cli.StringFlag{
			EnvVar:      "HWNET_DOMAIN",
			Name:        "domain",
			Usage:       "domain the service is reachable at",
			Value:       "",
			Destination: &config.Domain,
		},
		cli.StringFlag{
			EnvVar: "HWNET_ADDRESS",
			Name:   "address",
			Usage:  "address to listen on",
			Value:  "127.0.0.1:65432",
		},
		cli.BoolFlag{
			EnvVar:      "HWNET_TLS_PROXY",
			Name:        "tlsProxy",
			Usage:       "set if service is behind a TLS proxy",
			Destination: &config.TLSProxy,
		},
		cli.BoolFlag{
			EnvVar:      "HWNET_GZIP",
			Name:        "gzip",
			Usage:       "enables gzip compression",
			Destination: &config.GZIPExtension,
		},
		cli.StringFlag{
			EnvVar:      "HWNET_INFLUX_LOAD_HOST",
			Name:        "influxLoadHost",
			Usage:       "InfluxDB load measurements host",
			Value:       "hive",
			Destination: &config.InfluxLoadHost,
		},
		cli.StringFlag{
			EnvVar:      "HWNET_INFLUX_UPSTREAM_HOST",
			Name:        "influxUpstreamHost",
			Usage:       "InfluxDB upstream measurements host",
			Value:       "127.0.0.1",
			Destination: &config.InfluxUpstreamHost,
		},
		cli.StringFlag{
			EnvVar:      "HWNET_INFLUX_UPSTREAM_INTERFACE",
			Name:        "influxUpstreamInterface",
			Usage:       "InfluxDB upstream measurements interface",
			Value:       "eth0",
			Destination: &config.InfluxUpstreamInterface,
		},
		cli.IntFlag{
			EnvVar:      "HWNET_INFLUX_UPSTREAM_MAX",
			Name:        "influxUpstreamMax",
			Usage:       "InfluxDB upstream maximum in kilobyte",
			Value:       125000,
			Destination: &config.InfluxUpstreamMax,
		},
		cli.StringFlag{
			EnvVar:      "HWNET_INFLUX_ADDRESS",
			Name:        "influxAddress",
			Usage:       "InfluxDB address",
			Value:       "http://127.0.0.1:8086",
			Destination: &config.InfluxAddress,
		},
		cli.StringFlag{
			EnvVar:      "HWNET_INFLUX_USERNAME",
			Name:        "influxUsername",
			Usage:       "InfluxDB username",
			Value:       "",
			Destination: &config.InfluxUsername,
		},
		cli.StringFlag{
			EnvVar:      "HWNET_INFLUX_PASSWORD",
			Name:        "influxPassword",
			Usage:       "InfluxDB password",
			Value:       "",
			Destination: &config.InfluxPassword,
		},
	}

	app.Action = func(cli *cli.Context) error {
		s, err := server.NewServer(config)
		if err != nil {
			return err
		}
		if err := s.Router.Run(cli.String("address")); err != nil {
			return err
		}
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
