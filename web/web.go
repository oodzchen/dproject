package web

import (
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
)

func HandleGetSessionErr(err error) {
	if err != nil {
		fmt.Printf("get session error: %v\n", err)
	}
}

func HandleSaveSessionErr(err error) {
	if err != nil {
		fmt.Printf("session save error: %+v", err)
	}
}

func GetLoginUserId(sessStore *sessions.CookieStore, w http.ResponseWriter, r *http.Request) (int, error) {
	sess, err := sessStore.Get(r, "one")
	if err != nil {
		fmt.Println("get session error", errors.WithStack(err))
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
