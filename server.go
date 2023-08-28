package main

import (
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/utils"
	"github.com/oodzchen/dproject/web"
)

type ServiceConfig struct {
	sessSecret string
	csrfSecret string
	store      *store.Store
}

var AuthRequiredPathes map[string]Methods = map[string]Methods{
	`^/logout($|/)`:              {"GET"},
	`^/articles($|/)`:            {"POST"},
	`^/articles/\d+/delete($|/)`: {"GET", "POST"},
	`^/articles/\d+/edit($|/)`:   {"GET", "POST"},
	`^/articles/\d+/reply($|/)`:  {"GET", "POST"},
	`^/users/?$`:                 {"GET"},
	`^/settings/account/?$`:      {"GET", "POST"},
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

func Service(c *ServiceConfig) http.Handler {
	baseTmpl := template.New("base").Funcs(TmplFuncs).Funcs(sprig.FuncMap())
	baseTmpl = template.Must(baseTmpl.ParseGlob("./views/*.tmpl"))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentEncoding("default", "gzip"))
	r.Use(middleware.AllowContentType("application/x-www-form-urlencoded"))
	r.Use(middleware.Compress(5, "text/html", "text/css", "text/plain", "text/javascript"))
	r.Use(middleware.GetHead)
	r.Use(middleware.RedirectSlashes)

	sessStore := sessions.NewCookieStore([]byte(c.sessSecret))
	sessStore.Options.HttpOnly = true
	sessStore.Options.Secure = !utils.IsDebug()
	sessStore.Options.SameSite = http.SameSiteLaxMode

	articleResource := web.NewArticleResource(baseTmpl, c.store, sessStore, r)
	userResource := web.NewUserResource(baseTmpl, c.store, sessStore, r)
	mainResource := web.NewMainResource(baseTmpl, c.store, sessStore, articleResource, r)

	rateLimit := 100
	if utils.IsDebug() {
		rateLimit = 10000
	}

	r.Use(httprate.Limit(
		rateLimit,
		1*time.Minute,
		httprate.WithKeyByIP(),
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			mainResource.Error("", nil, w, r, http.StatusTooManyRequests)
		}),
	))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		mainResource.Error("", nil, w, r, http.StatusNotFound)
	})
	r.Use(CreateCheckAuthMiddleware(AuthRequiredPathes, sessStore))
	r.Mount("/debug", middleware.Profiler())
	FileServer(r, "/static", http.Dir("./static"))
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/favicon.ico", http.StatusFound)
	})
	r.Mount("/", mainResource.Routes())
	r.Mount("/articles", articleResource.Routes())
	r.Mount("/users", userResource.Routes())

	// chi.Walk(r, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
	// 	////
	// 	fmt.Println("walk:", method, route)
	// 	return nil
	// })

	CSRF := csrf.Protect([]byte(c.csrfSecret),
		csrf.FieldName("tk"),
		csrf.CookieName("sc"),
		csrf.HttpOnly(true),
		csrf.Secure(!utils.IsDebug()),
		csrf.Path("/"),
		// csrf.ErrorHandler(r),
	)
	return CSRF(r)
}
