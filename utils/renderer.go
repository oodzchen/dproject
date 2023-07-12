package utils

import (
	"net/http"
	"text/template"
)

type PageData struct {
	Title string
	Data  any
}

type Renderer struct {
	Tmpl *template.Template
}

func (rd *Renderer) Render(w http.ResponseWriter, r *http.Request, name string, data *PageData) {
	err := rd.Tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
