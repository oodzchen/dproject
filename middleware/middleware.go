package mdw

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/service"
	"github.com/oodzchen/dproject/store"
	"github.com/pkg/errors"
)

func FetchUserData(store *store.Store, sessStore *sessions.CookieStore, permissionSrv *service.Permission) func(http.Handler) http.Handler {
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

func AuthCheck(sessStore *sessions.CookieStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// fmt.Printf("r.URL.Path: %s\n", r.URL.Path)
			// fmt.Printf("r.Method: %s\n", r.Method)
			if !isLogin(sessStore, w, r) {
				if r.Method == "GET" {
					sess, _ := sessStore.Get(r, "one")
					sess.Values["target_url"] = r.URL.Path
					sess.Save(r, w) // error here can be ignored
				}

				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func PermitCheck(permissionSrv *service.Permission, permissionIdList []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// fmt.Printf("r.URL.Path: %s\n", r.URL.Path)
			// fmt.Printf("r.Method: %s\n", r.Method)
			if len(permissionIdList) > 0 {
				for _, permissionId := range permissionIdList {
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
						http.Redirect(w, r, "/403", http.StatusFound)
						return
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
