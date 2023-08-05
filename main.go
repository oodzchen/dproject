package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/store/pgstore"
	"github.com/oodzchen/dproject/web"
)

// func createSessionMiddleware(sessStore *sessions.CookieStore) func(http.Handler) http.Handler {
// 	func(next http.Handler) http.Handler {
// 		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 			// create new context from `r` request context, and assign key `"user"`
// 			// to value of `"123"`
// 			ctx := context.WithValue(r.Context(), "user", "123")

// 			// call the next handler in the chain, passing the response writer and
// 			// the updated request object with the new context value.
// 			//
// 			// note: context.Context values are nested, so any previously set
// 			// values will be accessible as well, and the new `"user"` key
// 			// will be accessible from this point forward.
// 			next.ServeHTTP(w, r)
// 		})
// 	}
// }

func main() {
	err := godotenv.Load(".env.local")
	if err != nil {
		log.Fatal(err)
	}

	pg := pgstore.New(&pgstore.DBConfig{
		DSN: os.Getenv("DB_DSN"),
	})

	err = pg.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer pg.CloseDB()

	dataStore, err := store.New(pg)

	tmpl := template.Must(template.New("base").Funcs(sprig.FuncMap()).ParseGlob("./views/*.html"))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// fmt.Printf("env var SESSION_SECRET:%s\n", os.Getenv("SESSION_SECRET"))

	sessStore := sessions.NewCookieStore([]byte(os.Getenv("SESSION_SECRET")))

	articleResource := web.NewArticleResource(tmpl, dataStore.Article, sessStore)

	FileServer(r, "/static", http.Dir("./static"))
	r.Get("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/favicon.ico", http.StatusFound)
	})
	r.Mount("/", web.NewMainResource(tmpl, articleResource, dataStore, sessStore).Routes())
	r.Mount("/articles", articleResource.Routes())
	r.Mount("/users", web.NewUserResource(tmpl, dataStore, sessStore).Routes())

	port := os.Getenv("PORT")
	fmt.Printf("Listening at http://localhost%v\n", port)
	log.Fatal(http.ListenAndServe(port, r))
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
