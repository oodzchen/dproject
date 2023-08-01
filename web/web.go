package web

import (
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
)

func HandleSessionErr(err error) {
	fmt.Printf("session save error: %+v", err)
}

func IsLogin(sessStoer *sessions.CookieStore, w http.ResponseWriter, r *http.Request) bool {
	sess, err := sessStoer.Get(r, "one-cookie")
	if err != nil {
		fmt.Println(errors.WithStack(err))
		return false
	}

	if userId, ok := (sess.Values["user_id"]).(int); ok && userId > 0 {
		return true
	}

	return false
}
