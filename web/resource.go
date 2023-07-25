package web

import (
	"log"
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/utils"
	"golang.org/x/crypto/bcrypt"
)

type MainResource struct {
	Renderer
	articleRs *ArticleResource
}

func NewMainResource(tmpl *template.Template, ar *ArticleResource) *MainResource {
	return &MainResource{
		Renderer{tmpl},
		ar,
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

	log.Printf("Register form.email: %s, form.username: %s, form.password: %s", email, username, password)

	hashedPwd, err := hashPassword(password)
	if err != nil {
		utils.HttpError("web.resource.Register", err, w, http.StatusInternalServerError)
	}

	user := &model.User{
		Email:    email,
		Name:     username,
		Password: hashedPwd,
	}

	err = user.Valid()
	if err != nil {
		utils.HttpError("web.resource.RegisterPage", err, w, http.StatusBadRequest)
		return
	}

	mr.Render(w, r, "register", &PageData{Title: "Register - Dproject", Data: ""})
}

func hashPassword(pwd string) (string, error) {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", nil
	}

	return string(hashedPwd), nil
}
