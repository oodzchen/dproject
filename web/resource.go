package web

import (
	"log"
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/utils"
)

type MainResource struct {
	Renderer
	articleRs *ArticleResource
	store     *store.Store
}

func NewMainResource(tmpl *template.Template, ar *ArticleResource, store *store.Store) *MainResource {
	return &MainResource{
		Renderer{tmpl},
		ar,
		store,
	}
}

func (mr *MainResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Get("/", mr.articleRs.List)
	rt.Get("/register", mr.RegisterPage)
	rt.Post("/register", mr.Register)

	return rt
}

func (mr *MainResource) RegisterPage(w http.ResponseWriter, r *http.Request) {
	mr.Render(w, r, "register", &PageData{Title: "Register - Dproject", Data: ""})
}

func (mr *MainResource) Register(w http.ResponseWriter, r *http.Request) {
	email := r.PostFormValue("email")
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	user := &model.User{
		Email:    email,
		Name:     username,
		Password: password,
	}

	err := user.Valid()
	if err != nil {
		utils.HttpError("web.resource.Register", err, w, http.StatusBadRequest)
		return
	}

	log.Printf("user model is %v", user)

	err = user.EncryptPassword()
	if err != nil {
		utils.HttpError("web.resource.Register1", err, w, http.StatusInternalServerError)
	}

	// fmt.Printf("Password value: %s\n", user.Password)
	id, err := mr.store.User.Create(user)
	if err != nil {
		utils.HttpError("web.resource.Register2", err, w, http.StatusInternalServerError)
	}

	log.Printf("create user success, user id: %d", id)

	mr.Render(w, r, "register", &PageData{Title: "Register - Dproject", Data: ""})
}
