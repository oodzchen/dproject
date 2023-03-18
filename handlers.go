package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
)

type RawPostItem struct {
	Id      int
	Title   string
	Author  string
	Content string
	Created time.Time
	Updated time.Time
}

type PostItem struct {
	RawPostItem
	CreatedStr string
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
	Post *PostItem
}

func render(w http.ResponseWriter, r *http.Request, name string, data any) {
	err := Tmpl.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	queryStr := "select id, title, author, content, created, updated from post where deleted = false order by created desc"
	rows, err := DBConn.Query(context.Background(), queryStr)

	if err != nil {
		fmt.Printf("Query database error: %v\n", err)
		return
	}

	defer rows.Close()

	var list []*PostItem
	for rows.Next() {
		var item RawPostItem
		err := rows.Scan(&item.Id, &item.Title, &item.Author, &item.Content, &item.Created, &item.Updated)
		if err != nil {
			fmt.Printf("Collect rows error: %v\n", err)
			return
		}
		listItem := &PostItem{item, item.Created.Format("2006年1月2日 15:04:05"), item.Updated.Format("2006年1月2日 15:04:05")}
		// fmt.Printf("listItem: %v\n", listItem)
		list = append(list, listItem)
	}

	var data HomePageData
	data.PageTitle = "Home - Dproject"
	data.Posts = list
	data.PostTotal = len(list)

	render(w, r, "index", data)
}

func CreatePageHandler(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	var data PostPageData

	if idParam == "" {
		data.PageTitle = "Create - Dproject"
		data.Post = &PostItem{}
	} else {
		postData, _ := getPostData(idParam)
		data.PageTitle = fmt.Sprintf("Edit - %v", postData.Title)
		data.Post = postData
	}

	render(w, r, "create", data)
}

func SubmitPostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	insertStr := fmt.Sprintf(
		"insert into post values (default, '%v', '%v', '%v', current_timestamp) returning (id)",
		r.Form["title"][0],
		r.Form["author"][0],
		r.Form["content"][0])
	// fmt.Printf("insertStr: %v\n", insertStr)

	var id int
	err = DBConn.QueryRow(context.Background(), insertStr).Scan(&id)
	fmt.Printf("res: %v\n", id)

	if err != nil {
		fmt.Printf("Insert into post error: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/posts/%v", id), http.StatusFound)
}

func UpdatePostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	updateStr := fmt.Sprintf(
		"update post set title = '%v', author = '%v', content = '%v', updated = current_timestamp where id = %v returning (id)",
		r.Form["title"][0],
		r.Form["author"][0],
		r.Form["content"][0],
		r.Form["id"][0])

	var id int
	err = DBConn.QueryRow(context.Background(), updateStr).Scan(&id)
	fmt.Printf("res: %v\n", id)

	if err != nil {
		fmt.Printf("Update post error. Post Id: %v.\n %v\n", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/posts/%v", id), http.StatusFound)
}

func getPostData(postId string) (*PostItem, error) {
	var item RawPostItem

	err := DBConn.QueryRow(context.Background(),
		fmt.Sprintf("select id, title, author, content, created, updated from post where id = %v", postId)).Scan(&item.Id, &item.Title, &item.Author, &item.Content, &item.Created, &item.Updated)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("item: %v\n", item)
	return &PostItem{item, item.Created.Format("2006年1月2日 15:04:05"), item.Updated.Format("2006年1月2日 15:04:05")}, nil
}

func PostDetailHandler(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	postData, err := getPostData((idParam))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	re := regexp.MustCompile(`\r`)

	var data PostPageData
	data.PageTitle = postData.Title
	data.Post = postData
	data.Post.Content = re.ReplaceAllString(postData.Content, "<br/>")
	render(w, r, "post", data)
}

func EditPostHandler(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	postData, err := getPostData((idParam))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	var data PostPageData
	data.PageTitle = postData.Title
	data.Post = postData
	render(w, r, "create", data)
}

func DeletePostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	idForm := r.Form["id"][0]

	// fmt.Printf("r.Form:\n %v\n", r.Form)
	// fmt.Printf("r.Form[\"title\"][0]: %v\n", r.Form["title"][0])

	updateStr := fmt.Sprintf("update post set deleted = true where id = %v returning (id)", idForm)
	// fmt.Printf("updateStr: %v\n", updateStr)

	var id int
	err = DBConn.QueryRow(context.Background(), updateStr).Scan(&id)
	fmt.Printf("res: %v\n", id)

	if err != nil {
		fmt.Printf("Delete post error. Post Id: %v.\n %v\n", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
