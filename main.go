package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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

	r.Mount("/", web.NewMainResource(tmpl, articleResource, dataStore, sessStore).Routes())
	r.Mount("/articles", articleResource.Routes())
	r.Mount("/users", web.NewUserResource(tmpl, dataStore, sessStore).Routes())

	port := os.Getenv("PORT")
	fmt.Printf("Listening at http://localhost%v\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
