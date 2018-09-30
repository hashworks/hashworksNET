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
			if !s.config.Debug {
				return s.css
			} else { // On debug mode we normally don't include the CSS in our binary. This means we can edit it live
				return template.CSS(bindata.MustAsset("sass/main.css"))
			}
		},
	}
}

func (s Server) loadTemplates() {
	// Load template file names from Asset
	templateNames, err := bindata.AssetDir("templates")
	if err != nil {
		panic(err)
	}

	// Create a base template where we add the template functions
	tmpl := template.New("")
	tmpl.Funcs(s.templateFunctionMap())

	// Iterate trough template files, load them into multitemplate
	multiT := multitemplate.New()
	for _, templateName := range templateNames {
		index := strings.Index(templateName, ".")
		basename := templateName[:index]
		tmpl := tmpl.New(basename)
		tmpl, err := tmpl.Parse(string(bindata.MustAsset("templates/" + templateName)))
		if err != nil {
			panic(err)
		}
		multiT.Add(basename, tmpl)
	}
	// multitemplate is our new HTML renderer
	s.Router.HTMLRender = multiT
}
