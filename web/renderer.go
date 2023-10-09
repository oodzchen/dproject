package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/microcosm-cc/bluemonday"
	"github.com/oodzchen/dproject/config"
	i18nc "github.com/oodzchen/dproject/i18n"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/service"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/utils"
	"github.com/pkg/errors"
)

type Renderer struct {
	tmpl      *template.Template
	sessStore *sessions.CookieStore
	router    *chi.Mux
	store     *store.Store
	// permission *config.PermissionData
	permissionSrv  *service.Permission
	uLogger        *service.UserLogger
	sanitizePolicy *bluemonday.Policy
	i18nCustom     *i18nc.I18nCustom
}

func NewRenderer(tmpl *template.Template, sessStore *sessions.CookieStore, router *chi.Mux, store *store.Store, permissionSrv *service.Permission, userLogger *service.UserLogger, sp *bluemonday.Policy, ic *i18nc.I18nCustom) *Renderer {
	return &Renderer{
		tmpl,
		sessStore,
		router,
		store,
		permissionSrv,
		userLogger,
		sp,
		ic,
	}
}

func (rd *Renderer) Render(w http.ResponseWriter, r *http.Request, name string, data *model.PageData) {
	rd.doRender(w, r, name, data, http.StatusOK)
}

func (rd *Renderer) ServerErrorp(msg string, err error, w http.ResponseWriter, r *http.Request) {
	rd.Error(msg, err, w, r, http.StatusInternalServerError)
}

func (rd *Renderer) ToLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (rd *Renderer) NotFound(w http.ResponseWriter, r *http.Request) {
	rd.Error("", nil, w, r, http.StatusNotFound)
}

func (rd *Renderer) ServerError(w http.ResponseWriter, r *http.Request) {
	rd.Error("", nil, w, r, http.StatusInternalServerError)
}

func (rd *Renderer) Forbidden(w http.ResponseWriter, r *http.Request) {
	rd.Error("", nil, w, r, http.StatusForbidden)
}

func (rd *Renderer) GetLoginedUserId(w http.ResponseWriter, r *http.Request) int {
	userId := rd.Session("one", w, r).GetValue("user_id")
	if userId, ok := userId.(int); ok {
		return userId
	}
	return 0
}

func (rd *Renderer) Error(msg string, err error, w http.ResponseWriter, r *http.Request, code int) {
	fmt.Printf("render err: %+v\n", err)
	fmt.Println("msg: ", msg)

	referer := r.Referer()
	refererUrl, _ := url.Parse(referer)
	prevUrl := ""

	if IsRegisterdPage(refererUrl, rd.router) {
		prevUrl = referer
	}

	errText := http.StatusText(code)

	type errPageData struct {
		HttpStatusCode int
		AppErrCode     model.AppErrCode
		ErrCode        int
		ErrText        string
		PrevUrl        string
	}
	var pageData errPageData
	pageData = errPageData{0, 0, 0, "", prevUrl}

	data := &model.PageData{
		Title: errText,
		Data:  &pageData,
	}

	if len(msg) > 0 {
		text := []rune(msg)
		errText += " - " + strings.ToUpper(string(text[:1])) + string(text[1:])
	}
	pageData.ErrText = errText

	if err, ok := err.(model.AppError); ok {
		pageData.ErrCode = int(err.ErrCode)
	}

	pageData.HttpStatusCode = code
	rd.doRender(w, r, "error", data, code)
}

func (rd *Renderer) doRender(w http.ResponseWriter, r *http.Request, name string, data *model.PageData, code int) {
	sess := rd.Session("one", w, r).Raw

	if flashes := sess.Flashes(); len(flashes) > 0 {
		for _, item := range flashes {
			if value, ok := item.(string); ok {
				data.TipMsg = append(data.TipMsg, value)
			}
		}
	}

	if userData, ok := r.Context().Value("user_data").(*model.User); ok {
		data.LoginedUser = userData
	}

	if uiSettings, ok := r.Context().Value("ui_settings").(*model.UISettings); ok {
		// fmt.Println("uiSettings in renderer: ", uiSettings)
		data.UISettings = uiSettings
	}

	err := sess.Save(r, w)
	if err != nil {
		fmt.Printf("session save error: %+v", err)
	}
	// fmt.Println("currLang: ", rd.i18nCustom.CurrLang)

	data.CSRFField = string(csrf.TemplateField(r))
	data.RoutePath = r.URL.Path
	data.Debug = config.Config.Debug
	data.BrandName = config.Config.BrandName
	data.BrandDomainName = config.Config.BrandDomainName
	data.Slogan = config.Config.Slogan
	data.PermissionEnabledList = rd.permissionSrv.PermissionData.EnabledFrondIdList

	rd.tmpl = rd.tmpl.Funcs(template.FuncMap{
		"permit":  rd.permissionSrv.PermissionData.Permit,
		"local":   rd.i18nCustom.LocalTpl,
		"timeAgo": rd.i18nCustom.TimeAgo.Format,
	})

	if data.Title != "" {
		data.Title += fmt.Sprintf(" - %s", config.Config.BrandName)
	} else {
		data.Title = fmt.Sprintf("%s", config.Config.BrandName)
	}

	if data.Debug {
		users, err := rd.store.User.List(1, 50, true)
		if err != nil {
			fmt.Println("get debug user data error: ", err)
		}
		data.DebugUsers = users

		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			HttpError("", errors.WithStack(err), w, http.StatusInternalServerError)
			return
		}

		data.JSONStr = string(jsonData)
	}

	header := w.Header()

	contentSecurity := []string{
		"default-src 'self'",
		"img-src 'self' https://*",
		"style-src 'self' 'unsafe-inline'",
		"child-src 'none'",
	}

	if data.Debug {
		contentSecurity = append(contentSecurity, "script-src 'self' 'unsafe-inline'")
	}
	// header.Set("Content-Type", "text/html")
	header.Add("Content-Security-Policy", strings.Join(contentSecurity, ";"))

	// fmt.Println("header", header.Values("Content-Security-Policy"))
	w.WriteHeader(code)

	err = rd.tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		HttpError("", errors.WithStack(err), w, http.StatusInternalServerError)
	}
}

func (rd *Renderer) getUserPermittedFrontIds(r *http.Request) []string {
	var frontIdList []string
	if userData, ok := r.Context().Value("user_data").(*model.User); ok {
		if userData.Super {
			return rd.permissionSrv.PermissionData.EnabledFrondIdList
		}

		if userData.Permissions != nil && len(userData.Permissions) > 0 {

			for _, item := range userData.Permissions {
				frontIdList = append(frontIdList, item.FrontId)
			}

			return frontIdList
		}
	}
	return nil
}

func (rd *Renderer) Session(name string, w http.ResponseWriter, r *http.Request) *Session {
	sess, err := rd.sessStore.Get(r, name)
	if err != nil {
		logSessError(name, errors.WithStack(err))
		ClearSession(sess, w, r)
	}
	return &Session{rd, sess, w, r}
}

func (rd *Renderer) ToRefererUrl(w http.ResponseWriter, r *http.Request) {
	targetUrl := "/"
	refererUrl, err := url.Parse(r.Referer())
	if err != nil {
		http.Redirect(w, r, targetUrl, http.StatusFound)
		return
	}

	if IsRegisterdPage(refererUrl, rd.router) {
		// fmt.Println("Matched!")
		targetUrl = r.Referer()
	}

	http.Redirect(w, r, targetUrl, http.StatusFound)
}

func (rd *Renderer) SavePrevPage(w http.ResponseWriter, r *http.Request) {
	referer := r.Referer()
	refererUrl, _ := url.Parse(referer)

	if refererUrl != nil && IsRegisterdPage(refererUrl, rd.router) {
		rd.Session("one", w, r).SetValue("prev_url", referer)
	}
}

func (rd *Renderer) ToPrevPage(w http.ResponseWriter, r *http.Request) {
	prevPgaeUrl := rd.Session("one", w, r).GetStringValue("prev_url")
	if prevPgaeUrl != "" {
		http.Redirect(w, r, prevPgaeUrl, http.StatusFound)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (rd *Renderer) Local(id string, data ...any) string {
	return rd.i18nCustom.LocalTpl(id, data...)
}

// func (rd *Renderer) SaveUserInfo(u *model.User, w http.ResponseWriter, r *http.Request) {
// 	ss := rd.Session("one", w, r)
// 	ss.SetValue("user_id", u.Id)
// 	ss.SetValue("user_name", u.Name)

// 	// gob.Register(model.User{})

// 	ss.SetValue("user_info", *u)
// }

// func (rd *Renderer) GetUserInfo(w http.ResponseWriter, r *http.Request) (*model.User, error) {
// 	ss := rd.Session("one", w, r)
// 	u := ss.GetValue("user_info")
// 	if v, ok := u.(model.User); ok {
// 		return &v, nil
// 	}
// 	return nil, errors.New("no user info stored in cookie")
// }

type Session struct {
	rd  *Renderer
	Raw *sessions.Session
	w   http.ResponseWriter
	r   *http.Request
}

// Get value from *sessions.Session.Values
func (ss *Session) GetValue(key string) any {
	return ss.Raw.Values[key]
}

func (ss *Session) GetStringValue(key string) string {
	val := ss.GetValue(key)
	if v, ok := val.(string); ok {
		return v
	}
	return ""
}

// Set data to *sessons.Session.Values and auto save, handle save error
func (ss *Session) SetValue(key string, val any) {
	ss.Raw.Values[key] = val

	ss.Raw.Options.HttpOnly = true
	ss.Raw.Options.Secure = !utils.IsDebug()
	ss.Raw.Options.SameSite = http.SameSiteLaxMode
	ss.Raw.Options.Path = "/"

	err := ss.Raw.Save(ss.r, ss.w)
	if err != nil {
		// fmt.Println("save session error: ", err)
		ss.rd.Error("", errors.WithStack(err), ss.w, ss.r, http.StatusInternalServerError)
		return
	}
}

func (ss *Session) Flash(data any, vars ...string) {
	ss.Raw.AddFlash(data, vars...)
	ss.Raw.Save(ss.r, ss.w)
}
