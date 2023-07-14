package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/store/pgstore"
	"github.com/oodzchen/dproject/web"
)

const port = ":3000"
const dsn = "postgres://admin:88886666@localhost:8088/discuss"

func main() {
	pg := pgstore.New(&pgstore.DBConfig{
		DSN: dsn,
	})

	err := pg.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer pg.CloseDB()

	s, err := store.New(pg)

	tmpl := template.Must(template.ParseGlob("./views/*.html"))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Mount("/", web.NewArticleResource(tmpl, s.Article).Routes())

	fmt.Printf("Listening at http://localhost%v\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
