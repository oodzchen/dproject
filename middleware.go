package main

import (
	"net/http"

	"github.com/gorilla/sessions"
)

var store = sessions.NewCookieStore([]byte("347db14a-ce84-4905-8ffe-197b54841669"))
var Session *sessions.Session

func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Session, _ = store.Get(r, "session-key")
		// fmt.Printf("Session.Values[\"tempUserId\"]: %v\n", Session.Values["tempUserId"])
		if Session.Values["tempUserId"] == nil {
			Session.Values["tempUserId"] = 1
			err := Session.Save(r, w)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

		next.ServeHTTP(w, r)
	})
}
