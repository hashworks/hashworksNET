package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	drawingUpstream "github.com/wcharczuk/go-chart/drawing"

	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/hashworks/go-chart"
)

var svgLoadDimensions = [][]int{
	{1120, 200},
	{1020, 200},
	{920, 200},
	{820, 200},
	{720, 200},
	{620, 200},
	{520, 200},
	{440, 200},
	{750, 200},
	{600, 200},
	{380, 200},
	{200, 115},
}

type Node struct {
	Name     string
	Services []Service
	Loads    []Load
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

func (s *Server) queryPrometheus(query string, ts time.Time) (model.Vector, error) {
	client, err := api.NewClient(api.Config{
		Address: "http://192.168.144.2:9090",
	})
	if err != nil {
		return nil, err
	}

	v1api := v1.NewAPI(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, warnings, err := v1api.Query(ctx, query, ts)
	if len(warnings) > 0 {
		log.Printf("Prometheus warnings: %v\n", warnings)
	}

	return result.(model.Vector), err
}

func (s *Server) queryPrometheusRange(query string, r v1.Range) (model.Matrix, error) {
	client, err := api.NewClient(api.Config{
		Address: "http://192.168.144.2:9090",
	})
	if err != nil {
		return nil, err
	}

	v1api := v1.NewAPI(client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, warnings, err := v1api.QueryRange(ctx, query, r)
	if len(warnings) > 0 {
		log.Printf("Prometheus warnings: %v\n", warnings)
	}

	return result.(model.Matrix), err
}

func (s *Server) queryNode(shortHostname string, fqdn string, plexDomain string, dotDomain string, snmp bool) (Node, error) {
	load1, err := s.queryPrometheus("node_load1{fqdn=\""+fqdn+"\"}", time.Now())
	if err != nil || load1.Len() != 1 {
		return Node{}, errors.New("Prometheus query 'node_load1' for " + shortHostname + " failed.")
	}

	load5, err := s.queryPrometheus("node_load5{fqdn=\""+fqdn+"\"}", time.Now())
	if err != nil || load5.Len() != 1 {
		return Node{}, errors.New("Prometheus query 'node_load5' for " + shortHostname + " failed.")
	}

	load15, err := s.queryPrometheus("node_load15{fqdn=\""+fqdn+"\"}", time.Now())
	if err != nil || load15.Len() != 1 {
		return Node{}, errors.New("Prometheus query 'node_load15' for " + shortHostname + " failed.")
	}

	var loads []Load

	for _, load := range []model.Vector{load1, load5, load15} {
		value, err := strconv.ParseFloat(load[0].Value.String(), 8)
		if err != nil {
			return Node{}, errors.New("Failed to parse load.")
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

	var services []Service

	probeSuccessPlex, err := s.queryPrometheus("probe_success{instance=\""+plexDomain+":32400\"}", time.Now())
	if err != nil || probeSuccessPlex.Len() != 1 {
		return Node{}, errors.New("Prometheus query 'probe_success' for plex on " + shortHostname + " failed.")
	}

	probeDurationPlex, err := s.queryPrometheus("probe_duration_seconds{instance=\""+plexDomain+":32400\"}", time.Now())
	if err != nil || probeDurationPlex.Len() != 1 {
		return Node{}, errors.New("Prometheus query 'probe_duration_seconds' for plex on " + shortHostname + " failed.")
	}

	probeNames := []string{"Plex"}
	probeSuccess := []model.Vector{probeSuccessPlex}
	probeDurations := []model.Vector{probeDurationPlex}

	if dotDomain != "" {
		probeSuccessDoT, err := s.queryPrometheus("probe_success{instance=\""+dotDomain+":853\"}", time.Now())
		if err != nil || probeSuccessDoT.Len() != 1 {
			return Node{}, errors.New("Prometheus query 'probe_success' for dot on " + shortHostname + " failed.")
		}

		probeDurationDoT, err := s.queryPrometheus("probe_duration_seconds{instance=\""+dotDomain+":853\"}", time.Now())
		if err != nil || probeDurationDoT.Len() != 1 {
			return Node{}, errors.New("Prometheus query 'probe_duration_seconds' for dot on " + shortHostname + " failed.")
		}

		probeNames = append(probeNames, "DNS")
		probeSuccess = append(probeSuccess, probeSuccessDoT)
		probeDurations = append(probeDurations, probeDurationDoT)
	}

	for i, probeDuration := range probeDurations {
		newService := Service{probeNames[i], "error", "No data!"}

		if probeSuccess[i][0].Value.String() != "1" {
			newService.Status = "error"
			newService.Message = "Offline."
		} else {
			probeDuration, err := strconv.ParseFloat(probeDuration[0].Value.String(), 8)
			if err != nil {
				return Node{}, errors.New("Failed to parse probe duration.")
			}
			if probeDuration >= 0.2 {
				newService.Status = "warning"
			} else {
				newService.Status = "ok"
			}
			newService.Message = fmt.Sprintf("Online. %.02fs latency.", probeDuration)
		}

		services = append(services, newService)
	}

	if snmp {
		outRate, err := s.queryPrometheus("irate(ifHCOutOctets{job=\"snmp\",ifName=\"eth0\"}[5m])", time.Now())
		if err != nil || outRate.Len() != 1 {
			return Node{}, errors.New("Prometheus query 'ifHCOutOctets' for " + shortHostname + " failed.")
		}
		outRateValue, err := strconv.ParseFloat(outRate[0].Value.String(), 8)
		if err != nil {
			return Node{}, errors.New("Failed to parse outRate.")
		}

		newService := Service{"Upstream Load", "error", "No data!"}
		if outRateValue != 0 {
			percentage := int(math.Min(outRateValue/float64(50000*1000)*100, 100))
			newService.Message = fmt.Sprintf("%d%% average utilisation over the last 5 minutes", percentage)
			if percentage > 90 {
				newService.Status = "error"
			} else if percentage > 50 {
				newService.Status = "warning"
			} else {
				newService.Status = "ok"
			}
		}
		services = append(services, newService)
	}

	return Node{shortHostname, services, loads}, nil
}

func (s *Server) handlerStatus(c *gin.Context) {
	pageStartTime := time.Now()

	nodeHive, err := s.queryNode("hive", "hive.hashworks.net", "plex.hive.hashworks.net", "", true)
	if err != nil {
		s.recoveryHandlerStatus(http.StatusInternalServerError, c, err)
		return
	}

	nodeHelios, err := s.queryNode("helios", "helios.kromlinger.eu", "plex.helios.hashworks.net", "dns.kromlinger.eu", false)
	if err != nil {
		s.recoveryHandlerStatus(http.StatusInternalServerError, c, err)
		return
	}

	c.Header("Cache-Control", "max-age=60")
	c.Header("Last-Modified", time.Now().Format(time.RFC1123))
	c.Header("Link", "</css/status.css>; rel=preload; as=style")
	c.HTML(http.StatusOK, "status", gin.H{
		"Title":         "status",
		"Description":   "Status information.",
		"StatusTab":     true,
		"PageStartTime": pageStartTime,
		"Nodes":         []Node{nodeHive, nodeHelios},
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

func (s *Server) handlerLoadSVG(hostname string, width, height int) func(*gin.Context) {
	return func(c *gin.Context) {
		values, err := s.queryPrometheusRange("node_load1{fqdn=\""+hostname+"\"}", v1.Range{
			Start: time.Now().Add(-time.Hour),
			End:   time.Now(),
			Step:  time.Minute,
		})
		if err != nil || values.Len() != 1 {
			s.recoveryHandlerStatus(http.StatusInternalServerError, c, errors.New("Prometheus query 'ifHCOutOctets' failed."))
			return
		}

		if len(values[0].Values) < 2 {
			messageSVG(c, "Not enough data collected in the last hour to draw a graph.", width)
			return
		}

		short := chart.TimeSeries{
			Name: "Short",
			Style: chart.Style{
				Show:      true,
				ClassName: "series",
				FillColor: drawingUpstream.ColorBlack, // Dummy-Fill so go-chart produces the fill-paths
			},
			XValues: []time.Time{},
			YValues: []float64{},
		}

		var max float64
		avg := 0
		count := 0 // Since len(…) could be wrong

		for _, samplePair := range values[0].Values {
			load, err := strconv.ParseFloat(samplePair.Value.String(), 8)
			if err != nil {
				s.recoveryHandlerStatus(http.StatusInternalServerError, c, errors.New("Failed to parse load."))
				return
			}

			short.XValues = append(short.XValues, samplePair.Timestamp.Time())
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
