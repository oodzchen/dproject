package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
)

type UserInfo struct {
	Id   int
	Name string
}

type PageData struct {
	Title       string
	Data        any
	TipMsg      []string
	LoginedUser *UserInfo
	JSONStr     string
	CSRFField   string
}

type Renderer struct {
	tmpl      *template.Template
	sessStore *sessions.CookieStore
}

func (rd *Renderer) Render(w http.ResponseWriter, r *http.Request, name string, data *PageData) {
	sess, err := rd.sessStore.Get(r, "one-cookie")
	if err != nil {
		HttpError("", err, w, http.StatusInternalServerError)
		return
	}

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

	sess.Save(r, w)
	if err != nil {
		HandleSessionErr(errors.WithStack(err))
	}

	data.CSRFField = string(csrf.TemplateField(r))

	rd.doRender(w, name, data, http.StatusOK)
}

func (rd *Renderer) Error(msg string, err error, w http.ResponseWriter, code int) {
	fmt.Printf("%+v\n", err)

	errText := http.StatusText(code)
	data := &PageData{
		Title: errText,
		Data:  errText,
	}

	if len(msg) > 0 {
		errText += " - " + strings.ToUpper(msg[:1]) + msg[1:]
		data.Data = errText
	}

	rd.doRender(w, "error", data, code)
}

func (rd *Renderer) doRender(w http.ResponseWriter, name string, data *PageData, code int) {
	data.Title += fmt.Sprintf(" - %s", os.Getenv("SITE_NAME"))

	// DEBUG
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		HttpError("", errors.WithStack(err), w, http.StatusInternalServerError)
		return
	}

	data.JSONStr = string(jsonData)

	w.WriteHeader(code)
	w.Header().Set("Content-Type", "text/html")
	err = rd.tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		HttpError("", errors.WithStack(err), w, http.StatusInternalServerError)
	}
}
