package server

import (
	"fmt"
	"html/template"
	"runtime"
	"strings"
	"time"

	"github.com/gin-contrib/multitemplate"
)

func (s Server) templateFunctionMap() template.FuncMap {
	return template.FuncMap{
		"css": func() template.CSS {
			return s.css
		},
		"version": func() string {
			return s.config.Version
		},
		"GoVer": func() string {
			return strings.ToTitle(runtime.Version())
		},
		"LoadTimes": func(startTime time.Time) string {
			timeSinceST := time.Since(startTime).Nanoseconds() / 1e3
			if timeSinceST >= 1000 {
				return fmt.Sprint(timeSinceST/1e3) + "ms"
			} else {
				return fmt.Sprint(timeSinceST) + "µs"
			}
		},
	}
}

func (s Server) loadTemplates() {
	// Load template file names from Asset
	templateDirEntries, err := s.config.StaticContent.ReadDir("templates")
	if err != nil {
		panic(err)
	}

	// Create a base template where we add the template functions
	tmpl := template.New("")
	tmpl.Funcs(s.templateFunctionMap())

	// Iterate trough template files, load them into multitemplate
	multiT := multitemplate.New()
	for _, templateDirEntry := range templateDirEntries {
		index := strings.Index(templateDirEntry.Name(), "/")
		basename := templateDirEntry.Name()[index+1:]
		index = strings.Index(basename, ".")
		basename = basename[:index]
		tmpl := tmpl.New(basename)
		data, err := s.config.StaticContent.ReadFile("templates/" + templateDirEntry.Name())
		if err != nil {
			panic(err)
		}
		tmpl, err = tmpl.Parse(string(data))
		if err != nil {
			panic(err)
		}
		multiT.Add(basename, tmpl)
	}
	// multitemplate is our new HTML renderer
	s.Router.HTMLRender = multiT
}
