package web

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/service"
	"github.com/oodzchen/dproject/utils"
	"github.com/pkg/errors"
)

// https://www.postgresql.org/docs/current/errcodes-appendix.html
const PGErrUniqueViolation = "23505"

type MainResource struct {
	*Renderer
	articleRs *ArticleResource
	// store     *store.Store
	userSrv *service.User
}

func NewMainResource(renderer *Renderer, ar *ArticleResource) *MainResource {
	return &MainResource{
		renderer,
		ar,
		&service.User{
			Store: renderer.store,
		},
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

	if config.Config.Debug {
		rt.Post("/login_debug", mr.LoginDebug)
	}

	return rt
}

func (mr *MainResource) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if IsLogin(mr.sessStore, w, r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	mr.Render(w, r, "register", &PageData{
		Title: "Register",
		Data:  "",
		BreadCrumbs: []*BreadCrumb{
			{
				"/register",
				"Register",
			},
		},
	})
}

func (mr *MainResource) Register(w http.ResponseWriter, r *http.Request) {
	email := r.PostFormValue("email")
	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	_, err := mr.userSrv.Register(email, password, username)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, model.ErrValidUserFailed) {
			mr.Error(err.Error(), err, w, r, http.StatusBadRequest)
		} else if errors.As(err, &pgErr) && pgErr.Code == PGErrUniqueViolation {
			// fmt.Println(pgErr.Code)
			// fmt.Println(pgErr.Message)
			mr.Error("the eamil already been registered", model.NewAppError(err, model.ErrAlreadyRegistered), w, r, http.StatusBadRequest)
		} else {
			mr.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}

		return
	}

	// log.Printf("create user success, user id: %d", id)

	// sess, err := mr.sessStore.Get(r, "one")
	// if err != nil {
	// 	mr.Error("", err, w, r, http.StatusInternalServerError)
	// }

	// sess.AddFlash("Account registered successfully")
	// err = sess.Save(r, w)
	// if err != nil {
	// 	HandleSaveSessionErr(errors.WithStack(err))
	// }

	mr.Session("one", w, r).Flash("Account registered successfully")

	http.Redirect(w, r, "/login", http.StatusFound)
}

func (mr *MainResource) LoginPage(w http.ResponseWriter, r *http.Request) {
	if IsLogin(mr.sessStore, w, r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	targetUrl := mr.Session("one", w, r).GetValue("target_url")
	referer := r.Referer()
	refererUrl, _ := url.Parse(referer)

	fmt.Println("exist target_url: ", targetUrl)
	fmt.Println("target_url is empty string: ", targetUrl == "")

	if (targetUrl == nil || targetUrl == "") && IsRegisterdPage(refererUrl, mr.router) {
		// fmt.Println("Matched!", "target:", referer)
		targetUrl = referer
	}

	mr.Session("one", w, r).SetValue("target_url", targetUrl)
	mr.Render(w, r, "login", &PageData{
		Title: "Login",
		Data:  "",
		BreadCrumbs: []*BreadCrumb{
			{
				"/login",
				"Login",
			},
		},
	})
}

func (mr *MainResource) Login(w http.ResponseWriter, r *http.Request) {
	email := r.PostFormValue("email")
	password := r.PostFormValue("password")

	mr.doLogin(w, r, email, password)

	// targetUrl, _ := sess.Values["target_url"].(string)
	target := mr.Session("one", w, r).GetValue("target_url")

	mr.Session("one", w, r).SetValue("target_url", "")
	// fmt.Println("target: ", target)
	if targetUrl, ok := target.(string); ok && len(targetUrl) > 0 {
		http.Redirect(w, r, targetUrl, http.StatusFound)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func (mr *MainResource) doLogin(w http.ResponseWriter, r *http.Request, email, password string) {
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
			mr.Error("the email has not been registered", model.NewAppError(err, model.ErrNotRegistered), w, r, http.StatusBadRequest)
		} else {
			mr.Error("email or password is incorrect", errors.WithStack(err), w, r, http.StatusBadRequest)
		}

		return
	}

	user, err := mr.store.User.Item(id)
	if err != nil {
		mr.Error("internal server error", err, w, r, http.StatusInternalServerError)
	}

	mr.permissionSrv.SetLoginedUser(user)

	sess, err := mr.sessStore.Get(r, "one")
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	sess.Values["user_id"] = user.Id
	sess.Values["user_name"] = user.Name

	// gob.Register([]string{})
	// sess.Values["user_permitted_id_list"] = permittedIdList

	sess.Options.HttpOnly = true
	sess.Options.Secure = !utils.IsDebug()
	sess.Options.SameSite = http.SameSiteLaxMode
	sess.Options.Path = "/"

	err = sess.Save(r, w)
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}
}

func (mr *MainResource) LoginDebug(w http.ResponseWriter, r *http.Request) {
	email := r.PostForm.Get("debug-user-email")
	password := config.Config.DB.UserDefaultPassword
	fmt.Println("debug-user-email: ", email)

	mr.doLogin(w, r, email, password)
	// mr.ToPrevPage(w, r)
	mr.ToRefererUrl(w, r)
}

func (mr *MainResource) Logout(w http.ResponseWriter, r *http.Request) {
	mr.doLogout(w, r)

	refererUrl, err := url.Parse(r.Referer())

	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		http.Redirect(w, r, refererUrl.String(), http.StatusFound)
	}
}

func (mr *MainResource) doLogout(w http.ResponseWriter, r *http.Request) {
	sess, err := mr.sessStore.Get(r, "one")
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}
	ClearSession(sess, w, r)

	// sess.Options.MaxAge = -1
	// err = sess.Save(r, w)
	// if err != nil {
	// 	HandleSaveSessionErr(errors.WithStack(err))
	// }

	// csrfExpiredCookie := &http.Cookie{
	// 	Name:     "sc",
	// 	Value:    "",
	// 	Expires:  time.Unix(0, 0),
	// 	HttpOnly: true,
	// 	Secure:   !utils.IsDebug(),
	// 	Path:     "/",
	// }

	// http.SetCookie(w, csrfExpiredCookie)
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
				return
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
