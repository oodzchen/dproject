package web

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
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
	*Renderer
	articleRs *ArticleResource
	store     *store.Store
}

func NewMainResource(tmpl *template.Template, ar *ArticleResource, store *store.Store, sessStore *sessions.CookieStore) *MainResource {
	return &MainResource{
		&Renderer{tmpl, sessStore},
		ar,
		store,
	}
}

func (mr *MainResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Get("/", mr.articleRs.List)
	rt.Get("/settings", mr.articleRs.List)
	rt.Post("/settings", mr.SaveSettings)
	rt.Get("/register", mr.RegisterPage)
	rt.Post("/register", mr.Register)
	rt.Get("/login", mr.LoginPage)
	rt.Post("/login", mr.Login)
	rt.Get("/logout", mr.Logout)

	return rt
}

func (mr *MainResource) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if IsLogin(mr.sessStore, w, r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	mr.Render(w, r, "register", &PageData{Title: "Register", Data: ""})
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

	user.Sanitize()

	err := user.Valid()
	if err != nil {
		mr.Error(err.Error(), errors.WithStack(err), w, r, http.StatusBadRequest)
		return
	}

	log.Printf("user model is %v", user)

	err = user.EncryptPassword()
	if err != nil {
		mr.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		return
	}

	// fmt.Printf("Password value: %s\n", user.Password)
	id, err := mr.store.User.Create(user)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PGErrUniqueViolation {
			// fmt.Println(pgErr.Code)
			// fmt.Println(pgErr.Message)
			mr.Error("the eamil already been registered", errors.WithStack(err), w, r, http.StatusBadRequest)
		} else {
			mr.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}

		return
	}

	log.Printf("create user success, user id: %d", id)

	sess, err := mr.sessStore.Get(r, "one-cookie")
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
	}

	sess.AddFlash("Register success")
	err = sess.Save(r, w)
	if err != nil {
		HandleSessionErr(errors.WithStack(err))
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

func (mr *MainResource) LoginPage(w http.ResponseWriter, r *http.Request) {
	if IsLogin(mr.sessStore, w, r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	mr.Render(w, r, "login", &PageData{Title: "Login", Data: ""})
}

func (mr *MainResource) Login(w http.ResponseWriter, r *http.Request) {
	email := r.PostFormValue("email")
	password := r.PostFormValue("password")

	if email == "" {
		mr.Error("email is required", nil, w, r, http.StatusBadRequest)
		return
	}

	if password == "" {
		mr.Error("password is required", nil, w, r, http.StatusBadRequest)
		return
	}

	if !utils.ValidateEmail(email) {
		mr.Error("email or password is incorrect", errors.WithStack(utils.NewError("email not valid")), w, r, http.StatusBadRequest)
		return
	}

	id, err := mr.store.User.Login(email, password)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			mr.Error("the email has not been registered", errors.WithStack(err), w, r, http.StatusBadRequest)
		} else {
			mr.Error("email or password is incorrect", errors.WithStack(err), w, r, http.StatusBadRequest)
		}

		return
	}

	user, err := mr.store.User.Item(id)
	if err != nil {
		mr.Error("internal server error", err, w, r, http.StatusInternalServerError)
	}

	fmt.Printf("user %d login success!\n", user.Id)

	sess, err := mr.sessStore.Get(r, "one-cookie")
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}
	// sess.AddFlash(fmt.Sprintf("Hi, %s", user.Name))

	sess.Values["user_id"] = user.Id
	sess.Values["user_name"] = user.Name

	targetUrl, _ := sess.Values["target_url"].(string)
	sess.Values["target_url"] = ""

	sess.Options.HttpOnly = true
	sess.Options.Secure = !utils.IsDebug()
	sess.Options.SameSite = http.SameSiteLaxMode
	sess.Options.Path = "/"

	err = sess.Save(r, w)
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	//TODO: replace with session cookie
	// refererUrl, err := url.Parse(r.Referer())

	if len(targetUrl) > 0 {
		http.Redirect(w, r, targetUrl, http.StatusFound)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func (mr *MainResource) Logout(w http.ResponseWriter, r *http.Request) {
	// if IsLogin(mr.sessStore, w, r) {
	// 	sess, _ := mr.sessStore.Get(r, "one-cookie")
	// 	sess.Options.MaxAge = -1
	// 	err := sess.Save(r, w)
	// 	if err != nil {
	// 		HandleSessionErr(errors.WithStack(err))
	// 	}
	// }

	sess, err := mr.sessStore.Get(r, "one-cookie")
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}
	sess.Options.MaxAge = -1
	err = sess.Save(r, w)
	if err != nil {
		HandleSessionErr(errors.WithStack(err))
	}

	csrfExpiredCookie := &http.Cookie{
		Name:     "secure",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   !utils.IsDebug(),
		Path:     "/",
	}

	http.SetCookie(w, csrfExpiredCookie)

	refererUrl, err := url.Parse(r.Referer())

	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		http.Redirect(w, r, refererUrl.String(), http.StatusFound)
	}
}

func (mr *MainResource) SaveSettings(w http.ResponseWriter, r *http.Request) {
	theme := r.PostForm.Get("theme")

	fmt.Printf("post theme: %s\n", theme)

	if regexp.MustCompile(`^light|dark|system$`).Match([]byte(theme)) {
		sess := mr.Session("one-cookie", w, r)
		sess.SetValue("page_theme", theme)
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
