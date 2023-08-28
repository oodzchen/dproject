package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/oodzchen/dproject/utils"
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

type UserInfo struct {
	Id   int
	Name string
}

type UISettings struct {
	Theme         string
	ContentLayout string
}

type PageData struct {
	Title       string
	Data        any
	TipMsg      []string
	LoginedUser *UserInfo
	JSONStr     string
	CSRFField   string
	UISettings  *UISettings
	RoutePath   string
}

type Renderer struct {
	tmpl      *template.Template
	sessStore *sessions.CookieStore
	router    *chi.Mux
}

func (rd *Renderer) Render(w http.ResponseWriter, r *http.Request, name string, data *PageData) {
	rd.doRender(w, r, name, data, http.StatusOK)
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

	userInfo := &UserInfo{}
	if userId, ok := sess.Values["user_id"].(int); ok {
		// fmt.Printf("logined user id: %d\n", userId)
		userInfo.Id = userId
	}

	if userName, ok := sess.Values["user_name"].(string); ok {
		// fmt.Printf("logined user name: %s\n", userName)
		userInfo.Name = userName
	}

	// fmt.Printf("*userInfo == (UserInfo{}): %+v\n", *userInfo == (UserInfo{}))
	// fmt.Printf("&UserInfo{}: %+v\n", &UserInfo{})

	if (UserInfo{}) != *userInfo {
		// fmt.Println("userInfo not empty")
		data.LoginedUser = userInfo
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
				uiSettings.ContentLayout = PageContentLayoutCentered
			}
		}
	}

	data.UISettings = uiSettings
	data.CSRFField = string(csrf.TemplateField(r))
	data.RoutePath = r.URL.Path

	data.Title += fmt.Sprintf(" - %s", os.Getenv("SITE_NAME"))

	if utils.IsDebug() {
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
