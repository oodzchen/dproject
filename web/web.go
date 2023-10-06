package web

import (
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/oodzchen/dproject/utils"
	"github.com/pkg/errors"
)

const (
	DefaultPage     int = 1
	DefaultPageSize int = 50
)

// func HandleGetSessionErr(err error) {
// 	if err != nil {
// 		fmt.Printf("get session error: %v\n", err)
// 	}
// }

// func HandleSaveSessionErr(err error) {
// 	if err != nil {
// 		fmt.Printf("session save error: %+v", err)
// 	}
// }

func logSessError(sessName string, err error) {
	if err != nil {
		fmt.Printf("get session '%s' error: %v\n", sessName, err)
	}
}

func ClearSession(sess *sessions.Session, w http.ResponseWriter, r *http.Request) {
	if sess != nil {
		sess.Options.MaxAge = -1
		err := sess.Save(r, w)
		if err != nil {
			// HandleSaveSessionErr(errors.WithStack(err))
			fmt.Printf("session save error: %+v", err)
		}
	}

	csrfExpiredCookie := &http.Cookie{
		Name:     "sc",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   !utils.IsDebug(),
		Path:     "/",
	}

	// fmt.Println("cookie: ", csrfExpiredCookie)

	http.SetCookie(w, csrfExpiredCookie)
}

func GetLoginUserId(sessStore *sessions.CookieStore, w http.ResponseWriter, r *http.Request) (int, error) {
	sess, err := sessStore.Get(r, "one")
	if err != nil {
		logSessError("one", errors.WithStack(err))
		return 0, err
	}

	if userId, ok := (sess.Values["user_id"]).(int); ok && userId > 0 {
		return userId, nil
	}

	return 0, errors.WithStack(errors.New("no user id in session"))
}

func IsLogin(sessStore *sessions.CookieStore, w http.ResponseWriter, r *http.Request) bool {
	_, err := GetLoginUserId(sessStore, w, r)
	return err == nil
}

func HttpError(msg string, err error, w http.ResponseWriter, code int) {
	errText := http.StatusText(code)

	if len(msg) > 0 {
		errText += " - " + msg
	}
	fmt.Printf("%+v\n", err)
	http.Error(w, errText, code)
}

func CeilInt(a, b int) int {
	return int(math.Ceil(float64(a) / float64(b)))
}

func IsRegisterdPage(url *url.URL, r *chi.Mux) bool {
	// currHostName := config.Config.DomainName
	// fmt.Println("url.Hostname(): ", url.Hostname())
	// fmt.Println("config.Config.DomainName", currHostName)
	// return currHostName == url.Hostname() && r.Match(chi.NewRouteContext(), "GET", url.Path)
	return r.Match(chi.NewRouteContext(), "GET", url.Path)
}
