package web

import (
	"fmt"
	"net/http"
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

// func ArticleContext(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		ctx := context.WithValue(r.Context(), "global_tip_status", nil)
// 		next.ServeHTTP(w, r.WithContext(ctx))
// 	})
// }

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

	authorId, err := strconv.Atoi(r.Form.Get("author_id"))
	if err != nil {
		http.Error(w, "Missing field required: author_id", http.StatusBadRequest)
		return
	}

	id, err := rs.store.Create(&model.Article{
		Title:    r.Form.Get("title"),
		AuthorId: authorId,
		Content:  r.Form.Get("content"),
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
	// fmt.Printf("r.Form: %v\n", r.Form)

	rId, err := strconv.Atoi(r.Form.Get("id"))
	authorId, err := strconv.Atoi(r.Form.Get("author_id"))

	if err != nil {
		fmt.Printf("Convert id or authorId error: %s ", err.Error())
		http.Error(w, "Bad Reqeust", http.StatusBadRequest)
		return
	}

	id, err := rs.store.Update(&model.Article{
		Title:    r.Form.Get("title"),
		AuthorId: authorId,
		Content:  r.Form.Get("content"),
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
	articleData, err := rs.store.Item(rId)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	articleData.TransformNewlines()
	rs.Render(w, r, "article", &PageData{Title: articleData.Title, Data: articleData})
}

func (rs *ArticleResource) EditPage(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	rId, err := strconv.Atoi(idParam)

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

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

	idForm := r.Form.Get("id")

	rId, err := strconv.Atoi(idForm)

	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	err = rs.store.Delete(rId)

	if err != nil {
		fmt.Printf("Delete posts error. Post Id: %v.\n %v\n", rId, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
