package web

import (
	"net/http"
	"text/template"
)

type PageData struct {
	Title     string
	Data      any
	TipStatus string
}

type Renderer struct {
	Tmpl *template.Template
}

func (rd *Renderer) Render(w http.ResponseWriter, r *http.Request, name string, data *PageData) {
	ctx := r.Context()
	status, ok := ctx.Value("global_tip_status").(TipStatus)
	if ok {
		data.TipStatus = TipStatusStr[status]
	}

	err := rd.Tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
