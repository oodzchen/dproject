package main

import (
	"net/http"
	"regexp"

	"github.com/gorilla/sessions"
	"github.com/oodzchen/dproject/web"
	"github.com/pkg/errors"
)

type Methods []string

type PahthesNeedAuth map[string]Methods

func CreateCheckAuthMiddleware(pathes PahthesNeedAuth, sessStore *sessions.CookieStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// fmt.Printf("r.URL.Path: %s\n", r.URL.Path)
			// fmt.Printf("r.Method: %s\n", r.Method)
			for pathRe, methods := range pathes {
				re, err := regexp.Compile(pathRe)
				if err != nil {
					web.HttpError("", errors.WithStack(err), w, http.StatusInternalServerError)
					return
				}
				if re.Match([]byte(r.URL.Path)) {
					for _, method := range methods {
						if r.Method == method {
							if !web.IsLogin(sessStore, w, r) {
								if method == "GET" {
									sess, _ := sessStore.Get(r, "one-cookie")
									sess.Values["target_url"] = r.URL.Path
									sess.Save(r, w) // error here can be ignored
								}

								http.Redirect(w, r, "/login", http.StatusFound)
								return
							}
						}
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}