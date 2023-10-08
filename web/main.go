package web

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oodzchen/dproject/config"
	mdw "github.com/oodzchen/dproject/middleware"
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
			Store:         renderer.store,
			SantizePolicy: renderer.sanitizePolicy,
		},
	}
}

func (mr *MainResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Get("/", mr.articleRs.List)
	rt.Get("/register", mr.RegisterPage)
	rt.With(mdw.UserLogger(
		mr.uLogger, model.AcTypeUser, model.AcActionRegister, model.AcModelEmpty, mdw.ULogEmpty),
	).Post("/register", mr.Register)
	rt.Get("/login", mr.LoginPage)
	rt.With(mdw.UserLogger(
		mr.uLogger, model.AcTypeUser, model.AcActionLogin, model.AcModelEmpty, mdw.ULogEmpty),
	).Post("/login", mr.Login)
	rt.With(mdw.AuthCheck(mr.sessStore), mdw.UserLogger(
		mr.uLogger, model.AcTypeUser, model.AcActionLogout, "", mdw.ULogEmpty),
	).Post("/logout", mr.Logout)

	rt.Route("/settings", func(r chi.Router) {
		r.Get("/", mr.SettingsPage)
		r.With(mdw.AuthCheck(mr.sessStore)).Get("/account", mr.SettingsAccountPage)

		r.With(mdw.AuthCheck(mr.sessStore), mdw.PermitCheck(mr.permissionSrv, []string{
			"user.update_intro_mine",
			// "user.update_intro_others",
		}, mr),
			mdw.UserLogger(mr.uLogger, model.AcTypeUser, model.AcActionUpdateIntro, model.AcModelEmpty, mdw.ULogEmpty),
		).Post("/account", mr.SaveAccountSettings)
		r.Get("/ui", mr.SettingsUIPage)
		r.Post("/ui", mr.SaveUISettings)
	})

	if config.Config.Debug {
		rt.With(mdw.UserLogger(mr.uLogger, model.AcTypeDev, model.AcActionLogin, model.AcModelEmpty, mdw.ULogEmpty)).Post("/login_debug", mr.LoginDebug)
	}

	return rt
}

func (mr *MainResource) RegisterPage(w http.ResponseWriter, r *http.Request) {
	if IsLogin(mr.sessStore, w, r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	mr.Render(w, r, "register", &model.PageData{
		Title: mr.Local("Register"),
		Data:  "",
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: "/register",
				Name: "Register",
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
			alreadyExistsTip := mr.Local("AlreadyExists", "FieldNames", mr.Local("Or", "A", mr.Local("Username"), "B", mr.Local("Email")))
			mr.Error(alreadyExistsTip, model.NewAppError(err, model.ErrAlreadyRegistered), w, r, http.StatusBadRequest)
		} else {
			mr.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}

		return
	}

	mr.Session("one", w, r).Flash(mr.i18nCustom.MustLocalize("AccountCreateSuccess", "", ""))

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
	// fmt.Println("exist target_url: ", targetUrl)
	// fmt.Println("target_url is empty string: ", targetUrl == "")

	if (targetUrl == nil || targetUrl == "") && IsRegisterdPage(refererUrl, mr.router) {
		// fmt.Println("Matched!", "target:", referer)
		targetUrl = referer
	}

	mr.Session("one", w, r).SetValue("target_url", targetUrl)
	mr.Render(w, r, "login", &model.PageData{
		Title: mr.i18nCustom.LocalTpl("Login"),
		Data:  "",
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: "/login",
				Name: "Login",
			},
		},
	})
}

func (mr *MainResource) Login(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PostFormValue("username"))
	password := strings.TrimSpace(r.PostFormValue("password"))

	mr.doLogin(w, r, username, password)

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

func (mr *MainResource) doLogin(w http.ResponseWriter, r *http.Request, username, password string) {
	if username == "" {
		userNameRequiredTip := mr.Local("Required", "FieldNames", mr.Local("Or", "A", mr.Local("Username"), "B", mr.Local("Email")))
		mr.Error(userNameRequiredTip, nil, w, r, http.StatusBadRequest)
		return
	}

	if password == "" {
		mr.Error(mr.Local("Required", "FieldNames", mr.Local("Password")), nil, w, r, http.StatusBadRequest)
		return
	}

	loginFailedTip := mr.Local("Incorrect", "FieldNames", mr.Local("Or", "A", mr.Local("Username"), "B", mr.Local("Password")))

	if regexp.MustCompile(`@`).Match([]byte(username)) {
		if err := model.ValidateEmail(username); err != nil {
			emailValidTip := mr.Local("Incorrect", "FieldNames", mr.Local("Or", "A", mr.Local("Email"), "B", mr.Local("Password")))
			mr.Error(emailValidTip, err, w, r, http.StatusBadRequest)
			return
		}
	} else {
		if err := model.ValidUsername(username); err != nil {
			usernameValidTip := loginFailedTip
			mr.Error(usernameValidTip, err, w, r, http.StatusBadRequest)
			return
		}
	}

	id, err := mr.store.User.Login(username, password)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			notRegisterdTip := mr.Local("NotRegistered", "FieldNames", mr.Local("Or", "A", mr.Local("Username"), "B", mr.Local("Email")))
			mr.Error(notRegisterdTip, model.NewAppError(err, model.ErrNotRegistered), w, r, http.StatusBadRequest)
		} else {
			mr.Error(loginFailedTip, err, w, r, http.StatusBadRequest)
		}

		return
	}

	user, err := mr.store.User.Item(id)
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
	}

	mr.permissionSrv.SetLoginedUser(user)

	sess, err := mr.sessStore.Get(r, "one")
	if err != nil {
		logSessError("one", errors.WithStack(err))
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

	ctx := context.WithValue(r.Context(), "user_data", user)
	*r = *r.WithContext(ctx)
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
		logSessError("one", errors.WithStack(err))
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}
	ClearSession(sess, w, r)

	mr.permissionSrv.ResetPermissionData()
}

func (mr *MainResource) SaveUISettings(w http.ResponseWriter, r *http.Request) {
	lang := r.PostForm.Get("lang")
	theme := r.PostForm.Get("theme")
	contentLayout := r.PostForm.Get("content_layout")

	uiSettings := &model.UISettings{}

	localSess := mr.Session("local", w, r)
	if lang, err := model.ParseLang(lang); err == nil {
		// fmt.Println("post lang: ", lang)
		uiSettings.Lang = lang
		mr.i18nCustom.SwitchLang(string(lang))
		localSess.SetValue("lang", lang)
	}

	if regexp.MustCompile(`^light|dark|system$`).Match([]byte(theme)) {
		uiSettings.Theme = theme
		localSess.SetValue("page_theme", theme)
	}

	if regexp.MustCompile(`^full|centered$`).Match([]byte(contentLayout)) {
		uiSettings.ContentLayout = contentLayout
		localSess.SetValue("page_content_layout", contentLayout)
	}

	oneSess := mr.Session("one", w, r)
	oneSess.Raw.AddFlash(mr.i18nCustom.MustLocalize("UISaveSuccess", "", 0))
	oneSess.Raw.Save(r, w)
	// fmt.Println("uiSettings after post: ", uiSettings)

	ctx := context.WithValue(r.Context(), "ui_settings", uiSettings)

	http.Redirect(w, r.WithContext(ctx), "/settings/ui", http.StatusFound)
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
	PageKey         SettingsPageKey
	AccountData     *model.User
	LanguageOptions []*model.OptionItem
}

func (mr *MainResource) handleSettingsPage(w http.ResponseWriter, r *http.Request, pageKey SettingsPageKey) {
	settingsTitleMap := map[SettingsPageKey]string{
		SettingsPageKeyUI:      mr.i18nCustom.MustLocalize("UI", "", ""),
		SettingsPageKeyAccount: mr.i18nCustom.MustLocalize("Account", "", ""),
	}

	var langStrEnums []model.StringEnum
	for _, item := range model.LangValues() {
		langStrEnums = append(langStrEnums, item)
	}
	langOptions := model.ConvertEnumToOPtions(langStrEnums, true, "", nil)
	pageData := &SettingsPageData{
		PageKey:         pageKey,
		LanguageOptions: langOptions,
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
	settingsText := mr.i18nCustom.MustLocalize("Settings", "", 2)
	mr.Render(w, r, "settings", &model.PageData{
		Title: settingsTitleMap[pageKey] + " " + settingsText,
		Data:  pageData,
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: "/settings",
				Name: settingsText,
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
		user.Sanitize(mr.sanitizePolicy)
		_, err := mr.store.User.Update(user, []string{"Introduction"})
		if err != nil {
			mr.Error("", errors.WithStack(err), w, r, http.StatusInternalServerError)
		}

		oneSess := mr.Session("one", w, r)
		oneSess.Raw.AddFlash(mr.i18nCustom.MustLocalize("AccountSaveSuccess", "", ""))
		oneSess.Raw.Save(r, w)

		http.Redirect(w, r, "/settings/account", http.StatusFound)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}
