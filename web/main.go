package web

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/oodzchen/dproject/config"
	mdw "github.com/oodzchen/dproject/middleware"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/service"
	"github.com/oodzchen/dproject/utils"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// https://www.postgresql.org/docs/current/errcodes-appendix.html
const PGErrUniqueViolation = "23505"

var (
	// ErrCodeVerifyExpired = errors.New("verification code is expired")
	ErrCodeVerifyIncorrect = errors.New("verification code is incorrect")
)

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
	rt.Get("/register_verify", mr.VerifyRegisterPage)
	rt.With(mdw.UserLogger(
		mr.uLogger, model.AcTypeUser, model.AcActionRegisterVerify, model.AcModelEmpty, mdw.ULogEmpty),
	).Post("/register_verify", mr.VerifyRegister)
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

	rt.Get("/send_code", mr.ResendVerifyEmail)
	rt.Get("/retrieve_password", mr.RetrievePasswordPage)
	rt.With(mdw.UserLogger(
		mr.uLogger, model.AcTypeUser, model.AcActionRetrievePassword, model.AcModelEmpty, mdw.ULogEmpty),
	).Post("/retrieve_password", mr.RetrievePassword)
	rt.Get("/reset_password", mr.ResetPasswordPage)
	rt.With(mdw.UserLogger(
		mr.uLogger, model.AcTypeUser, model.AcActionResetPassword, model.AcModelEmpty, mdw.ULogEmpty),
	).Post("/reset_password", mr.ResetPassword)

	rt.Route("/settings", func(r chi.Router) {
		r.Get("/", mr.SettingsPage)
		r.With(mdw.AuthCheck(mr.sessStore)).Get("/account", mr.SettingsAccountPage)

		r.With(mdw.AuthCheck(mr.sessStore), mdw.PermitCheck(mr.srv.Permission, []string{
			"user.update_intro_mine",
			// "user.update_intro_others",
		}, mr),
			mdw.UserLogger(mr.uLogger, model.AcTypeUser, model.AcActionUpdateIntro, model.AcModelEmpty, mdw.ULogEmpty),
		).Post("/account", mr.SaveAccountSettings)
		r.Get("/ui", mr.SettingsUIPage)
		r.Post("/ui", mr.SaveUISettings)
	})

	rt.Get("/login_auth", mr.LoginWithOAuth)
	rt.Get("/auth_callback", mr.LoginAuthCallback)
	rt.Get("/auth_callback_github", mr.LoginAuthCallbackGithub)

	rt.With(mdw.AuthCheck(mr.sessStore)).Get("/messages", mr.MessageList)

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
				Name: mr.Local("Register"),
			},
		},
	})
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

	user.TrimSpace()

	if user.Name == "" {
		user.Name = model.ExtractNameFromEmail(user.Email)
	}

	user.Sanitize(mr.sanitizePolicy)
	err := user.Valid(true)
	if err != nil {
		errStr := err.Error()
		if errors.Is(err, model.ErrEmailValidFailed) {
			errStr = model.ErrEmailValidFailed.Error()
		}
		mr.Error(errStr, err, w, r, http.StatusBadRequest)
		return
	}

	// userData, err := mr.store.User.ItemWithEmail(email)
	userId, err := mr.store.User.Exists(user.Email, user.Name)
	if err != nil && err != pgx.ErrNoRows {
		mr.ServerErrorp("", err, w, r)
		return
	} else {
		if userId > 0 {
			// alreadyExistsTip := mr.Local("AlreadyExists", "FieldNames", mr.Local("Or", "A", mr.Local("Username"), "B", mr.Local("Email")))
			mr.Error(model.ErrEmailValidFailed.Error(), err, w, r, http.StatusBadRequest)
			return
		}
	}

	// log.Printf("user model is %v", user)

	err = user.EncryptPassword()
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	err = mr.rdb.Set(context.Background(), "register_pass_"+user.Email, user.Password, service.DefaultCodeLifeTime).Err()
	if err != nil {
		mr.Error(err.Error(), err, w, r, http.StatusBadRequest)
		return
	}
	mr.Session("one", w, r).SetValue("register_email", user.Email)
	mr.Session("one", w, r).SetValue("register_username", user.Name)

	go mr.sendVerifyCode(email, service.VerifCodeRegister, w, r)

	http.Redirect(w, r, "/register_verify", http.StatusFound)
}

func (mr *MainResource) VerifyRegisterPage(w http.ResponseWriter, r *http.Request) {
	email := mr.Session("one", w, r).GetStringValue("register_email")
	if email == "" {
		mr.Error("", errors.New("get register email failed"), w, r, http.StatusInternalServerError)
		return
	}

	username := mr.Session("one", w, r).GetStringValue("register_username")
	if username == "" {
		mr.Error("", errors.New("get register username failed"), w, r, http.StatusInternalServerError)
		return
	}

	type PageData struct {
		Email        string
		Username     string
		CodeLifeTime int
	}
	if IsLogin(mr.sessStore, w, r) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	mr.Render(w, r, "register_verify", &model.PageData{
		Title: mr.Local("EmailVerify") + " - " + mr.Local("Register"),
		Data: &PageData{
			Email: email,
			// Username: username,
			CodeLifeTime: int(service.DefaultCodeLifeTime.Minutes()),
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Name: mr.Local("EmailVerify"),
			},
		},
	})
}

func (mr *MainResource) ResendVerifyEmail(w http.ResponseWriter, r *http.Request) {
	codeType := service.VerifCodeType(r.URL.Query().Get("type"))
	if _, ok := service.VerifCodeTypeMap[codeType]; !ok {
		mr.Error("", errors.New("code type is invalid"), w, r, http.StatusBadRequest)
		return
	}

	var email string
	var redirectPath string
	switch codeType {
	case service.VerifCodeRegister:
		email = mr.Session("one", w, r).GetStringValue("register_email")
		username := mr.Session("one", w, r).GetStringValue("register_username")
		if username == "" {
			mr.Error("", errors.New("get register username failed"), w, r, http.StatusInternalServerError)
			return
		}

		redirectPath = "/register_verify"
	case service.VerifCodeResetPassword:
		email = mr.Session("one", w, r).GetStringValue("email_reset_pass")
		redirectPath = "reset_password"
	}

	if email == "" {
		mr.Error("", errors.New("get email failed"), w, r, http.StatusInternalServerError)
		return
	}

	if redirectPath == "" {
		mr.Error("", errors.New("redirect path is empty"), w, r, http.StatusInternalServerError)
		return
	}

	var isRegistered bool
	if userData, _ := mr.store.User.ItemWithEmail(email); userData != nil && userData.Id > 0 {
		isRegistered = true
	}

	switch codeType {
	case service.VerifCodeRegister:
		if !isRegistered {
			go mr.sendVerifyCode(email, codeType, w, r)
		}
	case service.VerifCodeResetPassword:
		if isRegistered {
			go mr.sendVerifyCode(email, codeType, w, r)
		}
	}

	http.Redirect(w, r, redirectPath, http.StatusFound)
}

func (mr *MainResource) sendVerifyCode(email string, codeType service.VerifCodeType, w http.ResponseWriter, r *http.Request) {
	verifier := mr.srv.Verifier
	code := verifier.GenCode()
	encryptCode, err := verifier.EncryptCode(code)
	if err != nil {
		fmt.Println("encrypt verification code error: ", err)
		return
	}
	// fmt.Println("saved code: ", encryptCode)

	err = verifier.SaveCode(email, encryptCode, codeType)
	if err != nil {
		fmt.Println("save verification code error: ", err)
		return
	}

	// savedCode, err := verifier.GetCode(email)
	// if err != nil {
	// 	fmt.Println("get verificaiton code error: ", err)
	// }
	// fmt.Println("get code from redis: ", savedCode)

	err = mr.srv.Mail.SendVerificationCode(email, code, codeType)
	if err != nil {
		fmt.Println("send verification code error: ", err)
		return
	}
}

func (mr *MainResource) VerifyRegister(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.Form.Get("code"))
	if code == "" {
		mr.Error("lack of code", errors.New("lack of code"), w, r, http.StatusBadRequest)
		return
	}
	// email := strings.TrimSpace(r.Form.Get("email"))
	email := mr.Session("one", w, r).GetStringValue("register_email")
	if email == "" {
		mr.Error("", errors.New("get register email failed"), w, r, http.StatusInternalServerError)
		return
	}

	err := mr.verifyMailCode(email, code, service.VerifCodeRegister, w, r)
	if err != nil {
		return
	}

	password, err := mr.rdb.Get(context.Background(), "register_pass_"+email).Result()
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}
	go func() {
		err = mr.rdb.Del(context.Background(), "register_pass"+email).Err()
		if err != nil {
			fmt.Println(errors.Join(err, errors.New("redis delete register password failed")))
		}
	}()

	username := mr.Session("one", w, r).GetStringValue("register_username")
	if username == "" {
		mr.Error("", errors.New("get register username failed"), w, r, http.StatusInternalServerError)
		return
	}

	_, err = mr.userSrv.Register(email, password, username)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.Is(err, model.AppErrUserValidFailed) {
			mr.Error(err.Error(), err, w, r, http.StatusBadRequest)
		} else if errors.As(err, &pgErr) && pgErr.Code == PGErrUniqueViolation {
			alreadyExistsTip := mr.Local("AlreadyExists", "FieldNames", mr.Local("Or", "A", mr.Local("Username"), "B", mr.Local("Email")))
			mr.Error(alreadyExistsTip, err, w, r, http.StatusBadRequest)
		} else {
			mr.Error("", err, w, r, http.StatusInternalServerError)
		}

		return
	}

	mr.Session("one", w, r).SetValue("register_email", "")
	mr.Session("one", w, r).SetValue("register_username", "")

	mr.Session("one", w, r).Flash(mr.i18nCustom.MustLocalize("AccountCreateSuccess", "", ""))

	http.Redirect(w, r, "/login", http.StatusFound)
}

func (mr *MainResource) verifyMailCode(email, code string, codeType service.VerifCodeType, w http.ResponseWriter, r *http.Request) error {
	if (config.Config.Debug || config.Config.Testing) && code == config.SuperCode {
		return nil
	}

	savedCode, err := mr.srv.Verifier.GetCode(email, codeType)
	// fmt.Println("savedCode: ", savedCode)
	// fmt.Println("err: ", err)

	if err != nil {
		if errors.Is(err, redis.Nil) {
			mr.Error(mr.Local("VerificationExpired"), err, w, r, http.StatusBadRequest)
		} else {
			mr.ServerErrorp("", errors.Join(err, errors.New("get code failed")), w, r)
		}
		return err
	}

	err = mr.srv.Verifier.VerifyCode(code, savedCode)
	if err != nil {
		mr.Error(mr.Local("VerificationIncorrect"), err, w, r, http.StatusBadRequest)
		return errors.Join(ErrCodeVerifyIncorrect, err)
	}

	go func() {
		err = mr.srv.Verifier.DeleteCode(email, codeType)
		if err != nil {
			fmt.Println(errors.Join(err, errors.New("redis delete verification code failed")))
		}
	}()

	return nil
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
				Name: mr.Local("Login"),
			},
		},
	})
}

func (mr *MainResource) Login(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.PostFormValue("username"))
	password := strings.TrimSpace(r.PostFormValue("password"))

	mr.doLogin(w, r, username, password)

	mr.ToTargetUrl(w, r)
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

	// id, err := mr.store.User.Login(username, password)
	hashedPwd, err := mr.store.User.GetPassword(username)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			notRegisterdTip := mr.Local("NotRegistered", "FieldNames", mr.Local("Or", "A", mr.Local("Username"), "B", mr.Local("Email")))
			mr.Error(notRegisterdTip, err, w, r, http.StatusBadRequest)
		} else {
			mr.Error(loginFailedTip, err, w, r, http.StatusBadRequest)
		}

		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(password))
	if err != nil {
		mr.Error(loginFailedTip, err, w, r, http.StatusBadRequest)
		return
	}

	user, err := mr.store.User.ItemWithUsernameEmail(username)
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
	}

	mr.saveUserInfo(w, r, user)
}

func (mr *MainResource) doLoginOAuth(w http.ResponseWriter, r *http.Request, email string) {
	if email == "" {
		mr.Error(mr.Local("Required", "FieldNames", mr.Local("Email")), nil, w, r, http.StatusBadRequest)
		return
	}

	if err := model.ValidateEmail(email); err != nil {
		emailValidTip := mr.Local("Incorrect", "FieldNames", mr.Local("Or", "A", mr.Local("Email"), "B", mr.Local("Password")))
		mr.Error(emailValidTip, err, w, r, http.StatusBadRequest)
		return
	}

	user, err := mr.store.User.ItemWithEmail(email)
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
	}

	mr.saveUserInfo(w, r, user)
}

func (mr *MainResource) saveUserInfo(w http.ResponseWriter, r *http.Request, user *model.User) {
	mr.srv.Permission.SetLoginedUser(user)

	sess, err := mr.sessStore.Get(r, "one")
	if err != nil {
		logSessError("one", err)
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

func (mr *MainResource) doRegisterWithOAuth(w http.ResponseWriter, r *http.Request, email string, authType model.AuthType) {
	if email == "" {
		mr.Error(mr.Local("Required", "FieldNames", mr.Local("Email")), nil, w, r, http.StatusBadRequest)
		return
	}

	if err := model.ValidateEmail(email); err != nil {
		emailValidTip := mr.Local("Incorrect", "FieldNames", mr.Local("Or", "A", mr.Local("Email"), "B", mr.Local("Password")))
		mr.Error(emailValidTip, err, w, r, http.StatusBadRequest)
		return
	}

	username := model.ExtractNameFromEmail(email)

	_, err := mr.store.User.CreateWithOAuth(email, username, string(model.DefaultUserRoleCommon), string(authType))
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}

	mr.doLoginOAuth(w, r, email)

	mr.ToTargetUrl(w, r)
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
	ClearSession(mr.sessStore, w, r)

	mr.srv.Permission.ResetPermissionData()
}

func (mr *MainResource) SaveUISettings(w http.ResponseWriter, r *http.Request) {
	lang := r.PostForm.Get("lang")
	theme := r.PostForm.Get("theme")
	contentLayout := r.PostForm.Get("content_layout")
	fontSizeStr := r.PostForm.Get("font_size")
	fontSizeCustomStr := r.PostForm.Get("font_size_custom")

	uiSettings := &model.UISettings{}

	localSess := mr.Session("local", w, r)
	if lang, err := model.ParseLang(lang); err == nil {
		// fmt.Println("post lang: ", lang)
		uiSettings.Lang = lang
		mr.i18nCustom.SwitchLang(string(lang))
		localSess.SetValue("lang", lang)
		model.UpdateErrI18n()
	}

	if regexp.MustCompile(`^light|dark|system$`).Match([]byte(theme)) {
		uiSettings.Theme = theme
		localSess.SetValue("page_theme", theme)
	}

	if regexp.MustCompile(`^full|centered$`).Match([]byte(contentLayout)) {
		uiSettings.ContentLayout = contentLayout
		localSess.SetValue("page_content_layout", contentLayout)
	}

	var fontSize int
	if strings.TrimSpace(fontSizeStr) == "x" {
		fontSize, _ = strconv.Atoi(fontSizeCustomStr)
		uiSettings.FontSizeCustom = true
		localSess.SetValue("font_size_custom", true)
	} else {
		fontSize, _ = strconv.Atoi(fontSizeStr)
		uiSettings.FontSizeCustom = false
		localSess.SetValue("font_size_custom", false)
	}

	if fontSize < 10 {
		mr.Error("", nil, w, r, http.StatusBadRequest)
		return
	}

	uiSettings.FontSize = fontSize
	localSess.SetValue("font_size", fontSize)

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
		SettingsPageKeyUI:      mr.Local("UI"),
		SettingsPageKeyAccount: mr.Local("Account"),
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
				mr.Error("", err, w, r, http.StatusInternalServerError)
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
			mr.Error("", err, w, r, http.StatusInternalServerError)
		}

		oneSess := mr.Session("one", w, r)
		oneSess.Raw.AddFlash(mr.i18nCustom.MustLocalize("AccountSaveSuccess", "", ""))
		oneSess.Raw.Save(r, w)

		http.Redirect(w, r, "/settings/account", http.StatusFound)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

type MessageStatus string

const (
	MessageStatusUnread MessageStatus = "unread"
	MessageStatusRead                 = "read"
	MessageStatusAll                  = "all"
)

var MessageStatusMap = map[MessageStatus]bool{
	MessageStatusUnread: true,
	MessageStatusRead:   true,
	MessageStatusAll:    true,
}

type MessageQueryData struct {
	*queryData
	Tab string
}
type MessagePageData struct {
	List  []*model.Message
	Query *MessageQueryData
}

func (mr *MainResource) MessageList(w http.ResponseWriter, r *http.Request) {
	tab := strings.TrimSpace(r.URL.Query().Get("tab"))

	page, pageSize := mr.GetPaginationData(r)

	if _, ok := MessageStatusMap[MessageStatus(tab)]; !ok {
		tab = string(MessageStatusAll)
	}

	userId := mr.GetLoginedUserId(w, r)

	list, total, err := mr.store.Message.List(userId, tab, page, pageSize)
	if err != nil {
		mr.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	var messageIds []any
	for _, item := range list {
		item.SourceArticle.FormatDeleted()
		item.SourceArticle.UpdateDisplayTitle()
		messageIds = append(messageIds, item.Id)
	}

	if len(messageIds) > 0 {
		err = mr.store.Message.ReadMany(messageIds)
		if err != nil {
			mr.Error("", err, w, r, http.StatusInternalServerError)
			return
		}
	}

	title := mr.Local("List", "Name", mr.Local("Message"))

	mr.Render(w, r, "message", &model.PageData{
		Title: title,
		Data: &MessagePageData{
			List: list,
			Query: &MessageQueryData{
				&queryData{
					Total: total, Page: page, TotalPage: CeilInt(total, pageSize),
				},
				tab,
			},
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Name: title,
			},
		},
	})
}

type AuthType string

const (
	AuthTypeGoogle    AuthType = "google"
	AuthTypeGithub             = "github"
	AuthTypeMicrosoft          = "microsoft"
)

var authTypeMap = map[AuthType]bool{
	AuthTypeGoogle:    true,
	AuthTypeGithub:    true,
	AuthTypeMicrosoft: true,
}

func (mr *MainResource) LoginWithOAuth(w http.ResponseWriter, r *http.Request) {
	authType := AuthType(r.URL.Query().Get("type"))
	if _, ok := authTypeMap[authType]; !ok {
		mr.Error("", errors.New("auth type not exist"), w, r, http.StatusBadRequest)
		return
	}

	var authUrl string
	switch authType {
	case AuthTypeGoogle:
		authUrl = getGoogleAuthUrl()
	case AuthTypeGithub:
		authUrl = getGithubAuthUrl()
	}

	state, err := genCSRFToken()
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}

	mr.Session("one", w, r).SetValue("auth_state", state)

	authUrl += fmt.Sprintf("&state=%s", state)

	// fmt.Println("authUrl", authUrl)

	http.Redirect(w, r, authUrl, http.StatusFound)
}

func getGoogleAuthUrl() string {
	return fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=%s&scope=%s",
		config.Config.GoogleClientID,
		config.Config.GetServerURL()+"/auth_callback",
		"code",
		"https://www.googleapis.com/auth/userinfo.email",
	)
}

func getGithubAuthUrl() string {
	return fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s",
		config.Config.GithubClientID,
		config.Config.GetServerURL()+"/auth_callback_github",
		"user:email",
	)
}

func genCSRFToken() (string, error) {
	randomData := make([]byte, 1024)
	// fmt.Println("randomData: ", randomData)

	_, err := rand.Read(randomData)
	if err != nil {
		fmt.Println("read random data failed", err)
		return "", err
	}

	// fmt.Println("randomData after read: ", randomData)

	hash := sha256.New()
	hash.Write(randomData)
	hashInBytes := hash.Sum(nil)

	hashString := hex.EncodeToString(hashInBytes)

	return hashString, nil
}

type GoogleTokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresInSeconds int    `json:"expires_in"`
	Scope            string `json:"scope"`
	TokenType        string `json:"Bearer"`
	IdToken          string `json:"id_token"`
}

type GoogleUserInfo struct {
	SubId         string `json:"sub"`
	Picture       string `json:"picture"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

func (mr *MainResource) LoginAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	savedState := mr.Session("one", w, r).GetStringValue("auth_state")
	// fmt.Println("saved state: ", savedState)
	if state == "" || state != savedState {
		mr.Error("state is incorrect", errors.New("state is incorrect"), w, r, http.StatusBadRequest)
		return
	}

	if code == "" {
		mr.Error("", errors.New("code is required"), w, r, http.StatusBadRequest)
		return
	}

	// fmt.Println("google auth code: ", code)
	// fmt.Fprintf(w, "google auth code: %s", code)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	payload := []byte(`{
		"client_id":"` + config.Config.GoogleClientID + `",
		"client_secret":"` + config.Config.GoogleClientSecret + `",
		"code":"` + code + `",
		"grant_type": "authorization_code",
		"redirect_uri": "` + config.Config.GetServerURL() + `/auth_callback",
	}`)

	// fmt.Println("payload: ", string(payload))

	req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/token", bytes.NewBuffer(payload))
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}
	defer resp.Body.Close()

	// fmt.Println("Response status: ", resp.Status)

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	// fmt.Println("response body: ", buf.String())

	var tokenData GoogleTokenResponse
	err = json.Unmarshal(buf.Bytes(), &tokenData)
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}
	// fmt.Println("tokenData: ", tokenData)

	if tokenData.AccessToken == "" {
		fmt.Println("google token response body: ", buf.String())
		mr.Error("get access token failed", errors.Join(err, errors.New("get google access token failed")), w, r, http.StatusBadRequest)
		return
	}

	// err := mr.rdb.Set(context.Background(), "google_token", value interface{}, expiration time.Duration)
	req, err = http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)

	resp, err = client.Do(req)
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}
	defer resp.Body.Close()

	// fmt.Println("user info response status: ", resp.Status)

	buf = new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	// fmt.Println("user info response body: ", buf.String())

	var userInfo GoogleUserInfo
	err = json.Unmarshal(buf.Bytes(), &userInfo)
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}

	if userInfo.Email == "" {
		mr.Error("can't get the email", err, w, r, http.StatusBadRequest)
		return
	}

	if !userInfo.EmailVerified {
		mr.Error("email is not verified", err, w, r, http.StatusBadRequest)
		return
	}

	// userId, err := mr.store.User.Exists(userInfo.Email, "")
	userData, err := mr.store.User.ItemWithEmail(userInfo.Email)
	if err != nil {
		if errors.Is(err, model.AppErrUserNotExist) {
			mr.doRegisterWithOAuth(w, r, userInfo.Email, model.AuthTypeGoogle)
		} else {
			mr.ServerErrorp("", err, w, r)
		}
		return
	}

	// fmt.Printf("user data: %#v", userData)

	if userData.AuthFrom == model.AuthTypeGoogle {
		mr.doLoginOAuth(w, r, userInfo.Email)
		mr.ToTargetUrl(w, r)
	} else {
		mr.Session("one", w, r).Flash(mr.Local("AcountExistsTip"))
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func (mr *MainResource) RetrievePasswordPage(w http.ResponseWriter, r *http.Request) {
	mr.Render(w, r, "retrieve_password", &model.PageData{
		Title: "Retrieve Password",
	})
}

func (mr *MainResource) RetrievePassword(w http.ResponseWriter, r *http.Request) {
	email := r.Form.Get("email")

	if email == "" {
		mr.Error(mr.Local("Required", "FieldNames", mr.Local("Email")), nil, w, r, http.StatusBadRequest)
		return
	}

	if err := model.ValidateEmail(email); err != nil {
		emailValidTip := mr.Local("Incorrect", "FieldNames", mr.Local("Or", "A", mr.Local("Email"), "B", mr.Local("Password")))
		mr.Error(emailValidTip, err, w, r, http.StatusBadRequest)
		return
	}

	mr.Session("one", w, r).SetValue("email_reset_pass", email)

	if userData, _ := mr.store.User.ItemWithEmail(email); userData != nil && userData.Id > 0 {
		go mr.sendVerifyCode(email, service.VerifCodeResetPassword, w, r)
	}

	http.Redirect(w, r, "/reset_password", http.StatusFound)
}

func (mr *MainResource) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	var email string
	userData := r.Context().Value("user_data")

	if v, ok := userData.(model.User); ok {
		email = v.Email
	} else {
		email = strings.TrimSpace(mr.Session("one", w, r).GetStringValue("email_reset_pass"))
	}

	if email == "" {
		mr.Error(mr.Local("Required", "FieldNames", mr.Local("Email")), nil, w, r, http.StatusBadRequest)
		return
	}

	type PageData struct {
		Email        string
		CodeLifeTime int
	}

	mr.Render(w, r, "reset_password", &model.PageData{
		Title: "Reset Password",
		Data: &PageData{
			Email:        email,
			CodeLifeTime: int(mr.srv.Verifier.CodeLifeTime.Minutes()),
		},
	})
}

func (mr *MainResource) ResetPassword(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.Form.Get("code"))
	if code == "" {
		mr.Error("lack of code", errors.New("lack of code"), w, r, http.StatusBadRequest)
		return
	}
	// email := strings.TrimSpace(r.Form.Get("email"))
	email := mr.Session("one", w, r).GetStringValue("email_reset_pass")
	if email == "" {
		mr.Error("", errors.New("get email failed"), w, r, http.StatusInternalServerError)
		return
	}

	err := mr.verifyMailCode(email, code, service.VerifCodeResetPassword, w, r)
	if err != nil {
		return
	}

	password := r.Form.Get("password")
	confirmPassword := r.Form.Get("confirm-password")

	if password == "" || confirmPassword == "" {
		mr.Error(mr.Local("Required", "FieldNames", mr.Local("Password")), errors.New("password is empty"), w, r, http.StatusBadRequest)
		return
	}

	err = model.ValidPassword(password)
	if err != nil {
		mr.Error(mr.Local("FormatError", "FieldNames", mr.Local("Password")), errors.New("password format error"), w, r, http.StatusBadRequest)
		return
	}

	if confirmPassword != password {
		mr.Error(mr.Local("PasswordConfirmError"), errors.New("password confirm error"), w, r, http.StatusBadRequest)
		return
	}

	encryptPassword, err := model.DoEncryptPassword(confirmPassword)
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}

	// fmt.Println("email: ", email)

	_, err = mr.store.User.UpdatePassword(email, encryptPassword)
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}

	mr.Session("one", w, r).SetValue("target_url", "/")
	mr.Session("one", w, r).SetValue("email_reset_pass", "")
	mr.Session("one", w, r).Flash(mr.Local("PassResetSuccess"))

	http.Redirect(w, r, "/login", http.StatusFound)
}

type GithubTokenResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	TokenType   string `json:"bearer"`
}

type GithubUserInfo struct {
	LoginId   string `json:"login"`
	AvatarUrl string `json:"avatar_url"`
	HtmlUrl   string `json:"html_url"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

func (mr *MainResource) LoginAuthCallbackGithub(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")

	savedState := mr.Session("one", w, r).GetStringValue("auth_state")
	// fmt.Println("saved state: ", savedState)
	if state == "" || state != savedState {
		mr.Error("state is incorrect", errors.New("state is incorrect"), w, r, http.StatusBadRequest)
		return
	}

	if code == "" {
		mr.Error("", errors.New("code is required"), w, r, http.StatusBadRequest)
		return
	}

	// fmt.Println("github auth code: ", code)
	// fmt.Fprintf(w, "github auth code: %s", code)

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	payload := []byte(`{
		"client_id":"` + config.Config.GithubClientID + `",
		"client_secret":"` + config.Config.GithubClientSecret + `",
		"code":"` + code + `",
                "redirect_uri": "` + config.Config.GetServerURL() + `/auth_callback_github"
	}`)

	// fmt.Println("payload: ", string(payload))

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(payload))
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}
	defer resp.Body.Close()

	// fmt.Println("github token response status: ", resp.Status)

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	// fmt.Println("github token response body: ", buf.String())

	// fmt.Fprint(w, buf.String())

	var tokenData GithubTokenResponse
	err = json.Unmarshal(buf.Bytes(), &tokenData)
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}

	if tokenData.AccessToken == "" {
		fmt.Println("github token response body: ", buf.String())
		mr.Error("get access token failed", errors.Join(err, errors.New("get github access token failed")), w, r, http.StatusBadRequest)
		return
	}
	// fmt.Println("tokenData: ", tokenData)

	req, err = http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenData.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err = client.Do(req)
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}
	defer resp.Body.Close()

	// fmt.Println("user info response status: ", resp.Status)

	buf = new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	// fmt.Println("user info response body: ", buf.String())
	// fmt.Fprint(w, buf.String())

	var userInfo GithubUserInfo
	err = json.Unmarshal(buf.Bytes(), &userInfo)
	if err != nil {
		mr.ServerErrorp("", err, w, r)
		return
	}

	// fmt.Printf("response user info: %#v", userInfo)

	if userInfo.Email == "" {
		mr.Error("can't get the email, this may cause by your github privacy setting", err, w, r, http.StatusBadRequest)
		return
	}

	userData, err := mr.store.User.ItemWithEmail(userInfo.Email)
	if err != nil {
		if errors.Is(err, model.AppErrUserNotExist) {
			mr.doRegisterWithOAuth(w, r, userInfo.Email, model.AuthTypeGithub)
		} else {
			mr.ServerErrorp("", err, w, r)
		}
		return
	}

	// fmt.Printf("user data: %#v", userData)

	if userData.AuthFrom == model.AuthTypeGithub {
		mr.doLoginOAuth(w, r, userInfo.Email)
		mr.ToTargetUrl(w, r)
	} else {
		mr.Session("one", w, r).Flash(mr.Local("AcountExistsTip"))
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}
