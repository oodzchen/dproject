package main

import (
	"context"
	"fmt"
	"net/http"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
)

var Tmpl *template.Template
var DBConn *pgx.Conn

const Port = ":3000"

func main() {
	DBConn = ConnectDB()
	defer DBConn.Close(context.Background())

	Tmpl = template.Must(template.ParseGlob("./views/*.html"))
	Tmpl.ParseGlob("./views/partials/*.html")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", HomeHandler)
	r.Get("/submit", CreatePageHandler)
	r.Post("/submit", SubmitPostHandler)
	r.Get("/posts", HomeHandler)
	r.Get("/posts/{id}", PostDetailHandler)

	fmt.Printf("Listening at http://localhost%v\n", Port)
	err := http.ListenAndServe(Port, r)
	if err != nil {
		fmt.Printf("Error at Listening: %v\n", err)
	}
}
