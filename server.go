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
	"github.com/oodzchen/dproject/web"
	"github.com/xeonx/timeago"
)

type ServiceConfig struct {
	sessSecret string
	csrfSecret string
	store      *store.Store
}

var tmplFuncs = template.FuncMap{
	"timeAgo": formatTimeAgo,
}

var AuthRequiredPathes map[string]Methods = map[string]Methods{
	`^/logout($|/)`:              {"GET"},
	`^/articles($|/)`:            {"POST"},
	`^/articles/\d+/delete($|/)`: {"DELETE"},
	`^/articles/\d+/edit($|/)`:   {"GET", "POST"},
}

func formatTimeAgo(t time.Time) string {
	return timeago.English.Format(t)
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
	baseTmpl := template.New("base").Funcs(tmplFuncs).Funcs(sprig.FuncMap())
	baseTmpl = template.Must(baseTmpl.ParseGlob("./views/*.html"))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentEncoding("default", "gzip"))
	r.Use(middleware.AllowContentType("application/x-www-form-urlencoded"))
	r.Use(middleware.Compress(5, "text/html", "text/css", "text/plain", "text/javascript"))
	r.Use(middleware.GetHead)
	r.Use(middleware.RedirectSlashes)
	r.Use(httprate.LimitByIP(100, 1*time.Minute))

	sessStore := sessions.NewCookieStore([]byte(c.sessSecret))
	articleResource := web.NewArticleResource(baseTmpl, c.store, sessStore)

	r.Use(CreateCheckAuthMiddleware(AuthRequiredPathes, sessStore))

	r.Mount("/debug", middleware.Profiler())
	FileServer(r, "/static", http.Dir("./static"))
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/favicon.ico", http.StatusFound)
	})
	r.Mount("/", web.NewMainResource(baseTmpl, articleResource, c.store, sessStore).Routes())
	r.Mount("/articles", articleResource.Routes())
	r.Mount("/users", web.NewUserResource(baseTmpl, c.store, sessStore).Routes())

	CSRF := csrf.Protect([]byte(c.csrfSecret), csrf.FieldName("tk"), csrf.CookieName("secure"))
	return CSRF(r)
}
