package mdw

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	i18nc "github.com/oodzchen/dproject/i18n"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/service"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/utils"
	"github.com/pkg/errors"
)

type Renderer interface {
	ServerError(w http.ResponseWriter, r *http.Request)
	Forbidden(w http.ResponseWriter, r *http.Request)
}

func FetchUserData(store *store.Store, sessStore *sessions.CookieStore, permissionSrv *service.Permission, renderer any) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, _ := sessStore.Get(r, "one")
			userId := sess.Values["user_id"]

			var userData *model.User
			if v, ok := userId.(int); ok {
				user, err := store.User.Item(v)
				if err != nil {
					if v, ok := renderer.(Renderer); ok {
						v.ServerError(w, r)
					} else {
						http.Redirect(w, r, "/500", http.StatusFound)
					}

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

func toForbiddenPage(renderer any, w http.ResponseWriter, r *http.Request) {
	if v, ok := renderer.(Renderer); ok {
		v.Forbidden(w, r)
	} else {
		http.Redirect(w, r, "/403", http.StatusFound)
	}
}

// User must have at least one permisison id in needPermissionIds
func PermitCheck(permissionSrv *service.Permission, needPermissionIds []string, renderer any) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(needPermissionIds) == 0 {
				toForbiddenPage(renderer, w, r)
				return
			} else {
				for _, permissionId := range needPermissionIds {
					moduleAction := strings.Split(permissionId, ".")
					if len(moduleAction) != 2 {
						fmt.Println("permission id error:", permissionId)
						continue
					}

					module := moduleAction[0]
					action := moduleAction[1]

					// fmt.Println("module", module)
					// fmt.Println("action", action)

					if permissionSrv.PermissionData.Permit(module, action) {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			toForbiddenPage(renderer, w, r)
		})
	}
}

type UserLoggerFn func(w http.ResponseWriter, r *http.Request) (targetId int)

var (
	ULogEmpty UserLoggerFn = func(w http.ResponseWriter, r *http.Request) int {
		return 0
	}

	ULogLoginedUserId = func(w http.ResponseWriter, r *http.Request) (targetId int) {
		id, err := strconv.Atoi(chi.URLParam(r, "userId"))
		if err != nil {
			return 0
		}
		return id
	}

	ULogRoleId = func(w http.ResponseWriter, r *http.Request) (targetId int) {
		id, err := strconv.Atoi(chi.URLParam(r, "roleId"))
		if err != nil {
			return 0
		}
		return id
	}

	ULogNewArticleId = func(w http.ResponseWriter, r *http.Request) (targetId int) {
		articleIdStr := r.Context().Value("article_id")
		articleId, ok := articleIdStr.(int)
		if !ok {
			return 0
		}

		// fmt.Println("articleId: ", articleId)

		return articleId
	}

	ULogURLArticleId = func(w http.ResponseWriter, r *http.Request) (targetId int) {
		id, err := strconv.Atoi(chi.URLParam(r, "articleId"))
		if err != nil {
			return 0
		}
		return id
	}
)

func getLoginedUserData(r *http.Request) (*model.User, error) {
	userData := r.Context().Value("user_data")

	// fmt.Println("user data: ", userData)
	if u, ok := userData.(*model.User); ok {
		return u, nil
	}
	return nil, errors.New("no user data in request context")
}

func UserLogger(uLogger *service.UserLogger, actType model.AcType, action model.AcAction, targetModel model.AcModel, uHandler UserLoggerFn) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// fmt.Println("in middleware test before http")
			next.ServeHTTP(w, r)
			// fmt.Println("in middleware test after http")
			user, _ := getLoginedUserData(r)
			// fmt.Println("user data: ", user)

			var targetId int

			if uHandler != nil {
				targetId = uHandler(w, r)
			}

			go func() {
				err := uLogger.Log(user, actType, action, targetModel, func(r *http.Request) *service.UserLogData {
					postData := make(map[string]any)
					for k, v := range r.PostForm {
						if k == "tk" || k == "password" {
							continue
						}
						for idx, str := range v {
							if len(str) > 200 {
								v[idx] = str[:200] + "..."
							}
						}
						postData[k] = v
					}

					details := utils.SprintJSONf(postData, "", "")
					return &service.UserLogData{
						TargetId:   targetId,
						Details:    details,
						DeviceInfo: r.UserAgent(),
						IPAddr:     strings.Split(r.RemoteAddr, ":")[0],
					}
				}, r)
				if err != nil {
					fmt.Println("user activity logger error:", err)
				}
			}()
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

func CreateUISettingsMiddleware(sessStore *sessions.CookieStore, ic *i18nc.I18nCustom) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			localSess, _ := sessStore.Get(r, "local")
			uiSettings := &model.UISettings{}
			uiSettingsKeys := []string{"lang", "page_theme", "page_content_layout"}
			acceptLang := getAcceptLang(r)
			// fmt.Println("acceptLang: ", acceptLang)

			for _, key := range uiSettingsKeys {
				sessVal := localSess.Values[key]
				switch key {
				case "lang":
					// fmt.Println("sessionVal: ", sessVal)
					if lang, ok := sessVal.(model.Lang); ok {
						uiSettings.Lang = lang
					} else {
						uiSettings.Lang = acceptLang
					}
					ic.SwitchLang(uiSettings.Lang.String())
				case "page_theme":
					if theme, ok := sessVal.(string); ok {
						uiSettings.Theme = theme
					} else {
						uiSettings.Theme = model.PageThemeSystem
					}
				case "page_content_layout":
					if layout, ok := sessVal.(string); ok {
						uiSettings.ContentLayout = layout
					} else {
						uiSettings.ContentLayout = model.PageContentLayoutCentered
					}
				}
			}

			ctx := context.WithValue(r.Context(), "ui_settings", uiSettings)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

var langReMap = map[model.Lang]string{
	model.LangZhHans: `^zh(?:-(?:(?:Hans|cmn)|(?:cmn-Hans)|(?:Hans-.*)|(?:CN|SG)))?$`,
	model.LangZhHant: `^zh(?:-(?:(?:Hant|cmn-Hant)|(?:Hant-.*)|(?:HK|TW|MO)))?$`,
	model.LangEn:     `^(?:en(?:-.*)?)$`,
	model.LangJp:     `^(?:jp(?:-.*)?)$`,
}

func parseStrLang(str string) model.Lang {
	for lang, pattern := range langReMap {
		re := regexp.MustCompile(pattern)
		if re.Match([]byte(str)) {
			return lang
		}
	}

	return model.LangEn
}

func getAcceptLang(r *http.Request) model.Lang {
	accpetLangs := r.Header.Get("Accept-Language")
	// fmt.Println("acceptLangs: ", accpetLangs)
	firstLang, _, found := strings.Cut(accpetLangs, ",")
	if !found || strings.TrimSpace(firstLang) == "" {
		// fmt.Println("accept first lang: ", firstLang)
		return model.LangEn
	}

	return parseStrLang(firstLang)
}
