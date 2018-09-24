package server

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
	"log"
	"net/http"
	"time"
)

func (s Server) handlerStatusSVG1940x1060(c *gin.Context) {
	s.handlerStatusSVG(c, 1940, 1060)
}

func (s Server) handlerStatusSVG1700x700(c *gin.Context) {
	s.handlerStatusSVG(c, 1700, 700)
}

func (s Server) handlerStatusSVG1380x520(c *gin.Context) {
	s.handlerStatusSVG(c, 1380, 520)
}

func (s Server) handlerStatusSVG1145x385(c *gin.Context) {
	s.handlerStatusSVG(c, 1145, 385)
}

func (s Server) handlerStatusSVG780x385(c *gin.Context) {
	s.handlerStatusSVG(c, 780, 385)
}

func (s Server) handlerStatusSVG500x335(c *gin.Context) {
	s.handlerStatusSVG(c, 500, 335)
}

func (s Server) handlerStatusSVG400x225(c *gin.Context) {
	s.handlerStatusSVG(c, 400, 225)
}

func (s Server) handlerStatusSVG200x115(c *gin.Context) {
	s.handlerStatusSVG(c, 200, 115)
}

func (s Server) handlerStatusSVG(c *gin.Context, width, height int) {
	httpClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:      s.config.InfluxAddress,
		Username:  s.config.InfluxUsername,
		Password:  s.config.InfluxPassword,
		UserAgent: "hashworksNET/" + s.config.Version,
	})
	defer httpClient.Close()

	if err != nil {
		recoveryHandler(c, err)
		return
	}

	q := client.Query{
		Command:   "SELECT mean(value) FROM bpm WHERE host = '" + s.config.InfluxHost + "' AND time > now() - 12h GROUP BY time(5m)",
		Database:  "body",
		Precision: "s",
	}

	resp, err := httpClient.Query(q)

	if err != nil {
		recoveryHandler(c, err)
		return
	}

	if len(resp.Results) == 0 || len(resp.Results[0].Series) == 0 || len(resp.Results[0].Series[0].Values) == 0 {
		recoveryHandler(c, errors.New("InfluxDB returned an empty result"))
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
			recoveryHandler(c, err)
			return
		}
		timeSeries.XValues = append(timeSeries.XValues, time.Unix(timestamp, 0))

		bpm, err := resp.Results[0].Series[0].Values[i][1].(json.Number).Float64()
		if err != nil {
			recoveryHandler(c, err)
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
		recoveryHandler(c, err)
		return
	}

	c.Status(200)
}

func (s Server) handlerStatus(c *gin.Context) {
	c.Header("Cache-Control", "max-age=600")
	c.HTML(http.StatusOK, "status", gin.H{
		"Title":     "status",
		"StatusTab": true,
	})
}
