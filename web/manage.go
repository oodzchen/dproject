package web

import (
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/oodzchen/dproject/store"
)

type ManageResource struct {
	*Renderer
	ur *UserResource
}

func NewManageResource(tmpl *template.Template, store *store.Store, sessStore *sessions.CookieStore, router *chi.Mux, ur *UserResource) *ManageResource {
	return &ManageResource{
		&Renderer{
			tmpl,
			sessStore,
			router,
			store,
		},
		ur,
	}
}

func (mr *ManageResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Route("/", func(r chi.Router) {
		r.Get("/permissions", mr.PermissionListPage)
		// r.Get("/roles", mr.RoleListPage)
		r.Get("/users", mr.ur.List)
	})

	return rt
}

func (mr *ManageResource) PermissionListPage(w http.ResponseWriter, r *http.Request) {

}
