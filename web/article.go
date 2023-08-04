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
	rt.Get("/new", ar.CreatePage)

	rt.Route("/{id}", func(r chi.Router) {
		r.Get("/", ar.Item)
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

	paramReplyTo := r.Form.Get("reply_to")
	// fmt.Printf("paramReplyTo: %s\n", paramReplyTo)

	var isReply bool
	var replyTo int
	if paramReplyTo != "" {
		num, err := strconv.Atoi(paramReplyTo)
		if err != nil {
			utils.HttpError("", err, w, http.StatusBadRequest)
			return
		}
		replyTo = num
		// fmt.Printf("replyTo:%d\n", replyTo)
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

	// fmt.Printf("article: %+v\n", article)

	article.Sanitize()

	err = article.Valid(isReply)
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
	// fmt.Printf("r.Form: %v\n", r.Form)

	rId, err := strconv.Atoi(r.Form.Get("id"))
	authorId, err := strconv.Atoi(r.Form.Get("author_id"))

	if err != nil {
		utils.HttpError("", err, w, http.StatusBadRequest)
		return
	}

	article := &model.Article{
		Title:    r.Form.Get("title"),
		AuthorId: authorId,
		Content:  r.Form.Get("content"),
		Id:       rId,
	}
	article.Sanitize()

	err = article.Valid(false)
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

	if article.Deleted {
		utils.HttpError("the article is gone", err, w, http.StatusGone)
		return
	}

	replyData, err := ar.store.GetReplies(articleId)
	if err != nil {
		utils.HttpError("", err, w, http.StatusInternalServerError)
		return
	}
	// articleTree := &articleWithReplies{
	// 	article,
	// 	make([]*articleWithReplies, 0),
	// }

	if len(replyData) > 0 {
		// articleTree = formatArticlesToTree(articleTree, replyData)
		article, err = genArticleTree(article, replyData)
		if err != nil {
			utils.HttpError("", err, w, http.StatusInternalServerError)
			return
		}
	}

	// fmt.Printf("articleTree.Replies: %+v\n", articleTree.Replies)

	ar.Render(w, r, "article", &PageData{Title: article.Title, Data: article})
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
			return nil, errors.New("no reply to the root in the list")
		}
	}

	for _, item := range list {
		if replies, ok := nodeMap[item.Id]; ok {
			item.Replies = replies
		}
	}
	return root, nil
}

// func formatArticlesToTree(rootAR *articleWithReplies, list []*model.Article) *articleWithReplies {
// 	for _, article := range list {
// 		if article.ReplyTo == rootAR.Article.Id {
// 			currAR := &articleWithReplies{
// 				Article: article,
// 				Replies: make([]*articleWithReplies, 0),
// 			}
// 			currAR = formatArticlesToTree(currAR, list)
// 			rootAR.Replies = append(rootAR.Replies, currAR)
// 		}
// 	}

// 	return rootAR
// }

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

	refereUrl := r.Referer()
	from := r.Form.Get("from")

	if from == "reply" && refereUrl != "" {
		http.Redirect(w, r, refereUrl, http.StatusFound)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
