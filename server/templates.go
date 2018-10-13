package server

import (
	"github.com/gin-contrib/multitemplate"
	"github.com/hashworks/hashworksNET/server/bindata"
	"html/template"
	"strings"
)

func (s Server) templateFunctionMap() template.FuncMap {
	return template.FuncMap{
		"css": func() template.CSS {
			return s.css
		},
		"version": func() string {
			return s.config.Version
		},
	}
}

func (s Server) loadTemplates() {
	// Load template file names from Asset
	templateNames, err := bindata.WalkDirs("templates", false)
	if err != nil {
		panic(err)
	}

	// Create a base template where we add the template functions
	tmpl := template.New("")
	tmpl.Funcs(s.templateFunctionMap())

	// Iterate trough template files, load them into multitemplate
	multiT := multitemplate.New()
	for _, templateName := range templateNames {
		index := strings.Index(templateName, "/")
		basename := templateName[index+1:]
		index = strings.Index(basename, ".")
		basename = basename[:index]
		tmpl := tmpl.New(basename)
		data, err := bindata.ReadFile(templateName)
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
