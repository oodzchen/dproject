package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/utils"
)

type ArticleResource struct {
	Renderer
	// DBConn *pgx.Conn
	// DBPool *pgxpool.Pool
	store     store.ArticleStore
	sessStore *sessions.CookieStore
}

func NewArticleResource(tmpl *template.Template, store store.ArticleStore, sessStore *sessions.CookieStore) *ArticleResource {
	return &ArticleResource{
		Renderer{tmpl, sessStore},
		store,
		sessStore,
	}
}

func (ar *ArticleResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Get("/", ar.List)
	rt.Post("/", ar.Submit)
	rt.Get("/new", ar.CreatePage)

	rt.Route("/{id}", func(r chi.Router) {
		r.Get("/", ar.Get)
		r.Get("/edit", ar.EditPage)
		r.Post("/edit", ar.Update)
		r.Post("/delete", ar.Delete)
	})

	return rt
}

func (ar *ArticleResource) List(w http.ResponseWriter, r *http.Request) {
	list, err := ar.store.List()
	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
		return
	}

	type ListData struct {
		Articles     []*model.Article
		ArticleTotal int
	}

	ar.Render(w, r, "article_list", &PageData{Title: "Home", Data: &ListData{list, len(list)}})
}

func (ar *ArticleResource) CreatePage(w http.ResponseWriter, r *http.Request) {
	// fmt.Printf("r.URL:%#v\n", r.URL)
	if !IsLogin(ar.sessStore, w, r) {
		var targetUrl string
		if r.TLS != nil {
			targetUrl = "https://" + r.Host + r.URL.Path
		} else {
			targetUrl = "http://" + r.Host + r.URL.Path
		}
		http.Redirect(w, r, "/login?target="+url.QueryEscape(targetUrl), http.StatusFound)
		return
	}

	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	// tempUserId := Session.Values["tempUserId"]
	// fmt.Printf("tempUserId: %v\n", tempUserId)

	// var data PostPageData
	var pageTitle string
	var data *model.Article

	if idParam == "" {
		pageTitle = "Create"
		data = &model.Article{}
	} else {
		rId, err := strconv.Atoi(idParam)

		if err != nil {
			utils.HttpError("", err, w, http.StatusBadRequest)
			return
		}
		// postData, _ := ar.getPostData(idParam)
		postData, err := ar.store.Item(rId)
		if err != nil {
			utils.HttpError("", err, w, http.StatusInternalServerError)
			return
		}
		pageTitle = fmt.Sprintf("Edit - %v", postData.Title)
		data = postData
	}

	ar.Render(w, r, "create", &PageData{Title: pageTitle, Data: data})
}

func (ar *ArticleResource) Submit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		utils.HttpError("", err, w, http.StatusBadRequest)
	}

	authorId, err := strconv.Atoi(r.Form.Get("author_id"))
	if err != nil {
		utils.HttpError("", err, w, http.StatusBadRequest)
		return
	}

	id, err := ar.store.Create(&model.Article{
		Title:    r.Form.Get("title"),
		AuthorId: authorId,
		Content:  r.Form.Get("content"),
	})

	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/articles/%v", id), http.StatusFound)
}

func (ar *ArticleResource) Update(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		utils.HttpError("", err, w, http.StatusBadRequest)
	}
	// fmt.Printf("r.Form: %v\n", r.Form)

	rId, err := strconv.Atoi(r.Form.Get("id"))
	authorId, err := strconv.Atoi(r.Form.Get("author_id"))

	if err != nil {
		utils.HttpError("", err, w, http.StatusBadRequest)
		return
	}

	id, err := ar.store.Update(&model.Article{
		Title:    r.Form.Get("title"),
		AuthorId: authorId,
		Content:  r.Form.Get("content"),
		Id:       rId,
	})

	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/articles/%d", id), http.StatusFound)
}

func (ar *ArticleResource) Get(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	rId, err := strconv.Atoi(idParam)

	if err != nil {
		utils.HttpError("", err, w, http.StatusBadRequest)
		return
	}

	// postData, err := ar.getPostData((idParam))
	articleData, err := ar.store.Item(rId)
	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
		return
	}

	articleData.TransformNewlines()
	ar.Render(w, r, "article", &PageData{Title: articleData.Title, Data: articleData})
}

func (ar *ArticleResource) EditPage(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	fmt.Printf("idParam: %v\n", idParam)

	rId, err := strconv.Atoi(idParam)

	if err != nil {
		utils.HttpError("", err, w, http.StatusBadRequest)
		return
	}

	postData, err := ar.store.Item(rId)
	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
		return
	}
	ar.Render(w, r, "create", &PageData{Title: postData.Title, Data: postData})
}

func (ar *ArticleResource) Delete(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		utils.HttpError("", err, w, http.StatusBadRequest)
	}

	idForm := r.Form.Get("id")

	rId, err := strconv.Atoi(idForm)

	if err != nil {
		utils.HttpError("", err, w, http.StatusBadRequest)
		return
	}

	err = ar.store.Delete(rId)

	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}
