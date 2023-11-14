package web

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5"
	mdw "github.com/oodzchen/dproject/middleware"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/service"
	"github.com/oodzchen/dproject/utils"
	"github.com/pkg/errors"
)

type ArticleResource struct {
	*Renderer
	articleSrv *service.Article
}

func NewArticleResource(renderer *Renderer) *ArticleResource {
	return &ArticleResource{
		renderer,
		renderer.srv.Article,
	}
}

func (ar *ArticleResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Get("/", ar.List)
	rt.With(
		mdw.AuthCheck(ar.sessStore),
		mdw.PermitCheck(ar.srv.Permission, []string{
			"article.create",
		}, ar),
		mdw.UserLogger(ar.uLogger, model.AcTypeUser, model.AcActionCreateArticle, model.AcModelArticle, mdw.ULogNewArticleId),
		httprate.Limit(
			6,
			1*time.Minute,
			httprate.WithKeyByIP(),
			httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
				ar.Error("", nil, w, r, http.StatusTooManyRequests)
			}),
		),
	).Post("/", ar.Submit)

	rt.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
		"article.create",
	}, ar),
	).Get("/new", ar.FormPage)

	rt.Route("/{articleId}", func(r chi.Router) {
		r.Get("/", ar.Item)

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.edit_mine",
			// "article.edit_others",
		}, ar)).Group(func(r chi.Router) {
			r.Get("/edit", ar.FormPage)
			r.With(mdw.UserLogger(
				ar.uLogger, model.AcTypeUser, model.AcActionEditArticle, model.AcModelArticle, mdw.ULogURLArticleId),
			).Post("/edit", ar.Update)
		})

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.delete_mine",
			"article.delete_others",
		}, ar)).Group(func(r chi.Router) {
			r.Get("/delete", ar.DeletePage)
			r.With(mdw.UserLogger(
				ar.uLogger, model.AcTypeUser, model.AcActionDeleteArticle, model.AcModelArticle, mdw.ULogURLArticleId, func(uLogData *service.UserLogData, w http.ResponseWriter, r *http.Request) error {
					// fmt.Println("uLogData: ", uLogData)
					var currUserId int
					if currUser, ok := r.Context().Value("user_data").(*model.User); ok {
						// fmt.Println("curr user id: ", currUser.Id)
						currUserId = currUser.Id
					}

					var deleteArticleAuthorId int
					if v, ok := ar.Session("one", w, r).GetValue("deleted_article_author_id").(int); ok {
						// fmt.Println("deleted article author id: ", v)
						deleteArticleAuthorId = v
						ar.Session("one", w, r).SetValue("deleted_article_author_id", "")
					}

					if currUserId != 0 && deleteArticleAuthorId != 0 && currUserId != deleteArticleAuthorId {
						uLogData.ActionType = model.AcTypeManage
					}

					return nil
				}),
			).Post("/delete", ar.Delete)
		})

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.reply",
		}, ar)).Group(func(r chi.Router) {
			r.Get("/reply", ar.ReplyPage)
			r.With(
				mdw.UserLogger(ar.uLogger, model.AcTypeUser, model.AcActionReplyArticle, model.AcModelArticle, mdw.ULogURLArticleId),
			).Post("/reply", ar.SubmitReply)
		})

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.vote_up",
			"article.vote_down",
		}, ar), mdw.UserLogger(
			ar.uLogger, model.AcTypeUser, model.AcActionVoteArticle, model.AcModelArticle, mdw.ULogURLArticleId),
		).Post("/vote", ar.Vote)

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.save",
		}, ar), mdw.UserLogger(
			ar.uLogger, model.AcTypeUser, model.AcActionSaveArticle, model.AcModelArticle, mdw.ULogURLArticleId),
		).Post("/save", ar.Save)

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.react",
		}, ar), mdw.UserLogger(
			ar.uLogger, model.AcTypeUser, model.AcActionReactArticle, model.AcModelArticle, mdw.ULogURLArticleId),
		).Post("/react", ar.React)

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.subscribe",
		}, ar), mdw.UserLogger(
			ar.uLogger, model.AcTypeUser, model.AcActionSubscribeArticle, model.AcModelArticle, mdw.ULogURLArticleId),
		).Post("/subscribe", ar.Subscribe)
	})

	return rt
}

var oldestStartTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

func (ar *ArticleResource) List(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	paramPage := r.Form.Get("page")
	sort := r.Form.Get("sort")

	var sortType model.ArticleSortType
	if model.ValidArticleSort(sort) {
		sortType = model.ArticleSortType(sort)
	} else {
		sortType = model.ListSortBest
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

	startTime := time.Now()

	var wg sync.WaitGroup
	var ch = make(chan any, 2)
	var total int
	var list []*model.Article

	wg.Add(2)

	go func() {
		defer wg.Done()
		total, err := ar.store.Article.Count()
		if err != nil {
			ch <- err
			return
		}
		ch <- total
		fmt.Printf("get article count duration: %dms\n", time.Since(startTime).Milliseconds())
	}()

	go func() {
		defer wg.Done()
		// list, err := ar.getArticleList(page, pageSize, currUserId, sortType)
		list, _, err := ar.store.Article.List(page, pageSize, sortType)
		if err != nil {
			ch <- err
			return
		}
		fmt.Printf("get article list duration: %dms\n", time.Since(startTime).Milliseconds())

		var ids []int
		listMap := make(map[int]*model.Article)
		for _, item := range list {
			ids = append(ids, item.Id)
			listMap[item.Id] = item
		}

		userStateList, err := ar.store.Article.ListUserState(ids, currUserId)
		if err != nil {
			ch <- err
			return
		}
		fmt.Printf("get user state article list duration: %dms\n", time.Since(startTime).Milliseconds())

		for _, stateItem := range userStateList {
			if item, ok := listMap[stateItem.Id]; ok {
				item.CurrUserState = stateItem.CurrUserState
			}
		}

		ch <- list
	}()

	go func() {
		wg.Wait()
		close(ch)
	}()

	for res := range ch {
		switch v := res.(type) {
		case error:
			if v != nil {
				if errors.Is(v, pgx.ErrNoRows) {
					ar.Error("", nil, w, r, http.StatusNotFound)
				} else {
					ar.Error("", v, w, r, http.StatusInternalServerError)
				}
				return
			}
		case int:
			total = v
		case []*model.Article:
			list = v
		}
	}

	fmt.Printf("get article list total duration: %dms\n", time.Since(startTime).Milliseconds())

	for _, item := range list {
		// fmt.Println("item.VoteScore: ", item.VoteScore)
		// item.FormatTimeStr()
		item.CalcScore()
		item.FormatNullValues()
		item.UpdateDisplayTitle()
		item.GenSummary(200)
		item.CheckShowScore(currUserId)
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

	pageData := &model.PageData{
		Data: &ListData{
			list,
			total,
			page,
			pageSize,
			CeilInt(total, pageSize),
			sortType,
		},
	}

	ar.Render(w, r, "article_list", pageData)
}

// func (ar *ArticleResource) getArticleList(page, pageSize, userId int, sortType model.ArticleSortType) ([]*model.Article, error) {
// 	currTime := time.Now()

// 	wholeList, _, err := ar.store.Article.List(0, -1, userId, currTime.Add(-24*time.Hour), currTime)
// 	// fmt.Println("article total:", total)

// 	if err != nil {
// 		return nil, err
// 	}
// 	wholeArticleList := model.NewArticleList(wholeList, sortType, page, pageSize)

// 	// fmt.Println("whole total:", wholeArticleList.TotalPage)

// 	var list []*model.Article
// 	if page > wholeArticleList.TotalPage {
// 		list, _, err = ar.store.Article.List(page, pageSize, userId, oldestStartTime, currTime)
// 		if err != nil {
// 			return nil, err
// 		}
// 	} else {
// 		for _, item := range wholeList {
// 			item.CalcWeight()
// 		}

// 		// fmt.Println("sortType: ", sortType)

// 		// wholeArticleList := model.NewArticleList(wholeList, sortType, page, pageSize)
// 		// sort.Sort(wholeArticleList)
// 		wholeArticleList.Sort(sortType)

// 		list = wholeArticleList.PagingList(page, pageSize)

// 		if page == wholeArticleList.TotalPage && len(list) < pageSize {
// 			addList, _, err := ar.store.Article.List(1, pageSize-len(list), userId, oldestStartTime, currTime.Add(-24*time.Hour))
// 			// fmt.Println("article add total:", addTotal)
// 			if err != nil {
// 				return nil, err
// 			}

// 			list = append(list, addList...)
// 		}
// 	}

// 	return list, nil

// }

func (ar *ArticleResource) FormPage(w http.ResponseWriter, r *http.Request) {
	if !IsLogin(ar.sessStore, w, r) {
		// http.Redirect(w, r, "/login?target="+r.URL.Path, http.StatusFound)
		sess := ar.Session("one", w, r)
		sess.SetValue("target_url", "/articles/new")
		//
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	id := chi.URLParam(r, "articleId")
	var pageTitle string
	var data *model.Article

	var moduleTitle = ar.Local("AddContent")
	if id == "" {
		pageTitle = ar.i18nCustom.LocalTpl("AddNew")
		data = &model.Article{}
	} else {
		moduleTitle = ar.Local("EditContent")
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

		if article.ReplyTo != 0 {
			article.GenSummary(100)
			pageTitle = fmt.Sprintf("%s - %s", ar.i18nCustom.LocalTpl("BtnEdit"), article.Summary)
		} else {
			pageTitle = fmt.Sprintf("%s - %s", ar.i18nCustom.LocalTpl("BtnEdit"), article.Title)
		}

		article.UpdateDisplayTitle()
		data = article
	}

	ar.SavePrevPage(w, r)

	type PageData struct {
		MaxTitleLen   int
		MaxContentLen int
		Article       *model.Article
	}

	ar.Render(w, r, "create", &model.PageData{
		Title: pageTitle,
		Data: &PageData{
			MaxTitleLen:   model.MAX_ARTICLE_TITLE_LEN,
			MaxContentLen: model.MAX_ARTICLE_CONTENT_LEN,
			Article:       data,
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Name: moduleTitle,
			},
		},
	})
}

func (ar *ArticleResource) handleSubmit(w http.ResponseWriter, r *http.Request, isReply bool) {
	err := r.ParseForm()
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	title := r.Form.Get("title")
	url := r.Form.Get("url")
	content := r.Form.Get("content")
	paramReplyTo := chi.URLParam(r, "articleId")

	var replyTo int
	if isReply {
		if paramReplyTo == "" {
			ar.Error("", err, w, r, http.StatusBadRequest)
			return
		} else {
			num, err := strconv.Atoi(paramReplyTo)
			if err != nil {
				ar.Error("", err, w, r, http.StatusBadRequest)
				return
			}
			replyTo = num
			isReply = replyTo > 0
		}
	}

	authorId, err := GetLoginUserId(ar.sessStore, w, r)
	if err != nil {
		sess, err := ar.sessStore.Get(r, "one")
		logSessError("one", errors.WithStack(err))

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
		id, err = ar.articleSrv.Create(title, url, content, authorId, 0)
	}
	// id, err := ar.articleSrv.Create(title, content, authorId, replyTo)
	if err != nil {
		if errors.Is(err, model.AppErrArticleValidFailed) {
			ar.Error(err.Error(), err, w, r, http.StatusBadRequest)
		} else {
			ar.Error("", err, w, r, http.StatusInternalServerError)
		}
		return
	}

	if isReply {
		count, err := ar.store.Article.CheckSubscribe(id, authorId)
		if err != nil {
			fmt.Printf("check subscribe error: %v\n", err)
		}

		// fmt.Println("check subscribe count: ", count)
		if count == 0 {
			err = ar.store.Article.Subscribe(id, authorId)
		}
	} else {
		err = ar.store.Article.Subscribe(id, authorId)

	}

	if err != nil {
		ar.ServerErrorp("", err, w, r)
		return
	}

	if isReply {
		go func() {
			err = ar.store.Article.Notify(authorId, replyTo, fmt.Sprintf("new reply id: %d", id))
			if err != nil {
				// ar.ServerErrorp("", err, w, r)
				fmt.Println("notify to subscribers error: ", err)
				return
			}
		}()
	}

	ssOne := ar.Session("one", w, r)

	ctx := context.WithValue(r.Context(), "article_id", id)
	*r = *r.WithContext(ctx)

	ssOne.Flash(ar.Local("PublishSuccess"))

	if isReply {
		if ssOne.GetStringValue("prev_url") != "" {
			ar.ToPrevPage(w, r)
		} else {
			http.Redirect(w, r, fmt.Sprintf("/articles/%d", replyTo), http.StatusFound)
		}
	} else {
		http.Redirect(w, r, fmt.Sprintf("/articles/%d", id), http.StatusFound)
	}
}

func (ar *ArticleResource) Submit(w http.ResponseWriter, r *http.Request) {
	ar.handleSubmit(w, r, false)
}

func (ar *ArticleResource) SubmitReply(w http.ResponseWriter, r *http.Request) {
	ar.handleSubmit(w, r, true)
}

func (ar *ArticleResource) Update(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
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
		article.Link = r.Form.Get("url")
	}

	article.TrimSpace()
	article.Sanitize(ar.sanitizePolicy)

	err = article.Valid(true)
	if err != nil {
		ar.Error(err.Error(), err, w, r, http.StatusBadRequest)
		return
	}

	updateFields := []string{"Content"}
	if !isReply {
		updateFields = append(updateFields, "Title", "Link")
	}

	id, err := ar.store.Article.Update(article, updateFields)

	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	ssOne := ar.Session("one", w, r)

	ssOne.Flash(ar.Local("PublishSuccess"))

	if isReply && ssOne.GetStringValue("prev_url") != "" {
		ar.ToPrevPage(w, r)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/articles/%d", id), http.StatusFound)
	}
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
	idParam := chi.URLParam(r, "articleId")
	sortTypeQ := r.URL.Query().Get("sort")
	pageQ := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageQ)

	var sortType model.ArticleSortType
	if page < 1 {
		page = 1
	}

	if model.ValidArticleSort(sortTypeQ) {
		sortType = model.ArticleSortType(sortTypeQ)
	} else {
		sortType = model.ReplySortBest
	}

	// fmt.Println("sort type", sortType)
	// fmt.Printf("idParam: %v\n", idParam)

	articleId, err := strconv.Atoi(idParam)
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	currUserId := ar.GetLoginedUserId(w, r)

	// fmt.Println("curr user id:", currUserId)

	if pageType != ArticlePageDetail && currUserId == 0 {
		ar.ToLogin(w, r)
		return
	}

	var articleTreeList []*model.Article
	var totalReplyCount int

	startTime := time.Now()

	var wg sync.WaitGroup
	ch := make(chan any, 2)

	wg.Add(2)

	go func() {
		defer wg.Done()
		totalReplyCount, err = ar.store.Article.CountTotalReply(articleId)
		if err != nil {
			ch <- err
			return
		}
		fmt.Printf("get total count duration: %dms\n", time.Since(startTime).Milliseconds())
		ch <- totalReplyCount
	}()

	go ar.getArticleTreeList(articleId, currUserId, page, DefaultPageSize, sortType, pageType, &wg, ch, startTime)

	go func() {
		wg.Wait()
		close(ch)
	}()

	for result := range ch {
		switch v := result.(type) {
		case error:
			if v != nil {
				if errors.Is(v, pgx.ErrNoRows) {
					// http.Redirect(w, r, "/404", http.StatusNotFound)
					ar.Error("", nil, w, r, http.StatusNotFound)
				} else {
					ar.Error("", v, w, r, http.StatusInternalServerError)
				}
				return
			}
		case int:
			totalReplyCount = v
		case []*model.Article:
			articleTreeList = v
		}
	}

	// fmt.Println("totalReplyCount:", totalReplyCount)
	fmt.Printf("get article item duration: %dms\n", time.Since(startTime).Milliseconds())

	if len(articleTreeList) == 0 {
		// http.Redirect(w, r, "/404", http.StatusNotFound)
		ar.Error("", nil, w, r, http.StatusNotFound)
		return
	}

	start1 := time.Now()
	var rootArticle *model.Article
	for _, item := range articleTreeList {
		item.FormatNullValues()
		item.FormatReactCounts()
		item.CalcScore()
		// item.CalcWeight()
		item.CheckShowScore(currUserId)

		if item.Id == articleId {
			rootArticle = item
		}
	}
	fmt.Printf("format article item duration: %dms\n", time.Since(start1).Milliseconds())

	if rootArticle == nil || rootArticle.Id == 0 {
		// ar.Error("the article is gone", err, w, r, http.StatusNotFound)
		ar.NotFound(w, r)
		return
	}

	rootArticle.TotalReplyCount = totalReplyCount

	if pageType == ArticlePageDel {
		// currUserId, err := GetLoginUserId(ar.sessStore, w, r)
		// if err != nil {
		// 	ar.Error("Please login", err, w, r, http.StatusUnauthorized)
		// 	return
		// }

		if (rootArticle.AuthorId != currUserId && !ar.srv.Permission.Permit("article", "delete_others")) || !ar.srv.Permission.Permit("article", "delete_mine") {
			// http.Redirect(w, r, fmt.Sprintf("/articles/%d", articleId), http.StatusFound)
			ar.Error("", err, w, r, http.StatusForbidden)
			return
		}
	}

	for _, item := range articleTreeList {
		item.FormatDeleted()
	}

	rootArticle, _ = genArticleTree(rootArticle, articleTreeList, page, sortType)
	// if err != nil {
	// 	fmt.Printf("generate article tree error: %v\n", err)
	// }
	// fmt.Println("replySort: ", replySort)
	// rootArticle = sortArticleTree(rootArticle, sortType)
	// rootArticle = pagingArticleTree(rootArticle, page)

	rootArticle.UpdateDisplayTitle()

	// if rootArticle.Deleted {
	// 	w.WriteHeader(http.StatusGone)
	// }

	reactList, err := ar.store.Article.GetReactList()
	if err != nil {
		ar.ServerErrorp("", err, w, r)
		return
	}

	reactMap := make(map[string]*model.ArticleReact)

	for _, item := range reactList {
		reactMap[item.FrontId] = item
	}

	type itemPageData struct {
		Article *model.Article
		// DelPage  bool
		MaxDepth     int
		PageType     ArticlePageType
		ReactOptions []*model.ArticleReact
		ReactMap     map[string]*model.ArticleReact
	}

	ar.Render(w, r, "article", &model.PageData{Title: rootArticle.DisplayTitle, Data: &itemPageData{
		rootArticle,
		// delPage,
		utils.GetReplyDepthSize(),
		pageType,
		reactList,
		reactMap,
	}})
}

func (ar *ArticleResource) getArticleTreeList(articleId, currUserId, page, pageSize int, sortType model.ArticleSortType, pageType ArticlePageType, wg *sync.WaitGroup, ch chan<- any, startTime time.Time) {
	// fmt.Println("get article tree list root id:", articleId)
	// fmt.Println("get article tree list pageType:", string(pageType))
	defer wg.Done()
	var list []*model.Article
	var err error
	if pageType == ArticlePageDetail {
		list, err = ar.store.Article.ItemTree(page, pageSize, articleId, sortType)
		if err != nil {
			ch <- err
			return
		}
		fmt.Println("item tree list top id:", list[0].Id)
		fmt.Printf("item tree duration: %dms\n", time.Since(startTime).Milliseconds())

		var ids []int
		listMap := make(map[int]*model.Article)
		for _, item := range list {
			ids = append(ids, item.Id)
			listMap[item.Id] = item
		}

		listUserState, err := ar.store.Article.ItemTreeUserState(ids, currUserId)
		if err != nil {
			ch <- err
			return
		}
		fmt.Printf("item tree user state duration: %dms\n", time.Since(startTime).Milliseconds())

		for _, stateItem := range listUserState {
			if article, ok := listMap[stateItem.Id]; ok {
				article.CurrUserState = stateItem.CurrUserState
			}
		}
	} else {
		singleArticle, err := ar.store.Article.Item(articleId, currUserId)
		if err != nil {
			ch <- err
			return
		}
		list = []*model.Article{singleArticle}
	}

	fmt.Printf("tree list total duration: %dms\n", time.Since(startTime).Milliseconds())
	ch <- list
}

const DefaultReplyPageSize = 50

func genArticleTree(root *model.Article, list []*model.Article, page int, sortType model.ArticleSortType) (*model.Article, error) {
	parentMap := make(map[int]*model.Article)
	nodeMap := make(map[int][]*model.Article)

	for _, item := range list {
		nodeMap[item.ReplyTo] = append(nodeMap[item.ReplyTo], item)
	}

	for _, item := range list {
		if _, ok := nodeMap[item.Id]; ok {
			parentMap[item.Id] = item
		}
	}

	for parentId, replies := range nodeMap {
		if parent, ok := parentMap[parentId]; ok {
			// fmt.Printf("parent id: %#v \n", parent.Id)
			for _, item := range replies {
				// fmt.Printf("item reply to: %#v \n", item.ReplyTo)
				item.TmpParent = parent
			}
		}
	}

	if replies, ok := nodeMap[root.Id]; ok {
		root.Replies = model.NewArticleList(replies, sortType, page, DefaultReplyPageSize, root.ChildrenCount)
	} else {
		if len(list) > 0 {
			return root, errors.New("no reply to the root in the list")
		}
	}

	for _, item := range list {
		if replies, ok := nodeMap[item.Id]; ok && item.Id != root.Id {
			item.Replies = model.NewArticleList(replies, sortType, page, DefaultReplyPageSize, item.ChildrenCount)
		}
	}

	for _, item := range list {
		if item.Id != root.Id && item.Deleted {
			// fmt.Printf("item replies len: %#v \n", item.Replies.Len())
			// fmt.Printf("item replies is nil : %#v \n", item.Replies == nil)
			removeEmptyItem(item)
		}
	}

	for _, item := range list {
		item.TmpParent = nil
	}

	return root, nil
}

// Remove deleted replies with no replies
func removeEmptyItem(item *model.Article) {
	if item.ReplyTo == 0 {
		return
	}

	if item.Deleted && item.TmpParent != nil {
		if item.Replies == nil || (item.Replies != nil && item.Replies.Len() == 0) {
			item.TmpParent.Replies.Remove(item.Id)
			removeEmptyItem(item.TmpParent)
		}
	}
}

// func sortArticleTree(root *model.Article, sortType model.ArticleSortType) *model.Article {
// 	if root.Replies != nil && root.Replies.Len() > 0 {
// 		root.Replies.Sort(sortType)
// 		for idx, item := range root.Replies.List {
// 			root.Replies.List[idx] = sortArticleTree(item, sortType)
// 		}
// 	}
// 	return root
// }

// func pagingArticleTree(root *model.Article, page int) *model.Article {
// 	if root.Replies != nil && root.Replies.Len() > 0 {
// 		if page < 1 {
// 			page = 1
// 		}
// 		if page > root.Replies.TotalPage {
// 			page = root.Replies.TotalPage
// 		}

// 		root.Replies.List = root.Replies.PagingList(page, root.Replies.PageSize)
// 		for idx, item := range root.Replies.List {
// 			root.Replies.List[idx] = pagingArticleTree(item, 1)
// 		}
// 	}
// 	return root
// }

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
		// ar.Error("", err, w, r, http.StatusUnauthorized)
		ar.ToLogin(w, r)
		return
	}

	article, err := ar.store.Article.Item(rId, currUserId)
	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	if (article.AuthorId != currUserId && !ar.srv.Permission.Permit("article", "delete_others")) || !ar.srv.Permission.Permit("article", "delete_mine") {
		// http.Redirect(w, r, fmt.Sprintf("/articles/%d", rId), http.StatusFound)
		ar.Error("", err, w, r, http.StatusForbidden)
		return
	}

	rootArticleId, err := ar.store.Article.Delete(rId)
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	ar.Session("one", w, r).Flash(ar.Local("DeleteSuccess"))
	ar.Session("one", w, r).SetValue("deleted_article_author_id", article.AuthorId)

	http.Redirect(w, r, fmt.Sprintf("/articles/%d", rootArticleId), http.StatusFound)
}

func (ar *ArticleResource) DeletePage(w http.ResponseWriter, r *http.Request) {
	ar.handleItem(w, r, ArticlePageDel)
}

func (ar *ArticleResource) ReplyPage(w http.ResponseWriter, r *http.Request) {
	ar.SavePrevPage(w, r)
	ar.handleItem(w, r, ArticlePageReply)
	// http.Redirect(w, r, fmt.Sprintf("/articles/%s", chi.URLParam(r, "articleId")), http.StatusFound)
}

func (ar *ArticleResource) Vote(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	voteType := r.Form.Get("type")
	if !model.IsValidVoteType(model.VoteType(voteType)) {
		ar.Error("", errors.New("vote type not valid: "+voteType), w, r, http.StatusBadRequest)
		return
	}

	articleIdS := chi.URLParam(r, "articleId")
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
			ar.ServerErrorp("", err, w, r)
			return
		}
	} else {
		ar.ToLogin(w, r)
		return
	}

	referer := r.Referer()
	refererUrl, _ := url.Parse(r.Referer())
	if IsRegisterdPage(refererUrl, ar.router) && rootId != "" && rootId != "0" && rootId != articleIdS {
		http.Redirect(w, r, fmt.Sprintf("%s#ar_%s", referer, articleIdS), http.StatusFound)
	} else {
		http.Redirect(w, r, referer, http.StatusFound)
	}
}

func (ar *ArticleResource) Save(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	articleIdS := chi.URLParam(r, "articleId")
	articleId, err := strconv.Atoi(articleIdS)
	if err != nil {
		ar.Error("", errors.New("get article id failed"), w, r, http.StatusBadRequest)
		return
	}

	rootId := r.Form.Get("root")
	userId := ar.GetLoginedUserId(w, r)
	if userId != 0 {
		err = ar.store.Article.Save(articleId, userId)
		if err != nil {
			ar.ServerErrorp("", err, w, r)
			return
		}
	} else {
		ar.ToLogin(w, r)
		return
	}

	referer := r.Referer()
	refererUrl, _ := url.Parse(r.Referer())
	if IsRegisterdPage(refererUrl, ar.router) && rootId != "" && rootId != "0" && rootId != articleIdS {
		http.Redirect(w, r, fmt.Sprintf("%s#ar_%s", referer, articleIdS), http.StatusFound)
	} else {
		http.Redirect(w, r, referer, http.StatusFound)
	}
}

func (ar *ArticleResource) React(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	articleIdS := chi.URLParam(r, "articleId")
	articleId, err := strconv.Atoi(articleIdS)
	if err != nil {
		ar.Error("", errors.New("get article id failed"), w, r, http.StatusBadRequest)
		return
	}

	rootId := r.Form.Get("root")
	reactId, err := strconv.Atoi(r.Form.Get("react_id"))
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}
	// reactMap := model.GetArticleReactEmojiMap()

	// reactType := model.ArticleReact(react)
	// if _, ok := reactMap[reactType]; !ok {
	// 	ar.Error("react type error", nil, w, r, http.StatusBadRequest)
	// 	return
	// }
	_, err = ar.store.Article.ReactItem(reactId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ar.Error("react type error", nil, w, r, http.StatusBadRequest)
			return
		}

		ar.ServerErrorp("", err, w, r)
		return
	}

	userId := ar.GetLoginedUserId(w, r)
	if userId != 0 {
		err = ar.store.Article.React(articleId, userId, reactId)
		if err != nil {
			ar.ServerErrorp("", err, w, r)
			return
		}
	} else {
		ar.ToLogin(w, r)
		return
	}

	referer := r.Referer()
	refererUrl, _ := url.Parse(r.Referer())
	// fmt.Println("referer: ", referer)
	// fmt.Println("refererUrl: ", refererUrl)
	if IsRegisterdPage(refererUrl, ar.router) && rootId != "" && rootId != "0" && rootId != articleIdS {
		http.Redirect(w, r, fmt.Sprintf("%s#ar_%s", referer, articleIdS), http.StatusFound)
	} else {
		http.Redirect(w, r, referer, http.StatusFound)
	}
}

func (ar *ArticleResource) Subscribe(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	articleIdS := chi.URLParam(r, "articleId")
	articleId, err := strconv.Atoi(articleIdS)
	if err != nil {
		ar.Error("", errors.New("get article id failed"), w, r, http.StatusBadRequest)
		return
	}

	rootId := r.Form.Get("root")
	userId := ar.GetLoginedUserId(w, r)
	if userId != 0 {
		err = ar.store.Article.Subscribe(articleId, userId)
		if err != nil {
			ar.ServerErrorp("", err, w, r)
			return
		}
	} else {
		ar.ToLogin(w, r)
		return
	}

	referer := r.Referer()
	refererUrl, _ := url.Parse(r.Referer())
	if IsRegisterdPage(refererUrl, ar.router) && rootId != "" && rootId != "0" && rootId != articleIdS {
		http.Redirect(w, r, fmt.Sprintf("%s#ar_%s", referer, articleIdS), http.StatusFound)
	} else {
		http.Redirect(w, r, referer, http.StatusFound)
	}
}
