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
	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
	"github.com/pkg/errors"
)

const (
	PageThemeLight  string = "light"
	PageThemeDark          = "dark"
	PageThemeSystem        = "system"
)

const (
	PageContentLayoutFull     string = "full"
	PageContentLayoutCentered        = "centered"
)

// type UserInfo struct {
// 	Id   int
// 	Name string
// }

type UISettings struct {
	Theme         string
	ContentLayout string
}

type BreadCrumb struct {
	Path string
	Name string
}

type PageData struct {
	Title       string
	Data        any
	TipMsg      []string
	LoginedUser *model.User
	JSONStr     string
	CSRFField   string
	UISettings  *UISettings
	RoutePath   string
	Debug       bool
	DebugUsers  []*model.User
	BreadCrumbs []*BreadCrumb
	I18nData    map[string]any
	BrandName   string
}

func (pd *PageData) AddI18nData(data map[string]any) {
	if pd.I18nData != nil {
		for k, v := range data {
			pd.I18nData[k] = v
		}
	} else {
		pd.I18nData = data
	}
}

type Renderer struct {
	tmpl      *template.Template
	sessStore *sessions.CookieStore
	router    *chi.Mux
	store     *store.Store
}

func (rd *Renderer) Render(w http.ResponseWriter, r *http.Request, name string, data *PageData) {
	rd.doRender(w, r, name, data, http.StatusOK)
}

func (rd *Renderer) ServerError(msg string, err error, w http.ResponseWriter, r *http.Request) {
	rd.Error(msg, err, w, r, http.StatusInternalServerError)
}

func (rd *Renderer) ToLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (rd *Renderer) NotFound(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/404", http.StatusNotFound)
}

func (rd *Renderer) GetLoginedUserId(w http.ResponseWriter, r *http.Request) int {
	userId := rd.Session("one", w, r).GetValue("user_id")
	if userId, ok := userId.(int); ok {
		return userId
	}
	return 0
}

func (rd *Renderer) Error(msg string, err error, w http.ResponseWriter, r *http.Request, code int) {
	fmt.Printf("%+v\n", err)

	referer := r.Referer()
	refererUrl, _ := url.Parse(referer)
	prevUrl := ""

	if IsRegisterdPage(refererUrl, rd.router) {
		prevUrl = referer
	}

	errText := http.StatusText(code)

	type errPageData struct {
		ErrText string
		PrevUrl string
	}
	data := &PageData{
		Title: errText,
		Data:  &errPageData{errText, prevUrl},
	}

	if len(msg) > 0 {
		errText += " - " + strings.ToUpper(msg[:1]) + msg[1:]
		data.Data = &errPageData{errText, prevUrl}
	}

	rd.doRender(w, r, "error", data, code)
}

func (rd *Renderer) doRender(w http.ResponseWriter, r *http.Request, name string, data *PageData, code int) {
	// sess, err := rd.sessStore.Get(r, "one")
	// HandleGetSessionErr(err)

	sess := rd.Session("one", w, r).Raw

	if flashes := sess.Flashes(); len(flashes) > 0 {
		for _, item := range flashes {
			if value, ok := item.(string); ok {
				data.TipMsg = append(data.TipMsg, value)
			}
		}
	}

	if userId, ok := sess.Values["user_id"].(int); ok {
		// fmt.Printf("logined user id: %d\n", userId)
		// userInfo.Id = userId
		loginedUser, err := rd.store.User.Item(userId)
		if err != nil {
			fmt.Println("get logined user info failed: ", err)
		}

		data.LoginedUser = loginedUser
	}

	err := sess.Save(r, w)
	if err != nil {
		HandleSaveSessionErr(errors.WithStack(err))
	}

	localSess := rd.Session("local", w, r)
	uiSettings := &UISettings{}
	uiSettingsKeys := []string{"page_theme", "page_content_layout"}
	for _, key := range uiSettingsKeys {
		sessVal := localSess.GetValue(key)
		switch key {
		case "page_theme":
			if theme, ok := sessVal.(string); ok {
				uiSettings.Theme = theme
			} else {
				uiSettings.Theme = PageThemeLight
			}
		case "page_content_layout":
			if layout, ok := sessVal.(string); ok {
				uiSettings.ContentLayout = layout
			} else {
				uiSettings.ContentLayout = PageContentLayoutFull
			}
		}
	}

	data.UISettings = uiSettings
	data.CSRFField = string(csrf.TemplateField(r))
	data.RoutePath = r.URL.Path
	data.Debug = config.Config.Debug
	data.BrandName = config.Config.BrandName
	data.Title += fmt.Sprintf(" - %s", config.Config.SiteName)

	// data.AddI18nData(map[string]any{
	// 	"ReplyNum": i18nc.Localizer.MustLocalize(&i18n.LocalizeConfig{
	// 		DefaultMessage: &i18n.Message{
	// 			ID:          "ReplyNum",
	// 			Description: "Reply number",
	// 			One:         "{{.Count}} reply",
	// 			Other:       "{{.Count}} replies",
	// 		},
	// 		PluralCount: 0,
	// 	}),
	// })

	// rd.tmpl = rd.tmpl.Funcs(template.FuncMap{

	// })

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

	w.WriteHeader(code)
	w.Header().Set("Content-Type", "text/html")

	err = rd.tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		HttpError("", errors.WithStack(err), w, http.StatusInternalServerError)
	}
}

func (rd *Renderer) Session(name string, w http.ResponseWriter, r *http.Request) *Session {
	sess, err := rd.sessStore.Get(r, name)
	HandleGetSessionErr(err)
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

// Set data to *sessons.Session.Values and auto save, handle save error
func (ss *Session) SetValue(key string, val any) {
	ss.Raw.Values[key] = val
	err := ss.Raw.Save(ss.r, ss.w)
	if err != nil {
		ss.rd.Error("", errors.WithStack(err), ss.w, ss.r, http.StatusInternalServerError)
	}
}
