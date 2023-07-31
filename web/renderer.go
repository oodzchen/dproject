package web

import (
	"net/http"
	"text/template"

	"github.com/gorilla/sessions"
	"github.com/oodzchen/dproject/utils"
)

type PageData struct {
	Title  string
	Data   any
	TipMsg []string
}

type Renderer struct {
	Tmpl      *template.Template
	sessStore *sessions.CookieStore
}

func (rd *Renderer) Render(w http.ResponseWriter, r *http.Request, name string, data *PageData) {
	sess, err := rd.sessStore.Get(r, "flash-msg")
	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
		return
	}

	if flashes := sess.Flashes(); len(flashes) > 0 {
		for _, item := range flashes {
			switch value := item.(type) {
			case string:
				data.TipMsg = append(data.TipMsg, value)
			}
		}
	}
	sess.Save(r, w)

	err = rd.Tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
