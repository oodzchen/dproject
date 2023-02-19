package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
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
	queryStr := "select id, title, author, content, created, updated from posts where deleted = false order by created desc"
	rows, err := DBConn.Query(context.Background(), queryStr)

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
		listItem.Created = item.Created
		listItem.CreatedStr = item.Created.Format("2006年1月2日 15:04:05")
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
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	data := new(PostPageData)

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

	// fmt.Printf("r.Form:\n %v\n", r.Form)
	// fmt.Printf("r.Form[\"title\"][0]: %v\n", r.Form["title"][0])

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

func UpdatePostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	// fmt.Printf("r.Form:\n %v\n", r.Form)
	// fmt.Printf("r.Form[\"title\"][0]: %v\n", r.Form["title"][0])

	updateStr := fmt.Sprintf(
		"update posts set title = '%v', author = '%v', content = '%v', updated = current_timestamp where id = %v returning (id)",
		r.Form["title"][0],
		r.Form["author"][0],
		r.Form["content"][0],
		r.Form["id"][0])
	// fmt.Printf("updateStr: %v\n", updateStr)

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
	var id int
	var title string
	var author string
	var content string
	var created time.Time
	var updated time.Time

	err := DBConn.QueryRow(context.Background(), fmt.Sprintf("select id, title, author, content, created, updated from posts where id = %v", postId)).Scan(
		&id,
		&title,
		&author,
		&content,
		&created,
		&updated)
	if err != nil {
		fmt.Printf("Query post detail error, id: %v, error: %v", postId, err)
		return nil, err
	}

	postItem := new(PostItem)
	postItem.Id = id
	postItem.Title = title
	postItem.Author = author
	postItem.Content = content
	postItem.Created = created
	postItem.Updated = updated
	postItem.CreatedStr = created.Format("2006年1月2日 15:04:05")
	postItem.UpdatedStr = updated.Format("2006年1月2日 15:04:05")

	return postItem, nil
}

func PostDetailHandler(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	postData, err := getPostData((idParam))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	re := regexp.MustCompile(`\r`)

	data := new(PostPageData)
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

	data := new(PostPageData)
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

	updateStr := fmt.Sprintf("update posts set deleted = true where id = %v returning (id)", idForm)
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
