package main

import (
	"net/http"
	"text/template"
)

type PageData struct {
	PageTitle string
	Data      any
}

type Renderer struct {
	tmpl *template.Template
}

func (rd *Renderer) render(w http.ResponseWriter, r *http.Request, name string, data *PageData) {
	err := rd.tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
