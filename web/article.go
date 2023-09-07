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
	"github.com/oodzchen/dproject/service"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/utils"
	"github.com/pkg/errors"
)

type ArticleResource struct {
	*Renderer
	articleSrv *service.Article
	// DBConn *pgx.Conn
	// DBPool *pgxpool.Pool
	// store *store.Store
}

func NewArticleResource(tmpl *template.Template, store *store.Store, sessStore *sessions.CookieStore, router *chi.Mux) *ArticleResource {
	return &ArticleResource{
		&Renderer{tmpl, sessStore, router, store},
		&service.Article{
			Store: store,
		},
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
		r.Post("/vote", ar.Vote)
		r.Post("/save", ar.Save)
		r.Post("/react", ar.React)
	})

	return rt
}

func (ar *ArticleResource) List(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	paramPage := r.Form.Get("page")
	sort := r.Form.Get("sort")

	var sortType model.ArticleSortType
	if sort == "" {
		sortType = model.ListSortBest
	} else {
		sortType = model.ArticleSortType(sort)
	}
	// fmt.Println("paramPage:", paramPage)
	page, err := strconv.Atoi(paramPage)
	if err != nil {
		// fmt.Printf("page err %v\n", err)
		page = 1
	}

	pageSize, err := strconv.Atoi(r.Form.Get("page_size"))
	if err != nil {
		pageSize = 50
	}

	currUserId := ar.GetLoginedUserId(w, r)

	wholeList, err := ar.store.Article.List(0, -1, currUserId)
	// list, err := ar.store.Article.List(page, pageSize)
	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	for _, item := range wholeList {
		item.CalcScore()
		item.CalcWeight()
	}

	// fmt.Println("sortType: ", sortType)

	wholeArticleList := model.NewArticleList(wholeList, sortType, page, pageSize)
	// sort.Sort(wholeArticleList)
	wholeArticleList.Sort(sortType)

	list := wholeArticleList.PagingList(page, pageSize)

	for _, item := range list {
		// fmt.Println("item.VoteScore: ", item.VoteScore)
		item.FormatTimeStr()
		item.FormatNullValues()
		item.UpdateDisplayTitle()
		item.GenSummary(200)
	}

	// total, err := ar.store.Article.Count()
	// if err != nil {
	// 	ar.Error("", err, w, r, http.StatusInternalServerError)
	// 	return
	// }

	type ListData struct {
		Articles     []*model.Article
		ArticleTotal int
		CurrPage     int
		PageSize     int
		TotalPage    int
		SortType     model.ArticleSortType
	}

	pageData := &PageData{
		Data: &ListData{
			list,
			wholeArticleList.Total,
			wholeArticleList.CurrPage,
			wholeArticleList.PageSize,
			wholeArticleList.TotalPage,
			wholeArticleList.SortType,
		},
	}

	ar.Render(w, r, "article_list", pageData)
}

func (ar *ArticleResource) FormPage(w http.ResponseWriter, r *http.Request) {
	if !IsLogin(ar.sessStore, w, r) {
		// http.Redirect(w, r, "/login?target="+r.URL.Path, http.StatusFound)
		sess := ar.Session("one", w, r)
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

		article, err := ar.store.Article.Item(rId, currUserId)
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

	title := r.Form.Get("title")
	content := r.Form.Get("content")
	paramReplyTo := r.Form.Get("reply_to")
	// rootId := r.Form.Get("root")

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
		sess, _ := ar.sessStore.Get(r, "one")
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

	// id, err := ar.store.Article.Create(article.Title, article.Content, authorId, replyTo)
	var id int
	if isReply {
		id, err = ar.articleSrv.Reply(replyTo, content, authorId)
	} else {
		id, err = ar.articleSrv.Create(title, content, authorId, 0)
	}
	// id, err := ar.articleSrv.Create(title, content, authorId, replyTo)
	if err != nil {
		if errors.Is(err, model.ErrValidArticleFailed) {
			ar.Error(err.Error(), err, w, r, http.StatusBadRequest)
		} else {
			ar.Error("", err, w, r, http.StatusInternalServerError)
		}
		return
	}

	refererUrl := r.Referer()
	referer, _ := url.Parse(refererUrl)

	if isReply && refererUrl != "" && IsRegisterdPage(referer, ar.router) {
		// http.Redirect(w, r, refererUrl, http.StatusFound)
		http.Redirect(w, r, fmt.Sprintf("/articles/%d?sort=latest#ar_%d", replyTo, id), http.StatusFound)
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

	id, err := ar.store.Article.Update(article, []string{"Content"})

	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/articles/%d", id), http.StatusFound)
}

func (ar *ArticleResource) Item(w http.ResponseWriter, r *http.Request) {
	ar.handleItem(w, r, ArticlePageDetail)
}

type ArticlePageType string

const (
	ArticlePageDel    ArticlePageType = "del"
	ArticlePageReply                  = "reply"
	ArticlePageDetail                 = "detail"
)

func (ar *ArticleResource) handleItem(w http.ResponseWriter, r *http.Request, pageType ArticlePageType) {
	idParam := chi.URLParam(r, "id")
	sortType := r.URL.Query().Get("sort")
	pageQ := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageQ)

	// fmt.Println("sort type", sortType)
	// fmt.Printf("idParam: %v\n", idParam)

	articleId, err := strconv.Atoi(idParam)
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	currUserId := ar.GetLoginedUserId(w, r)

	articleTreeList, err := ar.store.Article.ItemTree(articleId, currUserId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// http.Redirect(w, r, "/404", http.StatusNotFound)
			ar.Error("", nil, w, r, http.StatusNotFound)
		} else {
			ar.Error("", err, w, r, http.StatusInternalServerError)
		}
		return
	}

	if len(articleTreeList) == 0 {
		// http.Redirect(w, r, "/404", http.StatusNotFound)
		ar.Error("", nil, w, r, http.StatusNotFound)
	}

	var rootArticle *model.Article
	for _, item := range articleTreeList {
		item.FormatNullValues()
		item.FormatTimeStr()
		item.CalcScore()
		item.CalcWeight()

		if item.Id == articleId {
			rootArticle = item
		}
	}

	// if rootArticle.Id == 0 {
	// 	ar.Error("the article is gone", err, w, r, http.StatusGone)
	// }

	if pageType == ArticlePageDel {
		// currUserId, err := GetLoginUserId(ar.sessStore, w, r)
		if err != nil {
			ar.Error("Please login", err, w, r, http.StatusUnauthorized)
			return
		}

		if rootArticle.AuthorId != currUserId {
			http.Redirect(w, r, fmt.Sprintf("/articles/%d", articleId), http.StatusFound)
			return
		}
	}

	for _, item := range articleTreeList {
		item.FormatDeleted()
	}

	rootArticle, err = genArticleTree(rootArticle, articleTreeList)
	if err != nil {
		// ar.Error("", err, w, r, http.StatusInternalServerError)
		fmt.Printf("generate article tree error: %v", err)
	}

	replySort := model.ReplySortBest
	if model.ValidReplySort(sortType) {
		replySort = model.ArticleSortType(sortType)
	}
	// fmt.Println("replySort: ", replySort)
	rootArticle = sortArticleTree(rootArticle, replySort)
	rootArticle = pagingArticleTree(rootArticle, page)

	rootArticle.UpdateDisplayTitle()

	if rootArticle.Deleted {
		w.WriteHeader(http.StatusGone)
	}

	type itemPageData struct {
		Article *model.Article
		// DelPage  bool
		MaxDepth     int
		PageType     ArticlePageType
		ReactOptions []model.ArticleReact
		ReactMap     map[model.ArticleReact]string
	}

	ar.Render(w, r, "article", &PageData{Title: rootArticle.DisplayTitle, Data: &itemPageData{
		rootArticle,
		// delPage,
		utils.GetReplyDepthSize(),
		pageType,
		model.GetArticleReactOptions(),
		model.GetArticleReactEmojiMap(),
	}})
}

const DefaultTopReplyPageSize = 50
const DefaultReplyPageSize = 10

func genArticleTree(root *model.Article, list []*model.Article) (*model.Article, error) {
	nodeMap := make(map[int][]*model.Article)
	for _, item := range list {
		nodeMap[item.ReplyTo] = append(nodeMap[item.ReplyTo], item)
	}

	if replies, ok := nodeMap[root.Id]; ok {
		root.Replies = model.NewArticleList(replies, model.ReplySortBest, 1, DefaultTopReplyPageSize)
	} else {
		if len(list) > 0 {
			return root, errors.New("no reply to the root in the list")
		}
	}

	for _, item := range list {
		if replies, ok := nodeMap[item.Id]; ok && item.Id != root.Id {
			item.Replies = model.NewArticleList(replies, model.ReplySortBest, 1, DefaultReplyPageSize)
		}
	}
	return root, nil
}

func sortArticleTree(root *model.Article, sortType model.ArticleSortType) *model.Article {
	if root.Replies != nil && root.Replies.Len() > 0 {
		root.Replies.Sort(sortType)
		for idx, item := range root.Replies.List {
			root.Replies.List[idx] = sortArticleTree(item, sortType)
		}
	}
	return root
}

func pagingArticleTree(root *model.Article, page int) *model.Article {
	if root.Replies != nil && root.Replies.Len() > 0 {
		if page < 1 {
			page = 1
		}
		if page > root.Replies.TotalPage {
			page = root.Replies.TotalPage
		}

		root.Replies.List = root.Replies.PagingList(page, root.Replies.PageSize)
		for idx, item := range root.Replies.List {
			root.Replies.List[idx] = pagingArticleTree(item, 1)
		}
	}
	return root
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

	// confirmText := r.Form.Get("confirm_del")
	// if confirmText != "yes" {
	// 	if confirmText == "no" {
	// 		http.Redirect(w, r, fmt.Sprintf("/articles/%d", rId), http.StatusFound)
	// 	} else {
	// 		ar.Error("Delete failed", err, w, r, http.StatusBadRequest)
	// 	}
	// 	return
	// }

	currUserId, err := GetLoginUserId(ar.sessStore, w, r)
	if err != nil {
		ar.Error("Please login", err, w, r, http.StatusUnauthorized)
		return
	}

	article, err := ar.store.Article.Item(rId, currUserId)
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

	ar.Session("one", w, r).Flash("Delete article successfully")

	http.Redirect(w, r, fmt.Sprintf("/articles/%d", rootArticleId), http.StatusFound)
}

func (ar *ArticleResource) DeletePage(w http.ResponseWriter, r *http.Request) {
	ar.handleItem(w, r, ArticlePageDel)
}

func (ar *ArticleResource) ReplyPage(w http.ResponseWriter, r *http.Request) {
	ar.handleItem(w, r, ArticlePageReply)
	// http.Redirect(w, r, fmt.Sprintf("/articles/%s", chi.URLParam(r, "id")), http.StatusFound)
}

func (ar *ArticleResource) Vote(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	voteType := r.Form.Get("type")
	if !model.IsValidVoteType(model.VoteType(voteType)) {
		ar.Error("", errors.New("vote type not valid: "+voteType), w, r, http.StatusBadRequest)
		return
	}

	articleIdS := chi.URLParam(r, "id")
	articleId, err := strconv.Atoi(articleIdS)
	if err != nil {
		ar.Error("", errors.New("get article id failed"), w, r, http.StatusBadRequest)
		return
	}

	rootId := r.Form.Get("root")

	userId := ar.GetLoginedUserId(w, r)
	if userId != 0 {
		err = ar.store.Article.Vote(articleId, userId, voteType)
		if err != nil {
			ar.ServerError("", err, w, r)
			return
		}
	} else {
		ar.ToLogin(w, r)
		return
	}

	referer := r.Referer()
	refererUrl, _ := url.Parse(r.Referer())
	if IsRegisterdPage(refererUrl, ar.router) {
		if rootId != "" && rootId != "0" && rootId != articleIdS {
			http.Redirect(w, r, fmt.Sprintf("/articles/%s#ar_%s", rootId, articleIdS), http.StatusFound)
		} else {
			http.Redirect(w, r, referer, http.StatusFound)
		}
	} else {
		http.Redirect(w, r, "/articles/"+articleIdS, http.StatusFound)
	}
}

func (ar *ArticleResource) Save(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	articleIdS := chi.URLParam(r, "id")
	articleId, err := strconv.Atoi(articleIdS)
	if err != nil {
		ar.Error("", errors.New("get article id failed"), w, r, http.StatusBadRequest)
		return
	}

	userId := ar.GetLoginedUserId(w, r)
	if userId != 0 {
		err = ar.store.Article.Save(articleId, userId)
		if err != nil {
			ar.ServerError("", err, w, r)
			return
		}
	} else {
		ar.ToLogin(w, r)
		return
	}

	referer := r.Referer()
	refererUrl, _ := url.Parse(r.Referer())
	if IsRegisterdPage(refererUrl, ar.router) {
		http.Redirect(w, r, referer, http.StatusFound)
	} else {
		http.Redirect(w, r, "/articles/"+articleIdS, http.StatusFound)
	}
}

func (ar *ArticleResource) React(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	articleIdS := chi.URLParam(r, "id")
	articleId, err := strconv.Atoi(articleIdS)
	if err != nil {
		ar.Error("", errors.New("get article id failed"), w, r, http.StatusBadRequest)
		return
	}

	rootId := r.Form.Get("root")
	react := r.Form.Get("react")
	reactMap := model.GetArticleReactEmojiMap()

	reactType := model.ArticleReact(react)
	if _, ok := reactMap[reactType]; !ok {
		ar.Error("react type error", nil, w, r, http.StatusBadRequest)
		return
	}

	userId := ar.GetLoginedUserId(w, r)
	if userId != 0 {
		err = ar.store.Article.React(articleId, userId, string(reactType))
		if err != nil {
			ar.ServerError("", err, w, r)
			return
		}
	} else {
		ar.ToLogin(w, r)
		return
	}

	referer := r.Referer()
	refererUrl, _ := url.Parse(r.Referer())
	fmt.Println("referer: ", referer)
	fmt.Println("refererUrl: ", refererUrl)
	if IsRegisterdPage(refererUrl, ar.router) {
		if rootId != "" && rootId != "0" && rootId != articleIdS {
			http.Redirect(w, r, fmt.Sprintf("/articles/%s#ar_%s", rootId, articleIdS), http.StatusFound)
		} else {
			http.Redirect(w, r, referer, http.StatusFound)
		}
	} else {
		http.Redirect(w, r, "/articles/"+articleIdS, http.StatusFound)
	}
}
