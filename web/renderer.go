package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/template"

	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/oodzchen/dproject/utils"
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
	Tmpl      *template.Template
	sessStore *sessions.CookieStore
}

func (rd *Renderer) Render(w http.ResponseWriter, r *http.Request, name string, data *PageData) {
	sess, err := rd.sessStore.Get(r, "one-cookie")
	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
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
	if userId, idOk := sess.Values["user_id"].(int); idOk {
		// fmt.Printf("logined user id: %d\n", userId)
		userInfo.Id = userId
	}

	if userName, nameOk := sess.Values["user_name"].(string); nameOk {
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

	data.Title += fmt.Sprintf(" - %s", os.Getenv("SITE_NAME"))
	data.CSRFField = string(csrf.TemplateField(r))

	// DEBUG
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		utils.HttpError("server error", errors.WithStack(err), w, http.StatusInternalServerError)
		return
	}

	data.JSONStr = string(jsonData)

	w.Header().Set("Content-Type", "text/html")
	err = rd.Tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
