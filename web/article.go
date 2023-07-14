package web

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
)

type ArticleResource struct {
	Renderer
	// DBConn *pgx.Conn
	// DBPool *pgxpool.Pool
	store store.ArticleStore
}

func NewArticleResource(tmpl *template.Template, store store.ArticleStore) *ArticleResource {
	return &ArticleResource{
		Renderer{Tmpl: tmpl},
		store,
	}
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
	list, err := rs.store.List()
	if err != nil {
		fmt.Printf("Query database error: %v\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	type ListData struct {
		Articles     []*model.Article
		ArticleTotal int
	}

	rs.Render(w, r, "index", &PageData{Title: "Home - Dproject", Data: &ListData{list, len(list)}})
}

func (rs *ArticleResource) CreatePage(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	// tempUserId := Session.Values["tempUserId"]
	// fmt.Printf("tempUserId: %v\n", tempUserId)

	// var data PostPageData
	var pageTitle string
	var data *model.Article

	if idParam == "" {
		pageTitle = "Create - Dproject"
		data = &model.Article{}
	} else {
		rId, err := strconv.Atoi(idParam)

		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		// postData, _ := rs.getPostData(idParam)
		postData, err := rs.store.Item(rId)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		pageTitle = fmt.Sprintf("Edit - %v", postData.Title)
		data = postData
	}

	rs.Render(w, r, "create", &PageData{Title: pageTitle, Data: data})
}

func (rs *ArticleResource) Submit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	authorId, err := strconv.Atoi(r.Form["author_id"][0])
	if err != nil {
		http.Error(w, "Bad Reqeust", http.StatusBadRequest)
		return
	}

	id, err := rs.store.Create(&model.Article{
		Title:    r.Form["title"][0],
		AuthorId: authorId,
		Content:  r.Form["content"][0],
	})

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

	rId, err := strconv.Atoi(r.Form["id"][0])
	authorId, err := strconv.Atoi(r.Form["author_id"][0])

	if err != nil {
		fmt.Printf("Convert id or authorId error: %s ", err.Error())
		http.Error(w, "Bad Reqeust", http.StatusBadRequest)
		return
	}

	id, err := rs.store.Update(&model.Article{
		Title:    r.Form["title"][0],
		AuthorId: authorId,
		Content:  r.Form["content"][0],
		Id:       rId,
	})

	if err != nil {
		fmt.Printf("Update posts error. Post Id: %v.\n %v\n", id, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/articles/%v", id), http.StatusFound)
}

func (rs *ArticleResource) Get(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	rId, err := strconv.Atoi(idParam)

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// postData, err := rs.getPostData((idParam))
	postData, err := rs.store.Item(rId)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	re := regexp.MustCompile(`\r`)

	postData.Content = re.ReplaceAllString(postData.Content, "<br/>")
	rs.Render(w, r, "article", &PageData{Title: postData.Title, Data: postData})
}

func (rs *ArticleResource) EditPage(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	rId, err := strconv.Atoi(idParam)

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// postData, err := rs.getPostData((idParam))
	postData, err := rs.store.Item(rId)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	rs.Render(w, r, "create", &PageData{Title: postData.Title, Data: postData})
}

func (rs *ArticleResource) Delete(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	idForm := r.Form["id"][0]

	rId, err := strconv.Atoi(idForm)

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// updateStr := fmt.Sprintf("update posts set deleted = true where id = %v returning (id)", idForm)
	// // fmt.Printf("updateStr: %v\n", updateStr)

	// var id int
	// err = rs.DBPool.QueryRow(context.Background(), updateStr).Scan(&id)
	// fmt.Printf("res: %v\n", id)

	err = rs.store.Delete(rId)

	if err != nil {
		fmt.Printf("Delete posts error. Post Id: %v.\n %v\n", rId, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
