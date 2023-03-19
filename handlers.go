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
	Id         int
	Title      string
	AuthorName string
	AuthorId   int
	Content    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type PostItem struct {
	RawPostItem
	CreatedAtStr string
	UpdatedAtStr string
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
	// fmt.Printf("sessionManager: %v\n", sessionManager)
	// SessionManager.Put(r.Context(), "tempUserId", 2)

	queryStr := `select
		p.id,
		p.title,
		u.name as author_name,
		p.author_id,
		p.content,
		p.created_at,
		p.updated_at,
		to_char(p.created_at, 'YYYY-MM-DD HH24:MI:SS') as created_at_str,
		to_char(p.updated_at, 'YYYY-MM-DD HH24:MI:SS') as updated_at_str
	from posts p
	left join users u
	on p.author_id = u.id
	where reply_to is null;`
	rows, err := DBConn.Query(context.Background(), queryStr)

	if err != nil {
		fmt.Printf("Query database error: %v\n", err)
		return
	}

	defer rows.Close()

	var list []*PostItem
	for rows.Next() {
		var item PostItem
		err := rows.Scan(
			&item.Id,
			&item.Title,
			&item.AuthorName,
			&item.AuthorId,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.CreatedAtStr,
			&item.UpdatedAtStr)
		if err != nil {
			fmt.Printf("Collect rows error: %v\n", err)
			return
		}
		// listItem := &PostItem{item, item.CreatedAt, item.UpdatedAt.Format("2006年1月2日 15:04:05")}
		// fmt.Printf("listItem: %v\n", listItem)
		list = append(list, &item)
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

	tempUserId := Session.Values["tempUserId"]
	fmt.Printf("tempUserId: %v\n", tempUserId)

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

	tempUserId := Session.Values["tempUserId"]
	fmt.Printf("tempUserId: %v\n", tempUserId)

	insertStr := fmt.Sprintf(
		"insert into posts (title, author_id, content) values ('%s', %d, '%s') returning (id)",
		r.Form["title"][0],
		// r.Form["author"][0],
		tempUserId,
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

	tempUserId := Session.Values["tempUserId"]

	// r.Form
	// fmt.Printf("r.Form: %v\n", r.Form)

	updateStr := fmt.Sprintf(
		"update posts set title = '%s', author_id = %d, content = '%s', updated_at = current_timestamp where id = %s returning (id)",
		r.Form["title"][0],
		// r.Form["author"][0],
		tempUserId,
		r.Form["content"][0],
		r.Form["id"][0])

	var id int
	err = DBConn.QueryRow(context.Background(), updateStr).Scan(&id)
	fmt.Printf("res: %v\n", id)

	if err != nil {
		fmt.Printf("Update posts error. Post Id: %v.\n %v\n", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/posts/%v", id), http.StatusFound)
}

func getPostData(postId string) (*PostItem, error) {
	var item PostItem
	queryStr := fmt.Sprintf(`select
		p.id,
		p.title,
		u.name as author_name,
		p.author_id,
		p.content,
		p.created_at,
		p.updated_at,
		to_char(p.created_at, 'YYYY-MM-DD HH24:MI:SS') as created_at_str,
		to_char(p.updated_at, 'YYYY-MM-DD HH24:MI:SS') as updated_at_str
	from posts p
	left join users u
	on p.author_id = u.id
	where p.id = %v`, postId)
	err := DBConn.QueryRow(context.Background(), queryStr).Scan(
		&item.Id,
		&item.Title,
		&item.AuthorName,
		&item.AuthorId,
		&item.Content,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.CreatedAtStr,
		&item.UpdatedAtStr)
	if err != nil {
		fmt.Println(err)
	}
	// fmt.Printf("item: %v\n", item)
	return &item, nil
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

	updateStr := fmt.Sprintf("update posts set deleted = true where id = %v returning (id)", idForm)
	// fmt.Printf("updateStr: %v\n", updateStr)

	var id int
	err = DBConn.QueryRow(context.Background(), updateStr).Scan(&id)
	fmt.Printf("res: %v\n", id)

	if err != nil {
		fmt.Printf("Delete posts error. Post Id: %v.\n %v\n", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
