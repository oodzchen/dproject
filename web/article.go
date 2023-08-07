package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"text/template"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/utils"
	"github.com/pkg/errors"
)

type ArticleResource struct {
	Renderer
	// DBConn *pgx.Conn
	// DBPool *pgxpool.Pool
	store     store.ArticleStore
	sessStore *sessions.CookieStore
}

type articleWithReplies struct {
	Article *model.Article
	Replies []*articleWithReplies
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
	rt.Get("/new", ar.FormPage)

	rt.Route("/{id}", func(r chi.Router) {
		r.Get("/", ar.Item)
		r.Get("/edit", ar.FormPage)
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

func (ar *ArticleResource) FormPage(w http.ResponseWriter, r *http.Request) {
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

	id := chi.URLParam(r, "id")
	var pageTitle string
	var data *model.Article

	if id == "" {
		pageTitle = "Create"
		data = &model.Article{}
	} else {
		rId, err := strconv.Atoi(id)

		if err != nil {
			utils.HttpError("", err, w, http.StatusBadRequest)
			return
		}
		// postData, _ := ar.getPostData(idParam)
		article, err := ar.store.Item(rId)
		if err != nil {
			utils.HttpError("", err, w, http.StatusInternalServerError)
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
		utils.HttpError("", err, w, http.StatusBadRequest)
	}

	paramReplyTo := r.Form.Get("reply_to")

	var isReply bool
	var replyTo int
	if paramReplyTo != "" {
		num, err := strconv.Atoi(paramReplyTo)
		if err != nil {
			utils.HttpError("", err, w, http.StatusBadRequest)
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

	if isReply {
		toArticle, err := ar.store.Item(replyTo)
		if err != nil {
			utils.HttpError("", err, w, http.StatusInternalServerError)
			return
		}
		utils.PrintJSONf("toArticle: ", toArticle)

		article.ReplyDepth = toArticle.ReplyDepth + 1

		if toArticle.ReplyDepth == 0 {
			article.ReplyRootArticleId = toArticle.Id
		} else {
			article.ReplyRootArticleId = toArticle.ReplyRootArticleId
		}
	}

	// fmt.Printf("article: %+v\n", article)
	utils.PrintJSONf("create article: ", article)

	article.Sanitize()

	err = article.Valid(false)
	if err != nil {
		utils.HttpError(err.Error(), err, w, http.StatusBadRequest)
		return
	}

	id, err := ar.store.Create(article)

	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
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
		utils.HttpError("", err, w, http.StatusBadRequest)
	}

	rId, err := strconv.Atoi(r.Form.Get("id"))
	replyDepth, err := strconv.Atoi(r.Form.Get("reply_depth"))
	// fmt.Printf("replyDepth: %d\n", replyDepth)
	if err != nil {
		utils.HttpError("", err, w, http.StatusBadRequest)
		return
	}

	isReply := replyDepth > 0
	fmt.Printf("isReply: %t\n", isReply)

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
		utils.HttpError(err.Error(), err, w, http.StatusBadRequest)
		return
	}

	id, err := ar.store.Update(article)

	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/articles/%d", id), http.StatusFound)
}

func (ar *ArticleResource) Item(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	// fmt.Printf("idParam: %v\n", idParam)

	articleId, err := strconv.Atoi(idParam)

	if err != nil {
		utils.HttpError("", err, w, http.StatusBadRequest)
		return
	}

	article, err := ar.store.Item(articleId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			utils.HttpError("the article is gone", err, w, http.StatusGone)
		} else {
			utils.HttpError("", err, w, http.StatusInternalServerError)
		}
		return
	}

	// if article.Deleted {
	// 	utils.HttpError("the article is gone", err, w, http.StatusGone)
	// 	return
	// }

	replyData, err := ar.store.GetReplies(articleId)
	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
		return
	}

	// utils.PrintJSONf("article: ", article)

	utils.PrintJSONf("replyData: ", replyData)
	if len(replyData) > 0 {
		for _, item := range replyData {
			item.FormatDeleted()
		}

		article, err = genArticleTree(article, replyData)
		if err != nil {
			// utils.HttpError("", err, w, http.StatusInternalServerError)
			fmt.Printf("generate article tree error: %v", err)
		}
	}

	article.UpdateDisplayTitle()

	if article.Deleted {
		w.WriteHeader(http.StatusGone)
	}
	ar.Render(w, r, "article", &PageData{Title: article.DisplayTitle, Data: article})
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

	refereUrl := r.Referer()
	from := r.Form.Get("from")

	if from == "reply" && refereUrl != "" {
		http.Redirect(w, r, refereUrl, http.StatusFound)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
