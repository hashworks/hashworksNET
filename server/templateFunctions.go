package server

import (
	"html/template"
)

func (s Server) templateFunctionMap() template.FuncMap {
	return template.FuncMap{
		"css": func() template.CSS {
			bytes, err := Asset("css/main.css")
			if err != nil {
				panic(err)
			}
			return template.CSS(bytes)
		},
	}
}
