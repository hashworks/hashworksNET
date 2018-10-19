package server

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/hashworks/go-chart"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/wcharczuk/go-chart/drawing"
	"log"
	"net/http"
	"strings"
	"time"
)

var svgBPMDimensions = [][]int{
	{1940, 300},
	{1700, 300},
	{1380, 300},
	{1145, 300},
	{980, 300},
	{780, 300},
	{580, 300},
	{380, 200},
	{200, 115},
}

var svgLoadDimensions = [][]int{
	{800, 200},
	{620, 200},
	{520, 200},
	{440, 200},
	{750, 200},
	{600, 200},
	{380, 200},
	{200, 115},
}

type Service struct {
	Name    string
	Status  string
	Message string
}

type Load struct {
	Status string
	Value  float64
}

func (s *Server) queryInfluxDB(c *gin.Context, command, db string) *client.Response {
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
		s.recoveryHandlerStatus(http.StatusInternalServerError, c, err)
		return nil
	}

	q := client.Query{
		Command:   command,
		Database:  db,
		Precision: "s",
	}

	resp, err := httpClient.Query(q)

	if err != nil {
		s.recoveryHandlerStatus(http.StatusBadGateway, c, err)
		return nil
	}

	if len(resp.Results) == 0 {
		s.recoveryHandlerStatus(http.StatusInternalServerError, c, errors.New("InfluxDB query failed."))
		return nil
	}

	if s.config.Debug {
		log.Println(resp)
	}

	return resp
}

func (s *Server) handlerStatus(c *gin.Context) {
	pageStartTime := time.Now()

	resp := s.queryInfluxDB(c, "SELECT last(*) FROM net_response WHERE port = '32400' AND protocol='tcp' AND time > now() - 2m;"+
		"SELECT last(*) FROM net_response WHERE port = '6697' AND protocol='tcp' AND time > now() - 2m;"+
		fmt.Sprintf("SELECT last(load1), last(load5), last(load15) FROM system WHERE host = '%s' AND time > now() - 2m", s.config.InfluxLoadHost), "telegraf")
	if resp == nil {
		return
	}

	var loads []Load
	var services []Service

	for id, result := range resp.Results {
		if len(result.Series) != 1 {
			continue
		}

		series := result.Series[0]

		if series.Name == "net_response" {
			var name string
			if id == 0 {
				name = "Plex"
			} else {
				name = "ZNC"
			}
			values := series.Values
			newService := Service{name, "error", "No data!"}
			if len(values) == 1 && len(values[0]) == 4 {
				result, ok := values[0][3].(string)
				if !ok {
					s.recoveryHandlerStatus(http.StatusInternalServerError, c, errors.New("Failed to cast JSON result into string"))
					return
				}

				responseTime, err := values[0][1].(json.Number).Float64()
				if err != nil {
					s.recoveryHandlerStatus(http.StatusInternalServerError, c, err)
					return
				}

				resultCode, err := values[0][2].(json.Number).Int64()
				if err != nil {
					s.recoveryHandlerStatus(http.StatusInternalServerError, c, err)
					return
				}

				if resultCode != 0 {
					newService.Status = "error"
				} else if responseTime >= 0.2 {
					newService.Status = "warning"
				} else {
					newService.Status = "ok"
				}

				if result == "success" {
					newService.Message = fmt.Sprintf("Online. %.02fs latency.", responseTime)
				} else {
					newService.Message = fmt.Sprintf("%s.", strings.Title(result))
				}

			}
			services = append(services, newService)
		} else if series.Name == "system" {
			if len(series.Values) == 1 && len(series.Values[0]) == 4 {
				for i := 1; i < 4; i++ {
					load := series.Values[0][i]

					if data, ok := load.(json.Number); ok {
						value, err := data.Float64()
						if err != nil {
							s.recoveryHandlerStatus(http.StatusInternalServerError, c, err)
							return
						}

						var status string
						if value >= 8 {
							status = "error"
						} else if value >= 4 {
							status = "warning"
						} else {
							status = "ok"
						}

						loads = append(loads, Load{
							Value:  value,
							Status: status,
						})
					}
				}
			}
		}
	}

	c.Header("Cache-Control", "max-age=60")
	c.Header("Last-Modified", time.Now().Format(time.RFC1123))
	c.Header("Link", "</css/status.css>; rel=preload; as=style")
	c.HTML(http.StatusOK, "status", gin.H{
		"Title":         "status",
		"Description":   "Status information.",
		"StatusTab":     true,
		"PageStartTime": pageStartTime,
		"Services":      services,
		"Loads":         loads,
	})
}

func (s *Server) drawChart(c *gin.Context, graph chart.Chart) {
	c.Header("Content-Type", chart.ContentTypeSVG)
	c.Header("Cache-Control", "max-age=600")
	c.Header("Last-Modified", time.Now().Format(time.RFC1123))

	if err := graph.Render(chart.SVGWithCSS(s.chartCSS, ""), c.Writer); err != nil {
		log.Printf("%s - Error: %s", time.Now().Format(time.RFC3339), err.Error())
		c.AbortWithStatus(500)
		return
	}

	c.Status(200)
}

func (s *Server) handlerBPMSVG(width, height int) func(*gin.Context) {
	return func(c *gin.Context) {
		resp := s.queryInfluxDB(c, fmt.Sprintf("SELECT mean(value) FROM bpm WHERE host = '%s' AND time > now() - 12h GROUP BY time(5m)", s.config.InfluxBPMHost), "body")
		if resp == nil {
			return
		}

		if len(resp.Results[0].Series) == 0 || len(resp.Results[0].Series[0].Values) < 2 {
			messageSVG(c, "Not enough data collected in the last 12h to draw a graph.", width)
			return
		}

		timeSeries := chart.TimeSeries{
			Name: "BPM",
			Style: chart.Style{
				Show:      true,
				ClassName: "series",
				FillColor: drawing.ColorBlack, // Dummy-Fill so go-chart produces the fill-paths
			},
			XValues: []time.Time{},
			YValues: []float64{},
		}

		var max float64
		avg := 0
		count := 0 // Since len(…) could be wrong

		series := resp.Results[0].Series[0]
		length := len(series.Values)

		// Get last time
		var lastBPMTime time.Time
		for i := length - 1; i >= 0; i-- {
			if len(series.Values[i]) < 2 || series.Values[i][0] == nil || series.Values[i][1] == nil {
				continue
			}

			timestamp, err := series.Values[i][0].(json.Number).Int64()
			if err != nil {
				s.recoveryHandler(c, err)
				return
			}
			lastBPMTime = time.Unix(timestamp, 0)
			break
		}

		for i := 0; i < length; i++ {
			if len(series.Values[i]) < 2 || series.Values[i][0] == nil || series.Values[i][1] == nil {
				continue
			}

			timestamp, err := series.Values[i][0].(json.Number).Int64()
			if err != nil {
				s.recoveryHandler(c, err)
				return
			}
			bpmTime := time.Unix(timestamp, 0)
			timeSeries.XValues = append(timeSeries.XValues, bpmTime)

			bpm, err := series.Values[i][1].(json.Number).Float64()
			if err != nil {
				s.recoveryHandler(c, err)
				return
			}
			timeSeries.YValues = append(timeSeries.YValues, bpm)
			if bpm > max {
				max = bpm
			}

			// Only calculate average of last hour
			if bpmTime.Add(time.Hour).After(lastBPMTime) {
				avg += int(bpm)
				count++
			}
		}

		if count != 0 {
			avg /= count
		}

		var statusClass string
		if avg >= 130 {
			statusClass = "error"
		} else if avg >= 100 {
			statusClass = "warning"
		} else if avg >= 40 {
			statusClass = "ok"
		} else if avg >= 30 {
			statusClass = "warning"
		} else {
			statusClass = "error"
		}
		timeSeries.Style.ClassName += " " + statusClass

		graph := chart.Chart{
			Height: int(height),
			Width:  int(width),
			Background: chart.Style{
				ClassName: "bg",
			},
			Canvas: chart.Style{
				ClassName: "bg",
			},
			XAxis: chart.XAxis{
				Style: chart.Style{
					ClassName: "axis",
					Show:      true,
				},
				ValueFormatter: chart.TimeValueFormatterWithFormat("15:04"),
			},
			YAxis: chart.YAxis{
				Range: &chart.ContinuousRange{Min: 0, Max: max},
				Style: chart.Style{
					ClassName: "axis",
					Show:      true,
				},
				ValueFormatter: chart.IntValueFormatter,
			},
			Series: []chart.Series{timeSeries},
		}

		graph.Elements = []chart.Renderable{
			chart.Legend(&graph, chart.Style{
				ClassName: "legend " + statusClass,
			}),
		}

		s.drawChart(c, graph)
	}
}

func (s *Server) handlerLoadSVG(width, height int) func(*gin.Context) {
	return func(c *gin.Context) {
		resp := s.queryInfluxDB(c, fmt.Sprintf("SELECT load1 FROM system WHERE host = '%s' AND time > now() - 1h", s.config.InfluxLoadHost), "telegraf")
		if resp == nil {
			return
		}

		if len(resp.Results[0].Series) == 0 || len(resp.Results[0].Series[0].Values) <= 2 || len(resp.Results[0].Series[0].Values[0]) < 2 {
			messageSVG(c, "Not enough data collected in the last hour to draw a graph.", width)
			return
		}

		short := chart.TimeSeries{
			Name: "Short",
			Style: chart.Style{
				Show:      true,
				ClassName: "series",
				FillColor: drawing.ColorBlack, // Dummy-Fill so go-chart produces the fill-paths
			},
			XValues: []time.Time{},
			YValues: []float64{},
		}

		var max float64
		avg := 0
		count := 0 // Since len(…) could be wrong

		for _, values := range resp.Results[0].Series[0].Values {
			timeInt, err := values[0].(json.Number).Int64()
			if err != nil {
				s.recoveryHandlerStatus(http.StatusInternalServerError, c, err)
				return
			}
			timestamp := time.Unix(timeInt, 0)
			load, err := values[1].(json.Number).Float64()
			if err != nil {
				s.recoveryHandlerStatus(http.StatusInternalServerError, c, err)
				return
			}

			short.XValues = append(short.XValues, timestamp)
			short.YValues = append(short.YValues, load)

			if load > max {
				max = load
			}

			avg += int(load)
			count++
		}
		avg /= count

		var statusClass string
		if avg >= 8 {
			statusClass = "error"
		} else if avg >= 4 {
			statusClass = "warning"
		} else {
			statusClass = "ok"
		}
		short.Style.ClassName += " " + statusClass

		graph := chart.Chart{
			Height: int(height),
			Width:  int(width),
			Background: chart.Style{
				ClassName: "bg",
			},
			Canvas: chart.Style{
				ClassName: "bg",
			},
			XAxis: chart.XAxis{
				Style: chart.Style{
					ClassName: "axis",
					Show:      true,
				},
				ValueFormatter: chart.TimeValueFormatterWithFormat("15:04"),
			},
			YAxis: chart.YAxis{
				Range: &chart.ContinuousRange{Min: 0, Max: max},
				Style: chart.Style{
					ClassName: "axis",
					Show:      true,
				},
				ValueFormatter: chart.FloatValueFormatter,
			},
			Series: []chart.Series{short},
		}

		graph.Elements = []chart.Renderable{
			chart.Legend(&graph, chart.Style{
				ClassName: "legend " + statusClass,
			}),
		}

		s.drawChart(c, graph)
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
