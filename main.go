package main

import (
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type PageData struct {
	PageTitle string
}

type PostItem struct {
	Id        int
	Title     string
	Author    string
	TimeStamp int64
	Date      string
}

type HomePageData struct {
	PageData PageData
	Posts    []PostItem
}

func main() {
	tmpl := template.Must(template.ParseGlob("./views/*.html"))
	tmpl.ParseGlob("./views/partials/*.html")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data := HomePageData{
			PageData: PageData{PageTitle: "Home - Dproject"},
			Posts: []PostItem{
				{Id: 0, Title: "这是第一个帖子的标题", Author: "张三", TimeStamp: 1676212151},
				{Id: 1, Title: "这是第二个帖子的标题", Author: "李四", TimeStamp: 1676212171},
				{Id: 2, Title: "这是第三个帖子的标题", Author: "王五", TimeStamp: 1676212191},
			},
		}
		for i, p := range data.Posts {
			data.Posts[i].Date = time.Unix(p.TimeStamp, 0).Format("2006-01-02 15:04:05")
			// data.Posts[i].Date = "Test"
		}
		err := tmpl.ExecuteTemplate(w, "index", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Get("/create", func(w http.ResponseWriter, r *http.Request) {
		err := tmpl.ExecuteTemplate(w, "create", struct {
			PageData PageData
		}{
			PageData: PageData{PageTitle: "Create - Dproject"},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Post("/submit", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		fmt.Println(r.Form)

		http.Redirect(w, r, "/create", http.StatusFound)
	})
	http.ListenAndServe(":3000", r)
}
