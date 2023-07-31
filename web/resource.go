package web

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/utils"
	"github.com/pkg/errors"
)

// https://www.postgresql.org/docs/current/errcodes-appendix.html
const PGErrUniqueViolation = "23505"

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
	rt.Get("/login", mr.LoginPage)
	rt.Post("/login", mr.Login)

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
		utils.HttpError("", errors.WithStack(err), w, http.StatusBadRequest)
		return
	}

	// log.Printf("user model is %v", user)

	err = user.EncryptPassword()
	if err != nil {
		utils.HttpError("", errors.WithStack(err), w, http.StatusInternalServerError)
		return
	}

	// fmt.Printf("Password value: %s\n", user.Password)
	id, err := mr.store.User.Create(user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PGErrUniqueViolation {
			// fmt.Println(pgErr.Code)
			// fmt.Println(pgErr.Message)
			utils.HttpError("the eamil already been registered", errors.WithStack(err), w, http.StatusBadRequest)
		} else {
			utils.HttpError("", errors.WithStack(err), w, http.StatusInternalServerError)
		}

		return
	}

	log.Printf("create user success, user id: %d", id)

	mr.Render(w, r, "register", &PageData{Title: "Register - Dproject", Data: ""})
}

func (mr *MainResource) LoginPage(w http.ResponseWriter, r *http.Request) {
	mr.Render(w, r, "login", &PageData{Title: "Login - Dproject", Data: ""})
}

func (mr *MainResource) Login(w http.ResponseWriter, r *http.Request) {
	email := r.PostFormValue("email")
	password := r.PostFormValue("password")

	if email == "" {
		utils.HttpError("email is required", nil, w, http.StatusBadRequest)
		return
	}

	if password == "" {
		utils.HttpError("password is required", nil, w, http.StatusBadRequest)
		return
	}

	ok := utils.ValidateEmail(email)
	if !ok {
		utils.HttpError("email or password incorrect", errors.WithStack(utils.NewError("email not valid")), w, http.StatusBadRequest)
		return
	}

	id, err := mr.store.User.Login(email, password)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.HttpError("the email has not been registered", errors.WithStack(err), w, http.StatusBadRequest)
		} else {
			utils.HttpError("email or password incorrect", errors.WithStack(err), w, http.StatusBadRequest)
		}

		return
	}

	user, err := mr.store.User.Item(id)
	if err != nil {
		utils.HttpError("internal server error", err, w, http.StatusInternalServerError)
	}

	fmt.Printf("user %d login success!\n", user.Id)

	//..
	mr.Render(w, r, "login", &PageData{Title: "Login - Dproject", Data: ""})
}
