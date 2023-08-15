package web

import (
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
	"github.com/pkg/errors"
)

type ArticleResource struct {
	*Renderer
	// DBConn *pgx.Conn
	// DBPool *pgxpool.Pool
	store *store.Store
}

func NewArticleResource(tmpl *template.Template, store *store.Store, sessStore *sessions.CookieStore) *ArticleResource {
	return &ArticleResource{
		&Renderer{tmpl, sessStore},
		store,
	}
}

func (ar *ArticleResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Get("/", ar.List)
	rt.Post("/", ar.Submit)
	rt.Get("/new", ar.FormPage)

	rt.Route("/{id}", func(r chi.Router) {
		r.Get("/", ar.Item)
		r.Get("/edit", ar.FormPage)
		r.Post("/edit", ar.Update)
		r.Get("/delete", ar.DeletePage)
		r.Post("/delete", ar.Delete)
		r.Get("/reply", ar.ReplyPage)
	})

	return rt
}

func (ar *ArticleResource) List(w http.ResponseWriter, r *http.Request) {
	list, err := ar.store.Article.List()
	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	type ListData struct {
		Articles     []*model.Article
		ArticleTotal int
	}

	pageData := &PageData{Title: "Home", Data: &ListData{list, len(list)}}

	if r.URL.Path == "/settings" {
		pageData.Type = PageTypeSettings
	}

	ar.Render(w, r, "article_list", pageData)
}

func (ar *ArticleResource) FormPage(w http.ResponseWriter, r *http.Request) {
	if !IsLogin(ar.sessStore, w, r) {
		// http.Redirect(w, r, "/login?target="+r.URL.Path, http.StatusFound)
		sess := ar.Session("one-cookie", w, r)
		sess.SetValue("target_url", "/articles/new")
		//
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	id := chi.URLParam(r, "id")
	var pageTitle string
	var data *model.Article

	if id == "" {
		pageTitle = "Create"
		data = &model.Article{}
	} else {
		rId, err := strconv.Atoi(id)

		if err != nil {
			ar.Error("", err, w, r, http.StatusBadRequest)
			return
		}

		currUserId, err := GetLoginUserId(ar.sessStore, w, r)
		if err != nil {
			ar.Error("Please login", err, w, r, http.StatusUnauthorized)
			return
		}

		article, err := ar.store.Article.Item(rId)
		if err != nil {
			ar.Error("", err, w, r, http.StatusInternalServerError)
			return
		}

		if article.AuthorId != currUserId {
			http.Redirect(w, r, fmt.Sprintf("/articles/%d", rId), http.StatusFound)
			return
		}

		pageTitle = fmt.Sprintf("Edit - %s", article.Title)

		article.UpdateDisplayTitle()
		data = article
	}

	ar.Render(w, r, "create", &PageData{Title: pageTitle, Data: data})
}

func (ar *ArticleResource) Submit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
	}

	paramReplyTo := r.Form.Get("reply_to")

	var isReply bool
	var replyTo int
	if paramReplyTo != "" {
		num, err := strconv.Atoi(paramReplyTo)
		if err != nil {
			ar.Error("", err, w, r, http.StatusBadRequest)
			return
		}
		replyTo = num
		isReply = replyTo > 0
	}

	authorId, err := GetLoginUserId(ar.sessStore, w, r)
	if err != nil {
		sess, _ := ar.sessStore.Get(r, "one-cookie")
		var callbackUrl string
		if isReply {
			callbackUrl = fmt.Sprintf("/articles/%d", replyTo)
		} else {
			callbackUrl = "/create"
		}
		sess.Values["login_callback"] = callbackUrl
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	article := &model.Article{
		Title:    r.Form.Get("title"),
		AuthorId: authorId,
		Content:  r.Form.Get("content"),
		ReplyTo:  replyTo,
	}

	// if isReply {
	// 	toArticle, err := ar.store.Article.Item(replyTo)
	// 	if err != nil {
	// 		ar.Error("", err, w, r, http.StatusInternalServerError)
	// 		return
	// 	}
	// 	// utils.PrintJSONf("toArticle: ", toArticle)

	// 	article.ReplyDepth = toArticle.ReplyDepth + 1

	// 	if toArticle.ReplyDepth == 0 {
	// 		article.ReplyRootArticleId = toArticle.Id
	// 	} else {
	// 		article.ReplyRootArticleId = toArticle.ReplyRootArticleId
	// 	}
	// }

	// fmt.Printf("article: %+v\n", article)
	// utils.PrintJSONf("create article: ", article)

	article.Sanitize()

	err = article.Valid(false)
	if err != nil {
		ar.Error(err.Error(), err, w, r, http.StatusBadRequest)
		return
	}

	id, err := ar.store.Article.Create(article)

	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	refererUrl := r.Referer()

	if isReply && refererUrl != "" {
		http.Redirect(w, r, refererUrl, http.StatusFound)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/articles/%d", id), http.StatusFound)
	}
}

func (ar *ArticleResource) Update(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
	}

	rId, err := strconv.Atoi(r.Form.Get("id"))
	replyDepth, err := strconv.Atoi(r.Form.Get("reply_depth"))
	// fmt.Printf("replyDepth: %d\n", replyDepth)
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	isReply := replyDepth > 0
	// fmt.Printf("isReply: %t\n", isReply)

	article := &model.Article{
		Content:    r.Form.Get("content"),
		Id:         rId,
		ReplyDepth: replyDepth,
	}
	if !isReply {
		article.Title = r.Form.Get("title")
	}
	article.Sanitize()

	err = article.Valid(true)
	if err != nil {
		ar.Error(err.Error(), err, w, r, http.StatusBadRequest)
		return
	}

	id, err := ar.store.Article.Update(article)

	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/articles/%d", id), http.StatusFound)
}

func (ar *ArticleResource) Item(w http.ResponseWriter, r *http.Request) {
	ar.handleItem(w, r, false)
}

func (ar *ArticleResource) handleItem(w http.ResponseWriter, r *http.Request, delPage bool) {
	idParam := chi.URLParam(r, "id")
	// fmt.Printf("idParam: %v\n", idParam)

	articleId, err := strconv.Atoi(idParam)

	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	article, err := ar.store.Article.Item(articleId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ar.Error("the article is gone", err, w, r, http.StatusGone)
		} else {
			ar.Error("", err, w, r, http.StatusInternalServerError)
		}
		return
	}

	if delPage {
		currUserId, err := GetLoginUserId(ar.sessStore, w, r)
		if err != nil {
			ar.Error("Please login", err, w, r, http.StatusUnauthorized)
			return
		}

		if article.AuthorId != currUserId {
			http.Redirect(w, r, fmt.Sprintf("/articles/%d", articleId), http.StatusFound)
			return
		}
	}

	// if article.Deleted {
	// 	ar.Error("the article is gone", err, w, r, http.StatusGone)
	// 	return
	// }

	replyData, err := ar.store.Article.GetReplies(articleId)
	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	// utils.PrintJSONf("article: ", article)
	// utils.PrintJSONf("replyData: ", replyData)
	if len(replyData) > 0 {
		for _, item := range replyData {
			item.FormatDeleted()
		}

		article, err = genArticleTree(article, replyData)
		if err != nil {
			// ar.Error("", err, w, r, http.StatusInternalServerError)
			fmt.Printf("generate article tree error: %v", err)
		}
	}

	article.UpdateDisplayTitle()

	if article.Deleted {
		w.WriteHeader(http.StatusGone)
	}

	type itemPageData struct {
		Article *model.Article
		DelPage bool
	}

	ar.Render(w, r, "article", &PageData{Title: article.DisplayTitle, Data: &itemPageData{
		article,
		delPage,
	}})
}

func genArticleTree(root *model.Article, list []*model.Article) (*model.Article, error) {
	nodeMap := make(map[int][]*model.Article)
	for _, item := range list {
		nodeMap[item.ReplyTo] = append(nodeMap[item.ReplyTo], item)
	}

	if replies, ok := nodeMap[root.Id]; ok {
		root.Replies = replies
	} else {
		if len(list) > 0 {
			return root, errors.New("no reply to the root in the list")
		}
	}

	for _, item := range list {
		if replies, ok := nodeMap[item.Id]; ok {
			item.Replies = replies
		}
	}
	return root, nil
}

func (ar *ArticleResource) Delete(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	idForm := r.Form.Get("id")

	rId, err := strconv.Atoi(idForm)
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	confirmText := r.Form.Get("confirm_del")
	if confirmText != "yes" {
		if confirmText == "no" {
			http.Redirect(w, r, fmt.Sprintf("/articles/%d", rId), http.StatusFound)
		} else {
			ar.Error("Delete failed", err, w, r, http.StatusBadRequest)
		}
		return
	}

	currUserId, err := GetLoginUserId(ar.sessStore, w, r)
	if err != nil {
		ar.Error("Please login", err, w, r, http.StatusUnauthorized)
		return
	}

	article, err := ar.store.Article.Item(rId)
	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	if article.AuthorId != currUserId {
		http.Redirect(w, r, fmt.Sprintf("/articles/%d", rId), http.StatusFound)
		ar.Error("Forbidden", err, w, r, http.StatusForbidden)
		return
	}

	rootArticleId, err := ar.store.Article.Delete(rId, currUserId)
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/articles/%d", rootArticleId), http.StatusFound)
}

func (ar *ArticleResource) DeletePage(w http.ResponseWriter, r *http.Request) {
	ar.handleItem(w, r, true)
}

func (ar *ArticleResource) ReplyPage(w http.ResponseWriter, r *http.Request) {
	// ar.handleItem(w, r, false)
	http.Redirect(w, r, fmt.Sprintf("/articles/%s", chi.URLParam(r, "id")), http.StatusFound)
}
