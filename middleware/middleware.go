package mdw

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	i18nc "github.com/oodzchen/dproject/i18n"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/service"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/utils"
	"github.com/oschwald/geoip2-golang"
	"github.com/pkg/errors"
)

type Renderer interface {
	ServerErrorp(msg string, err error, w http.ResponseWriter, r *http.Request)
	Forbidden(err error, w http.ResponseWriter, r *http.Request)
	GetLoginedUserData(r *http.Request) *model.User
}

func logSessError(sessName string, err error) {
	if err != nil {
		fmt.Printf("get session '%s' error: %v\n", sessName, err)
	}
}

func FetchUserData(store *store.Store, sessStore *sessions.CookieStore, permissionSrv *service.Permission, renderer any) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := sessStore.Get(r, "one")
			logSessError("one", errors.WithStack(err))

			userId := sess.Values["user_id"]

			var userData *model.User
			if v, ok := userId.(int); ok {
				user, err := store.User.Item(v)
				if err != nil {
					sess.Options.MaxAge = -1
					err = sess.Save(r, w)
					if err != nil {
						fmt.Println("clear session error:", err)
					}

					fmt.Println("get user data error:", err)
					if v, ok := renderer.(Renderer); ok {
						// v.ServerError(w, r)
						v.ServerErrorp("", err, w, r)
					} else {
						http.Redirect(w, r, "/500", http.StatusFound)
					}

					return
				}
				userData = user
				// permissionSrv.SetLoginedUser(user)
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
					sess, err := sessStore.Get(r, "one")
					logSessError("one", errors.WithStack(err))

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

func toForbiddenPage(renderer Renderer, w http.ResponseWriter, r *http.Request) {
	renderer.Forbidden(nil, w, r)
}

// User must have at least one permisison id in needPermissionIds
func PermitCheck(permissionSrv *service.Permission, needPermissionIds []string, renderer Renderer) func(http.Handler) http.Handler {
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

					if permissionSrv.Permit(renderer.GetLoginedUserData(r), module, action) {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			toForbiddenPage(renderer, w, r)
		})
	}
}

type UserLoggerFn func(uLogData *service.UserLogData, w http.ResponseWriter, r *http.Request) error

var (
	ULogEmpty UserLoggerFn = func(u *service.UserLogData, w http.ResponseWriter, r *http.Request) error {
		return nil
	}

	ULogLoginedUserId = func(u *service.UserLogData, w http.ResponseWriter, r *http.Request) error {
		username := chi.URLParam(r, "username")
		if username == "" {
			return errors.New("username is empty")
		}
		u.TargetId = username
		return nil
	}

	ULogRoleId = func(u *service.UserLogData, w http.ResponseWriter, r *http.Request) error {
		id, err := strconv.Atoi(chi.URLParam(r, "roleId"))
		if err != nil {
			return err
		}
		u.TargetId = id
		return nil
	}

	ULogNewArticleId = func(u *service.UserLogData, w http.ResponseWriter, r *http.Request) error {
		articleIdStr := r.Context().Value("article_id")
		id, ok := articleIdStr.(int)
		if !ok {
			return errors.New("get article id failed")
		}

		// fmt.Println("articleId: ", articleId)

		u.TargetId = id
		return nil
	}

	ULogURLArticleId = func(u *service.UserLogData, w http.ResponseWriter, r *http.Request) error {
		id, err := strconv.Atoi(chi.URLParam(r, "articleId"))
		if err != nil {
			return err
		}
		u.TargetId = id
		return nil
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

func UserLogger(uLogger *service.UserLogger, actType model.AcType, action model.AcAction, targetModel model.AcModel, uHandlers ...UserLoggerFn) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// fmt.Println("in middleware test before http")
			next.ServeHTTP(w, r)

			uLogData := &service.UserLogData{
				ActionType:  actType,
				Action:      action,
				TargetModel: targetModel,
				DeviceInfo:  r.UserAgent(),
				IPAddr:      getRealIP(r),
			}
			// fmt.Println("in middleware test after http")
			user, _ := getLoginedUserData(r)
			// fmt.Println("user data: ", user)

			// var targetId int

			// if uHandler != nil {
			// 	uLogData.TargetId = uHandler(w, r)
			// }

			for _, handler := range uHandlers {
				if handler != nil {
					err := handler(uLogData, w, r)
					if err != nil {
						fmt.Println("UserLogger error: ", err)
					}
				}
			}

			go func() {
				err := uLogger.Log(user, func(r *http.Request) *service.UserLogData {
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
					uLogData.Details = details
					return uLogData
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
	logSessError("one", errors.WithStack(err))

	if userId, ok := (sess.Values["user_id"]).(int); ok && userId > 0 {
		return userId, nil
	}

	return 0, errors.WithStack(errors.New("no user id in session"))
}

func isLogin(sessStore *sessions.CookieStore, w http.ResponseWriter, r *http.Request) bool {
	_, err := getLoginUserId(sessStore, w, r)
	return err == nil
}

func CreateUISettingsMiddleware(sessStore *sessions.CookieStore, sm *service.SettingsManager, ic *i18nc.I18nCustom) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			localSess, err := sessStore.Get(r, "local")
			logSessError("local", errors.WithStack(err))

			uiSettings := model.DefaultUiSettings
			uiSettings.Lang = getAcceptLang(r)

			settingsId := localSess.Values["ui-settings-id"]

			if id, ok := settingsId.(string); ok {
				settings, err := sm.GetSettings(id)
				if err != nil {
					fmt.Println("get ui settings error: ", err)
				} else {
					uiSettings = settings
				}
			}

			ic.SwitchLang(uiSettings.Lang.String())
			model.UpdateErrI18n()

			ctx := context.WithValue(r.Context(), "ui_settings", uiSettings)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

var langReMap = map[model.Lang]string{
	model.LangZhHans: `^zh(?:-(?:(?:Hans|cmn)|(?:cmn-Hans)|(?:Hans-.*)|(?:CN|SG)))?$`,
	model.LangZhHant: `^zh-(?:(?:Hant|cmn-Hant)|(?:Hant-.*)|(?:HK|TW|MO))$`,
	model.LangEn:     `^(?:en(?:-.*)?)$`,
	model.LangJa:     `^(?:j(p|a)(?:-.*)?)$`,
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

func RequestDuration(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		// fmt.Println("start time: ", startTime)
		ctx := context.WithValue(r.Context(), "req_duration_start", startTime)
		defer func() {
			fmt.Printf("response duration: %dms\n", time.Since(startTime).Milliseconds())
		}()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CreateGeoDetect(geoDB *geoip2.Reader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			realIP := getRealIP(r)

			// realIP = "120.238.203.9"

			record, err := geoDB.Country(net.ParseIP(realIP))
			if err != nil {
				fmt.Println("parse geo ip error:", err)
				next.ServeHTTP(w, r)
			} else {
				fmt.Printf("country ISO code: %#v\n", record.Country.IsoCode)

				ctx := context.WithValue(r.Context(), "region_country_iso_code", record.Country.IsoCode)

				next.ServeHTTP(w, r.WithContext(ctx))
			}

		})
	}
}

func getRealIP(r *http.Request) string {
	realIP := r.Header.Get("CF-Connecting-IP")

	if realIP == "" {
		realIP = r.Header.Get("X-Real-IP")
		// fmt.Println("x-real-ip:", realIP)
	}

	if realIP == "" {
		realIP = r.Header.Get("X-Forwarded-For")
		// fmt.Println("x-Forwarded-for:", realIP)
	}

	if realIP == "" {
		realIP = strings.Split(r.RemoteAddr, ":")[0]
		// ip := "38.59.236.10"
		// fmt.Println("geo db metadata:", geoDB.Metadata())
		// fmt.Println("r.RemoteAddr:", realIP)
	}
	return realIP
}
