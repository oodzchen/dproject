package main

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"text/template"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type ArticleResource struct {
	Renderer
	DBConn *pgx.Conn
}

type ArticleItem struct {
	Id           int
	Title        string
	AuthorName   string
	AuthorId     int
	Content      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedAtStr string
	UpdatedAtStr string
}

func newArticleResource(tmpl *template.Template, conn *pgx.Conn) *ArticleResource {
	return &ArticleResource{Renderer{tmpl}, conn}
}

func (rs *ArticleResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Get("/", rs.List)
	rt.Get("/articles", rs.List)
	rt.Post("/articles", rs.Submit)
	rt.Get("/articles/new", rs.CreatePage)

	rt.Route("/articles/{id}", func(r chi.Router) {
		r.Get("/", rs.Get)
		r.Get("/edit", rs.EditPage)
		r.Post("/edit", rs.Update)
		r.Post("/delete", rs.Delete)
	})

	return rt
}

func (rs *ArticleResource) List(w http.ResponseWriter, r *http.Request) {
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
	where reply_to is null and deleted is false;`
	rows, err := rs.DBConn.Query(context.Background(), queryStr)

	if err != nil {
		fmt.Printf("Query database error: %v\n", err)
		return
	}

	defer rows.Close()

	var list []*ArticleItem
	for rows.Next() {
		var item ArticleItem
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
		list = append(list, &item)
	}

	type ListData struct {
		Articles     []*ArticleItem
		ArticleTotal int
	}

	rs.render(w, r, "index", &PageData{"Home - Dproject", &ListData{list, len(list)}})
}

func (rs *ArticleResource) CreatePage(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	// tempUserId := Session.Values["tempUserId"]
	// fmt.Printf("tempUserId: %v\n", tempUserId)

	// var data PostPageData
	var pageTitle string
	var data *ArticleItem

	if idParam == "" {
		pageTitle = "Create - Dproject"
		data = &ArticleItem{}
	} else {
		postData, _ := rs.getPostData(idParam)
		pageTitle = fmt.Sprintf("Edit - %v", postData.Title)
		data = postData
	}

	rs.render(w, r, "create", &PageData{pageTitle, data})
}

func (rs *ArticleResource) Submit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	// tempUserId := Session.Values["tempUserId"]
	// fmt.Printf("tempUserId: %v\n", tempUserId)

	insertStr := fmt.Sprintf(
		"insert into posts (title, author_id, content) values ('%s', %s, '%s') returning (id)",
		r.Form["title"][0],
		r.Form["author"][0],
		// tempUserId,
		r.Form["content"][0])
	// fmt.Printf("insertStr: %v\n", insertStr)

	var id int
	err = rs.DBConn.QueryRow(context.Background(), insertStr).Scan(&id)
	fmt.Printf("res: %v\n", id)

	if err != nil {
		fmt.Printf("Insert into posts error: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/articles/%v", id), http.StatusFound)
}

func (rs *ArticleResource) Update(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	fmt.Printf("r.Form: %v\n", r.Form)

	updateStr := fmt.Sprintf(
		"update posts set title = '%s', author_id = %s, content = '%s', updated_at = current_timestamp where id = %s returning (id)",
		r.Form["title"][0],
		r.Form["author_id"][0],
		// tempUserId,
		r.Form["content"][0],
		r.Form["id"][0])

	var id int
	err = rs.DBConn.QueryRow(context.Background(), updateStr).Scan(&id)
	fmt.Printf("res: %v\n", id)

	if err != nil {
		fmt.Printf("Update posts error. Post Id: %v.\n %v\n", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/articles/%v", id), http.StatusFound)
}

func (rs *ArticleResource) getPostData(postId string) (*ArticleItem, error) {
	var item ArticleItem
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
	err := rs.DBConn.QueryRow(context.Background(), queryStr).Scan(
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
		return nil, err
	}
	// fmt.Printf("item: %v\n", item)
	return &item, nil
}

func (rs *ArticleResource) Get(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	postData, err := rs.getPostData((idParam))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	re := regexp.MustCompile(`\r`)

	postData.Content = re.ReplaceAllString(postData.Content, "<br/>")
	rs.render(w, r, "article", &PageData{postData.Title, postData})
}

func (rs *ArticleResource) EditPage(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	postData, err := rs.getPostData((idParam))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	rs.render(w, r, "create", &PageData{postData.Title, postData})
}

func (rs *ArticleResource) Delete(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	idForm := r.Form["id"][0]

	updateStr := fmt.Sprintf("update posts set deleted = true where id = %v returning (id)", idForm)
	// fmt.Printf("updateStr: %v\n", updateStr)

	var id int
	err = rs.DBConn.QueryRow(context.Background(), updateStr).Scan(&id)
	fmt.Printf("res: %v\n", id)

	if err != nil {
		fmt.Printf("Delete posts error. Post Id: %v.\n %v\n", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
