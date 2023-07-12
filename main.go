package main

import (
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const port = ":3000"
const dsn = "postgres://admin:88886666@localhost:8088/discuss"

func main() {
	db := &DB{DSN: dsn}
	err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tmpl := template.Must(template.ParseGlob("./views/*.html"))

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Mount("/", newArticleResource(tmpl, db.Pool).Routes())

	fmt.Printf("Listening at http://localhost%v\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
