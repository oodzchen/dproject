package web

import (
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/oodzchen/dproject/store"
)

type UserResource struct {
	Renderer
	store store.UserStore
}

func NewUserResource(tmpl *template.Template, store store.UserStore) *UserResource {
	return &UserResource{
		Renderer{Tmpl: tmpl},
		store,
	}
}

func (ur *UserResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Get("/register", ur.RegisterPage)

	return rt
}

func (ur *UserResource) RegisterPage(w http.ResponseWriter, r *http.Request) {
	ur.Render(w, r, "register", &PageData{Title: "Register - Dproject", Data: ""})
}
