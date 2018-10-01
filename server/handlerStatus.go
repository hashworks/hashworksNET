package server

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
	"log"
	"net/http"
	"strings"
	"time"
)

var svgDimensions = [][]int{
	{1940, 1060},
	{1700, 700},
	{1380, 520},
	{1145, 385},
	{780, 385},
	{500, 335},
	{400, 225},
	{200, 115},
}

func (s *Server) handlerStatusSVG(width, height int) func(*gin.Context) {
	return func(c *gin.Context) {
		influxConfig := client.HTTPConfig{
			Addr:      s.config.InfluxAddress,
			UserAgent: "hashworksNET/" + s.config.Version,
		}
		if s.config.InfluxUsername != "" && s.config.InfluxPassword != "" {
			influxConfig.Username = s.config.InfluxUsername
			influxConfig.Password = s.config.InfluxPassword
		}
		httpClient, err := client.NewHTTPClient(influxConfig)
		defer httpClient.Close()

		if err != nil {
			s.recoveryHandlerStatus(http.StatusBadGateway, c, err)
			return
		}

		q := client.Query{
			Command:   "SELECT mean(value) FROM bpm WHERE host = '" + s.config.InfluxHost + "' AND time > now() - 12h GROUP BY time(5m)",
			Database:  "body",
			Precision: "s",
		}

		resp, err := httpClient.Query(q)

		if err != nil {
			s.recoveryHandlerStatus(http.StatusBadGateway, c, err)
			return
		}

		if len(resp.Results) == 0 || len(resp.Results[0].Series) == 0 {
			s.recoveryHandlerStatus(http.StatusInternalServerError, c, errors.New("InfluxDB query failed."))
			return
		}

		if len(resp.Results[0].Series[0].Values) < 2 {
			messageSVG(c, "Not enough heart-rate data collected in the last 12h to draw a graph.", width)
			return
		}

		if s.config.Debug {
			log.Println(resp.Results[0].Series[0].Values)
		}

		timeSeries := chart.TimeSeries{
			Name: "BPM",
			Style: chart.Style{
				Show:        true,
				StrokeColor: drawing.ColorRed,
				FillColor:   drawing.ColorRed.WithAlpha(64),
			},
			XValues: []time.Time{},
			YValues: []float64{},
		}

		for i := 0; i < len(resp.Results[0].Series[0].Values); i++ {
			if len(resp.Results[0].Series[0].Values[i]) == 0 || resp.Results[0].Series[0].Values[i][0] == nil || resp.Results[0].Series[0].Values[i][1] == nil {
				continue
			}

			timestamp, err := resp.Results[0].Series[0].Values[i][0].(json.Number).Int64()
			if err != nil {
				s.recoveryHandler(c, err)
				return
			}
			timeSeries.XValues = append(timeSeries.XValues, time.Unix(timestamp, 0))

			bpm, err := resp.Results[0].Series[0].Values[i][1].(json.Number).Float64()
			if err != nil {
				s.recoveryHandler(c, err)
				return
			}
			timeSeries.YValues = append(timeSeries.YValues, bpm)
		}

		backgroundColor := drawing.ColorFromHex("272727")
		foregroundColor := drawing.ColorWhite

		graph := chart.Chart{
			Height: int(height),
			Width:  int(width),
			Background: chart.Style{
				FillColor: backgroundColor,
			},
			Canvas: chart.Style{
				FillColor: backgroundColor,
			},
			XAxis: chart.XAxis{
				Style: chart.Style{
					FontColor: foregroundColor,
					Show:      true,
				},
				ValueFormatter: chart.TimeValueFormatterWithFormat("15:04"),
			},
			YAxis: chart.YAxis{
				Style: chart.Style{
					FontColor: foregroundColor,
					Show:      true,
				},
				ValueFormatter: chart.IntValueFormatter,
			},
			Series: []chart.Series{timeSeries},
		}

		graph.Elements = []chart.Renderable{
			chart.Legend(&graph, chart.Style{
				FillColor: backgroundColor,
				FontColor: foregroundColor,
			}),
		}

		c.Header("Content-Type", "image/svg+xml")
		c.Header("Cache-Control", "max-age=600")
		c.Header("Content-Security-Policy", s.getCSP(false)) // Our SVGs require inline CSS

		if err := graph.Render(chart.SVG, c.Writer); err != nil {
			c.AbortWithStatus(500)
			return
		}

		c.Status(200)
	}
}

func messageSVG(c *gin.Context, message string, width int) {
	var messages []string
	width += 100
	charactersPerLine := width / 15
	if len(message) <= charactersPerLine {
		messages = append(messages, `<tspan x="0" dy="30">`+strings.TrimSpace(message)+`</tspan>`)
	} else {
		words := strings.Fields(strings.TrimSpace(message))
		charactersLeft := charactersPerLine
		line := ""
		for _, word := range words {
			wordLength := len(word)
			if charactersLeft > 0 && (wordLength <= charactersLeft || wordLength > charactersPerLine) {
				line += word + " "
				charactersLeft -= wordLength
			} else {
				messages = append(messages, `<tspan x="0" dy="30">`+strings.TrimSpace(line)+`</tspan>`)
				line = word + " "
				charactersLeft = charactersPerLine - wordLength
			}
		}
		line = strings.TrimSpace(line)
		if line != "" {
			messages = append(messages, `<tspan x="0" dy="30">`+line+`</tspan>`)
		}
	}
	width += 10
	height := len(messages)*30 + 20

	c.Header("Content-Type", "image/svg+xml")
	c.Header("Cache-Control", "no-store")
	c.String(200, fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d">`+
		`<rect width="100%%" height="100%%" fill="#272727"/>`+
		`<text x="0" y="0" fill="white" font-size="24" font-family="sans-serif">%s</text>`+
		`</svg>`, width, height, strings.Join(messages, "")))
}

func (s *Server) handlerStatus(c *gin.Context) {
	c.Header("Cache-Control", "max-age=600")
	c.HTML(http.StatusOK, "status", gin.H{
		"Title":     "status",
		"StatusTab": true,
	})
}
