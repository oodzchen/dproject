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

func NewMainResource(tmpl *template.Template, store *store.Store, sessStore *sessions.CookieStore, ar *ArticleResource, router *chi.Mux) *MainResource {
	return &MainResource{
		&Renderer{tmpl, sessStore, router},
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
	rt.Get("/logout", mr.Logout)
	rt.Route("/settings", func(r chi.Router) {
		r.Get("/", mr.SettingsPage)
		r.Get("/account", mr.SettingsAccountPage)
		r.Post("/account", mr.SaveAccountSettings)
		r.Get("/ui", mr.SettingsUIPage)
		r.Post("/ui", mr.SaveUISettings)
	})

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

	sess, err := mr.sessStore.Get(r, "one")
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
	}

	sess.AddFlash("Register success")
	err = sess.Save(r, w)
	if err != nil {
		HandleSaveSessionErr(errors.WithStack(err))
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

func (mr *MainResource) LoginPage(w http.ResponseWriter, r *http.Request) {
	if IsLogin(mr.sessStore, w, r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	targetUrl := ""
	refererUrl, _ := url.Parse(r.Referer())

	// fmt.Println("referUrl: ", r.Referer())
	// fmt.Println("referUrl host: ", refererUrl.Host)
	// fmt.Println("current host: ", config.Config.GetHost())
	if IsRegisterdPage(refererUrl, mr.router) {
		fmt.Println("Matched!")
		targetUrl = r.Referer()
	}

	mr.Session("one", w, r).SetValue("target_url", targetUrl)
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

	sess, err := mr.sessStore.Get(r, "one")
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
	// 	sess, _ := mr.sessStore.Get(r, "one")
	// 	sess.Options.MaxAge = -1
	// 	err := sess.Save(r, w)
	// 	if err != nil {
	// 		HandleSaveSessionErr(errors.WithStack(err))
	// 	}
	// }

	sess, err := mr.sessStore.Get(r, "one")
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}
	sess.Options.MaxAge = -1
	err = sess.Save(r, w)
	if err != nil {
		HandleSaveSessionErr(errors.WithStack(err))
	}

	csrfExpiredCookie := &http.Cookie{
		Name:     "sc",
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

func (mr *MainResource) SaveUISettings(w http.ResponseWriter, r *http.Request) {
	theme := r.PostForm.Get("theme")
	contentLayout := r.PostForm.Get("content_layout")

	// fmt.Printf("post theme: %s\n", theme)

	localSess := mr.Session("local", w, r)
	if regexp.MustCompile(`^light|dark|system$`).Match([]byte(theme)) {
		localSess.Raw.Options.HttpOnly = false
		localSess.Raw.Options.Path = "/"
		localSess.Raw.Options.SameSite = http.SameSiteLaxMode
		localSess.Raw.Options.Secure = !utils.IsDebug()
		localSess.Raw.Options.MaxAge = 0
		localSess.SetValue("page_theme", theme)
	}

	if regexp.MustCompile(`^full|centered$`).Match([]byte(contentLayout)) {
		localSess.Raw.Options.HttpOnly = false
		localSess.Raw.Options.Path = "/"
		localSess.Raw.Options.SameSite = http.SameSiteLaxMode
		localSess.Raw.Options.Secure = !utils.IsDebug()
		localSess.Raw.Options.MaxAge = 0
		localSess.SetValue("page_content_layout", contentLayout)
	}

	oneSess := mr.Session("one", w, r)
	oneSess.Raw.AddFlash("UI settings successfully saved")
	oneSess.Raw.Save(r, w)

	http.Redirect(w, r, "/settings/ui", http.StatusFound)
}

func (mr *MainResource) SettingsPage(w http.ResponseWriter, r *http.Request) {
	if IsLogin(mr.sessStore, w, r) {
		http.Redirect(w, r, "/settings/account", http.StatusFound)
	} else {
		http.Redirect(w, r, "/settings/ui", http.StatusFound)
	}
}

type SettingsPageKey string

const (
	SettingsPageKeyUI      SettingsPageKey = "ui"
	SettingsPageKeyAccount                 = "account"
)

type SettingsPageData struct {
	PageKey     SettingsPageKey
	AccountData *model.User
}

func (mr *MainResource) handleSettingsPage(w http.ResponseWriter, r *http.Request, pageKey SettingsPageKey) {
	settingsTitleMap := map[SettingsPageKey]string{
		SettingsPageKeyUI:      "UI",
		SettingsPageKeyAccount: "Account",
	}

	pageData := &SettingsPageData{
		PageKey: pageKey,
	}
	if pageKey == SettingsPageKeyAccount {
		userId := mr.Session("one", w, r).GetValue("user_id")
		// fmt.Println("user id: ", userId)
		if userId, ok := userId.(int); ok {
			user, err := mr.store.User.Item(userId)
			if err != nil {
				mr.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
			}
			pageData.AccountData = user
		} else {
			http.Redirect(w, r, "/login", http.StatusFound)
		}
	}

	// mr.Session("one-cookie", w, r).SetValue("next_url", r.Referer())
	mr.Render(w, r, "settings", &PageData{
		Title: settingsTitleMap[pageKey] + " Settings",
		Data:  pageData,
		BreadCrumbs: []*BreadCrumb{
			{
				"/settings",
				"Settings",
			},
		},
	})
}

func (mr *MainResource) SettingsAccountPage(w http.ResponseWriter, r *http.Request) {
	mr.handleSettingsPage(w, r, SettingsPageKeyAccount)
}

func (mr *MainResource) SettingsUIPage(w http.ResponseWriter, r *http.Request) {
	mr.handleSettingsPage(w, r, SettingsPageKeyUI)
}

func (mr *MainResource) SaveAccountSettings(w http.ResponseWriter, r *http.Request) {
	introduction := r.FormValue("introduction")

	if userId, ok := mr.Session("one", w, r).GetValue("user_id").(int); ok {
		user := &model.User{
			Id:           userId,
			Introduction: introduction,
		}
		user.Sanitize()
		_, err := mr.store.User.Update(user, []string{"Introduction"})
		if err != nil {
			mr.Error("Update account failed", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}

		oneSess := mr.Session("one", w, r)
		oneSess.Raw.AddFlash("Account settings successfully saved")
		oneSess.Raw.Save(r, w)

		http.Redirect(w, r, "/settings/account", http.StatusFound)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}
