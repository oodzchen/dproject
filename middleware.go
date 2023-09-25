package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/service"
	"github.com/oodzchen/dproject/store"
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
					// web.HttpError("", errors.WithStack(err), w, http.StatusInternalServerError)
					http.Redirect(w, r, "/500", http.StatusFound)
					return
				}
				if re.Match([]byte(r.URL.Path)) {
					for _, method := range methods {
						if r.Method == method {
							if !isLogin(sessStore, w, r) {
								if method == "GET" {
									sess, _ := sessStore.Get(r, "one")
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

func CreateUpdateUserDataMiddleware(store *store.Store, sessStore *sessions.CookieStore, permissionSrv *service.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, _ := sessStore.Get(r, "one")
			userId := sess.Values["user_id"]

			var userData *model.User
			if v, ok := userId.(int); ok {
				user, err := store.User.Item(v)
				if err != nil {
					http.Redirect(w, r, "/500", http.StatusFound)
					return
				}
				userData = user
				permissionSrv.SetLoginedUser(user)
			} else {
				userData = nil
			}

			ctx := context.WithValue(r.Context(), "user_data", userData)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type RouteRe struct {
	Method string
	Path   string
}

func CreatePermissionCheckMiddleware(permissionSrv *service.Permission, permissionRouteMap map[string][]RouteRe) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for permissionIds, routeReList := range permissionRouteMap {
				pmArr := strings.Split(permissionIds, ",")
				if len(pmArr) > 0 {
					for _, permissionId := range pmArr {
						moduleAction := strings.Split(permissionId, ".")
						if len(moduleAction) != 2 {
							fmt.Println("permission id error:", permissionId)
							continue
						}

						module := moduleAction[0]
						action := moduleAction[1]

						// fmt.Println("module", module)
						// fmt.Println("action", action)

						if !permissionSrv.PermissionData.Permit(module, action) {
							for _, routeRe := range routeReList {
								methodRe, err := regexp.Compile(routeRe.Method)
								if err != nil {
									http.Redirect(w, r, "/500", http.StatusFound)
									return
								}

								pathRe, err := regexp.Compile(routeRe.Path)
								if err != nil {
									http.Redirect(w, r, "/500", http.StatusFound)
									return
								}

								if methodRe.Match([]byte(r.Method)) && pathRe.Match([]byte(r.URL.Path)) {
									http.Redirect(w, r, "/403", http.StatusFound)
									return
								}
							}
						}
					}
				}

			}
			next.ServeHTTP(w, r)
		})
	}
}

func getLoginUserId(sessStore *sessions.CookieStore, w http.ResponseWriter, r *http.Request) (int, error) {
	sess, err := sessStore.Get(r, "one")
	if err != nil {
		fmt.Println("get session error", errors.WithStack(err))
		// web.ClearSession(sessStore, w, r)
		web.ClearSession(sess, w, r)
		return 0, err
	}

	if userId, ok := (sess.Values["user_id"]).(int); ok && userId > 0 {
		return userId, nil
	}

	return 0, errors.WithStack(errors.New("no user id in session"))
}

func isLogin(sessStore *sessions.CookieStore, w http.ResponseWriter, r *http.Request) bool {
	_, err := getLoginUserId(sessStore, w, r)
	return err == nil
}
