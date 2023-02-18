package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type RawPostItem struct {
	Id      int
	Title   string
	Author  string
	Content string
	Updated time.Time
}

type PostItem struct {
	RawPostItem
	UpdatedStr string
}

type BasePageData struct {
	PageTitle string
}

type HomePageData struct {
	BasePageData
	Posts     []*PostItem
	PostTotal int
}

type PostPageData struct {
	BasePageData
	PostItem
}

func render(w http.ResponseWriter, r *http.Request, name string, data any) {
	err := Tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := DBConn.Query(context.Background(), "select id, title, author, content, created from posts")

	if err != nil {
		fmt.Printf("Query database error: %v\n", err)
		return
	}

	rawList, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByPos[RawPostItem])

	if err != nil {
		fmt.Printf("Collect rows error: %v\n", err)
		return
	}

	var list []*PostItem
	for _, item := range rawList {
		// fmt.Printf("List item: %v\n", item)
		listItem := new(PostItem)
		listItem.Id = item.Id
		listItem.Title = item.Title
		listItem.Author = item.Author
		listItem.Content = item.Content
		listItem.Updated = item.Updated
		listItem.UpdatedStr = item.Updated.Format("2006年1月2日 15:04:05")

		list = append(list, listItem)
	}

	data := new(HomePageData)
	data.PageTitle = "Home - Dproject"
	data.Posts = list
	data.PostTotal = len(list)

	render(w, r, "index", data)
}

func CreatePageHandler(w http.ResponseWriter, r *http.Request) {
	render(w, r, "create", &BasePageData{PageTitle: "Create - Dproject"})
}

func SubmitPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	fmt.Printf("r.Form:\n %v\n", r.Form)
	fmt.Printf("r.Form[\"title\"][0]: %v\n", r.Form["title"][0])

	insertStr := fmt.Sprintf(
		"insert into posts values (default, '%v', '%v', '%v', current_timestamp) returning (id)",
		r.Form["title"][0],
		r.Form["author"][0],
		r.Form["content"][0])
	// fmt.Printf("insertStr: %v\n", insertStr)

	var id int
	err = DBConn.QueryRow(context.Background(), insertStr).Scan(&id)
	fmt.Printf("res: %v\n", id)

	if err != nil {
		fmt.Printf("Insert into posts error: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/posts/%v", id), http.StatusFound)
}

func PostDetailHandler(w http.ResponseWriter, r *http.Request) {
	var id int
	var title string
	var author string
	var content string
	var created time.Time

	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	err := DBConn.QueryRow(context.Background(), fmt.Sprintf("select id, title, author, content, created from posts where id = %v", idParam)).Scan(
		&id,
		&title,
		&author,
		&content,
		&created,
	)
	if err != nil {
		fmt.Printf("Query post detail error, id: %v, error: %v", idParam, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	data := new(PostPageData)
	data.PageTitle = title
	data.Title = title
	data.Author = author
	data.Content = content
	data.Updated = created
	data.UpdatedStr = created.Format("2006年1月2日 15:04:05")
	render(w, r, "post", data)
}
