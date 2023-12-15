package web

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

type aList struct {
	Pinned bool
	List   []*model.Article
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
			"article.edit_others",
		}, ar)).Group(func(r chi.Router) {
			r.Get("/edit", ar.FormPage)
			r.With(mdw.UserLogger(
				ar.uLogger, model.AcTypeUser, model.AcActionEditArticle, model.AcModelArticle, mdw.ULogURLArticleId),
			).Post("/edit", ar.Update)
		})

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.edit_mine",
			"article.edit_others",
		}, ar)).Group(func(r chi.Router) {
			r.Get("/block_regions", ar.BlockRegionsPage)
			r.With(mdw.UserLogger(
				ar.uLogger, model.AcTypeManage, model.AcActionBlockRegions, model.AcModelArticle, mdw.ULogURLArticleId),
			).Post("/block_regions", ar.BlockRegions)
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

		r.Get("/history", ar.HistoryPage)

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.edit_others",
		}, ar), mdw.UserLogger(
			ar.uLogger, model.AcTypeManage, model.AcActionToggleHideHistory, model.AcModelArticle, mdw.ULogURLArticleId),
		).Post("/history/{historyId}/toggle_hide", ar.ToggleHideHistory)

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.delete_others",
		}, ar), mdw.UserLogger(
			ar.uLogger, model.AcTypeManage, model.AcActionRecover, model.AcModelArticle, mdw.ULogURLArticleId),
		).Post("/recover", ar.Recover)

		r.Get("/share", ar.Share)

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.edit_others",
		}, ar), mdw.UserLogger(
			ar.uLogger, model.AcTypeManage, model.AcActionLockArticle, model.AcModelArticle, mdw.ULogURLArticleId),
		).Post("/lock", ar.ToggleLock)

		r.With(mdw.AuthCheck(ar.sessStore), mdw.PermitCheck(ar.srv.Permission, []string{
			"article.edit_others",
		}, ar), mdw.UserLogger(
			ar.uLogger, model.AcTypeManage, model.AcActionFadeOutArticle, model.AcModelArticle, mdw.ULogURLArticleId),
		).Post("/fade_out", ar.ToggleFadeOut)
	})

	return rt
}

var oldestStartTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

func filterArray[K any](arr []K, filterFunc func(K) bool) []K {
	var result []K
	for _, value := range arr {
		if filterFunc(value) {
			result = append(result, value)
		}
	}
	return result
}

func (ar *ArticleResource) List(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	paramPage := r.Form.Get("page")
	sort := r.Form.Get("sort")
	categoryFrontId := chi.URLParam(r, "categoryFrontId")

	currUserId := ar.GetLoginedUserId(w, r)

	var category *model.Category
	var err error
	if categoryFrontId != "" {
		category, err = ar.store.Category.Item(categoryFrontId, currUserId)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				ar.Error("", err, w, r, http.StatusNotFound)
				return
			}
			ar.ServerErrorp("", err, w, r)
			return
		}
	}

	var sortType model.ArticleSortType
	defaultSort := model.DefaultArticleListSortType

	if uiSettings, ok := r.Context().Value("ui_settings").(*model.UISettings); ok {
		defaultSort = uiSettings.DefaultArticleSortType
	}

	if model.ValidArticleSort(sort) {
		sortType = model.ArticleSortType(sort)
	} else {
		sortType = defaultSort
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

	startTime := time.Now()

	var wg sync.WaitGroup
	var ch = make(chan any, 3)
	var total int
	var list []*model.Article
	var pinnedList []*model.Article

	wg.Add(3)

	go func() {
		defer wg.Done()
		total, err := ar.store.Article.Count(categoryFrontId)
		if err != nil {
			ch <- err
			return
		}
		ch <- total
		fmt.Printf("get article count duration: %dms\n", time.Since(startTime).Milliseconds())
	}()

	go ar.getArticleList(&wg, page, pageSize, sortType, categoryFrontId, currUserId, startTime, ch, false)

	if page == 1 {
		go ar.getArticleList(&wg, page, pageSize, sortType, categoryFrontId, currUserId, startTime, ch, true)
	} else {
		wg.Done()
	}

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
		case *aList:
			if v.Pinned {
				pinnedList = v.List
			} else {
				list = v.List
			}
		}
	}

	fmt.Printf("get article list total duration: %dms\n", time.Since(startTime).Milliseconds())

	// fmt.Println("pinnedList:", pinnedList)
	if page == 1 {
		list = append(pinnedList, list...)
	}

	requestRegionCode := r.Context().Value("region_country_iso_code")
	for _, item := range list {
		item.CheckShowScore(currUserId)

		if code, ok := requestRegionCode.(string); ok {
			item.UpdateBlockedState(code)
		}
	}

	canEditOthers := ar.CheckPermit(r, "article", "edit_others")
	list = filterArray(list, func(item *model.Article) bool {
		return canEditOthers || !item.Blocked
	})

	// total, err := ar.store.Article.Count()
	// if err != nil {
	// 	ar.Error("", err, w, r, http.StatusInternalServerError)
	// 	return
	// }

	type ListData struct {
		Articles        []*model.Article
		ArticleTotal    int
		CurrPage        int
		PageSize        int
		TotalPage       int
		SortType        model.ArticleSortType
		DefaultSortType model.ArticleSortType
		Category        *model.Category
		SortTabList     []model.ArticleSortType
		SortTabNames    map[model.ArticleSortType]string
	}

	pageData := &model.PageData{
		Data: &ListData{
			list,
			total,
			page,
			pageSize,
			CeilInt(total, pageSize),
			sortType,
			defaultSort,
			category,
			model.GetSortTypeList(false, defaultSort),
			model.GetSortTypeNames(ar.i18nCustom),
		},
	}

	if categoryFrontId != "" && category != nil {
		pageData.Title = category.Name
		pageData.BreadCrumbs = []*model.BreadCrumb{
			{
				Path: fmt.Sprintf("/categories/%s", category.FrontId),
				Name: category.Name,
			},
		}
		pageData.Description = category.Describe
	}

	ar.Render(w, r, "article_list", pageData)
}

func (ar *ArticleResource) getArticleList(
	wg *sync.WaitGroup,
	page,
	pageSize int,
	sortType model.ArticleSortType,
	categoryFrontId string,
	currUserId int,
	startTime time.Time,
	ch chan<- any,
	pinned bool,
) {
	defer wg.Done()
	// list, err := ar.getArticleList(page, pageSize, currUserId, sortType)
	list, _, err := ar.store.Article.List(page, pageSize, sortType, categoryFrontId, pinned, false, false, "")
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

	ch <- &aList{
		Pinned: pinned,
		List:   list,
	}
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

	categoryList, err := ar.store.Category.List(model.CategoryStateAll)
	if err != nil {
		ar.ServerErrorp("", err, w, r)
		return
	}

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
			ar.Error("", err, w, r, http.StatusUnauthorized)
			return
		}

		article, err := ar.store.Article.Item(rId, currUserId)
		if err != nil {
			if errors.Is(err, model.AppErrArticleNotExist) {
				ar.NotFound(w, r)
			} else {
				ar.Error("", err, w, r, http.StatusInternalServerError)
			}
			return
		}

		// fmt.Println("article:", article)
		// fmt.Println("ar.srv.Permission:", ar.srv.Permission)

		if (article.AuthorId != currUserId && !ar.CheckPermit(r, "article", "edit_others")) || !ar.CheckPermit(r, "article", "edit_mine") {
			// http.Redirect(w, r, fmt.Sprintf("/articles/%d", articleId), http.StatusFound)
			ar.Forbidden(nil, w, r)
			return
		}

		// if article.AuthorId != currUserId {
		// 	http.Redirect(w, r, fmt.Sprintf("/articles/%d", rId), http.StatusFound)
		// 	return
		// }

		if article.ReplyToId != 0 {
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
		MaxTitleLen         int
		MaxContentLen       int
		Article             *model.Article
		Categories          []*model.Category
		CurrCategoryFrontId string
	}

	ar.Render(w, r, "create", &model.PageData{
		Title: pageTitle,
		Data: &PageData{
			MaxTitleLen:         model.MAX_ARTICLE_TITLE_LEN,
			MaxContentLen:       model.MAX_ARTICLE_CONTENT_LEN,
			Article:             data,
			Categories:          categoryList,
			CurrCategoryFrontId: r.URL.Query().Get("category"),
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
	categoryFrontId := r.Form.Get("category_front_id")
	paramReplyTo := chi.URLParam(r, "articleId")

	pinned := r.Form.Get("pinned")
	lockedStr := r.Form.Get("locked")
	// fmt.Println("pinned: ", pinned)

	var pinnedExpireAt time.Time
	var locked bool

	if pinned == "1" {
		pinnedExpireAtStr := r.Form.Get("pinned_expire_at")
		fmt.Println("pinned expires at:", pinnedExpireAtStr)
		if pinnedExpireAtStr != "" {
			// time.Parse("", value string)
			pinnedExpireAtStr = strings.Join(strings.Split(pinnedExpireAtStr, "T"), " ") + ":00"

			// fmt.Println("pinned expires at2:", pinnedExpireAtStr)
			pinnedExpireAt, err = time.Parse(time.DateTime, pinnedExpireAtStr)
			if err != nil {
				ar.Error(ar.Local("FormatError", "FieldNames", ar.Local("PinExpireTime")), err, w, r, http.StatusBadRequest)
				return
			}
		} else {
			ar.Error(ar.Local("Required", "FieldNames", ar.Local("PinExpireTime")), errors.New("pinned expires time is required"), w, r, http.StatusBadRequest)
			return
		}
	}

	if lockedStr == "1" {
		locked = true
	}

	var replyToId int
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
			replyToId = num
			isReply = replyToId > 0
		}

		if isReply {
			err = ar.checkLocked(replyToId, r)
			if err != nil {
				ar.Forbidden(err, w, r)
				return
			}
		}
	}

	authorId, err := GetLoginUserId(ar.sessStore, w, r)
	if err != nil {
		sess, err := ar.sessStore.Get(r, "one")
		logSessError("one", errors.WithStack(err))

		var callbackUrl string
		if isReply {
			callbackUrl = fmt.Sprintf("/articles/%d", replyToId)
		} else {
			callbackUrl = "/create"
		}
		sess.Values["login_callback"] = callbackUrl
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// id, err := ar.store.Article.Create(article.Title, article.Content, authorId, replyToId)
	var id int
	if isReply {
		id, err = ar.articleSrv.Reply(replyToId, content, authorId, pinnedExpireAt, locked)
	} else {
		id, err = ar.articleSrv.Create(title, url, content, authorId, 0, categoryFrontId, pinnedExpireAt, locked)
	}
	// id, err := ar.articleSrv.Create(title, content, authorId, replyToId)
	if err != nil {
		if errors.Is(err, model.AppErrArticleValidFailed) {
			ar.Error(err.Error(), err, w, r, http.StatusBadRequest)
		} else {
			ar.Error("", err, w, r, http.StatusInternalServerError)
		}
		return
	}

	ssOne := ar.Session("one", w, r)

	ctx := context.WithValue(r.Context(), "article_id", id)
	*r = *r.WithContext(ctx)

	ssOne.Flash(ar.Local("PublishSuccess"))

	if isReply {
		if ssOne.GetStringValue("prev_url") != "" {
			ar.ToPrevPage(w, r)
		} else {
			http.Redirect(w, r, fmt.Sprintf("/articles/%d", replyToId), http.StatusFound)
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

	// rId, err := strconv.Atoi(r.Form.Get("id"))
	id, err := strconv.Atoi(chi.URLParam(r, "articleId"))
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	err = ar.checkLocked(id, r)
	if err != nil {
		ar.Forbidden(err, w, r)
		return
	}

	replyDepth, err := strconv.Atoi(r.Form.Get("reply_depth"))
	// fmt.Printf("replyDepth: %d\n", replyDepth)
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	pinned := r.Form.Get("pinned")
	lockedStr := r.Form.Get("locked")
	hideChanges := r.Form.Get("hide_changes")
	// fmt.Println("pinned: ", pinned)

	var pinnedExpireAt time.Time
	var locked bool
	var isHideEditHisotry bool

	if pinned == "1" {
		pinnedExpireAtStr := r.Form.Get("pinned_expire_at")
		fmt.Println("pinned expires at:", pinnedExpireAtStr)
		if pinnedExpireAtStr != "" {
			// time.Parse("", value string)
			pinnedExpireAtStr = strings.Join(strings.Split(pinnedExpireAtStr, "T"), " ") + ":00"

			fmt.Println("pinned expires at2:", pinnedExpireAtStr)
			pinnedExpireAt, err = time.Parse(time.DateTime, pinnedExpireAtStr)
			if err != nil {
				ar.Error(ar.Local("FormatError", "FieldNames", ar.Local("PinExpireTime")), err, w, r, http.StatusBadRequest)
				return
			}
		} else {
			ar.Error(ar.Local("Required", "FieldNames", ar.Local("PinExpireTime")), errors.New("pinned expires time is required"), w, r, http.StatusBadRequest)
			return
		}
	}

	if lockedStr == "1" {
		locked = true
	}

	if hideChanges == "1" {
		isHideEditHisotry = true
	}

	isReply := replyDepth > 0
	// fmt.Printf("isReply: %t\n", isReply)

	article := &model.Article{
		Content:    r.Form.Get("content"),
		Id:         id,
		ReplyDepth: replyDepth,
	}
	if !isReply {
		article.Title = r.Form.Get("title")
		article.Link = r.Form.Get("url")
		article.CategoryFrontId = r.Form.Get("category_front_id")
	}

	article.TrimSpace()
	article.Sanitize(ar.sanitizePolicy)

	err = article.Valid(true)
	if err != nil {
		ar.Error(err.Error(), err, w, r, http.StatusBadRequest)
		return
	}

	// updateFields := []string{"Content"}
	// if !isReply {
	// 	updateFields = append(updateFields, "Title", "Link", "CategoryFrontId")
	// }

	// id, err := ar.store.Article.Update(article, updateFields)

	currUserId := ar.GetLoginedUserId(w, r)
	if currUserId == 0 {
		ar.Error("", err, w, r, http.StatusUnauthorized)
		return
	}

	oldArticle, err := ar.store.Article.Item(id, 0)
	if err != nil {
		ar.ServerErrorp("", err, w, r)
		return
	}

	if (oldArticle.AuthorId != currUserId && !ar.CheckPermit(r, "article", "edit_others")) || !ar.CheckPermit(r, "article", "edit_mine") {
		ar.Forbidden(nil, w, r)
		return
	}

	if isReply {
		_, err = ar.store.Article.UpdateReply(id, article.Content, pinnedExpireAt, locked)
	} else {
		_, err = ar.store.Article.UpdateRootArticle(id, article.Title, article.Content, article.Link, article.CategoryFrontId, pinnedExpireAt, locked)
	}

	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	// if !pinnedExpireAt.IsZero() {
	// 	err = ar.store.Article.Pin(id, pinnedExpireAt)
	// } else {
	// 	err = ar.store.Article.Unpin(id)
	// }

	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	go ar.addHistoryLog(article.Id, oldArticle, currUserId, isReply, isHideEditHisotry)

	ssOne := ar.Session("one", w, r)

	ssOne.Flash(ar.Local("PublishSuccess"))

	if isReply && ssOne.GetStringValue("prev_url") != "" {
		ar.ToPrevPage(w, r)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/articles/%d", id), http.StatusFound)
	}
}

func (ar *ArticleResource) addHistoryLog(articleId int, oldArticle *model.Article, currUserId int, isReply bool, isHidden bool) {
	article, err := ar.store.Article.Item(articleId, 0)
	if err != nil {
		fmt.Println("get latest article data when add history error:", err)
		return
	}

	oldArticle.Content = html.UnescapeString(oldArticle.Content)
	article.Content = html.UnescapeString(article.Content)

	// fmt.Printf("old article title, url, content, category_front_id: %s, %s, %s, %s\n", oldArticle.Title, oldArticle.Link, oldArticle.Content, oldArticle.CategoryFrontId)
	// fmt.Printf("new article title, url, content, category_front_id: %s, %s, %s, %s\n", article.Title, article.Link, article.Content, article.CategoryFrontId)

	var contentDelta, titleDelta, urlDelta, categoryFrontDelta string
	if article.Content != oldArticle.Content {
		contentDiffs := ar.dmp.DiffMain(article.Content, oldArticle.Content, false)
		contentDelta = ar.dmp.DiffToDelta(contentDiffs)
	}

	if isReply {
		if contentDelta != "" {
			_, err = ar.store.Article.AddHistory(article.Id, currUserId, article.UpdatedAt, oldArticle.UpdatedAt, "", "", contentDelta, "", isHidden)
		}
	} else {
		if article.Title != oldArticle.Title {
			titleDiffs := ar.dmp.DiffMain(article.Title, oldArticle.Title, false)
			titleDelta = ar.dmp.DiffToDelta(titleDiffs)
		}

		if article.Link != oldArticle.Link {
			urlDiffs := ar.dmp.DiffMain(article.Link, oldArticle.Link, false)
			urlDelta = ar.dmp.DiffToDelta(urlDiffs)
		}

		if article.CategoryFrontId != oldArticle.CategoryFrontId {
			categoryFrontDiffs := ar.dmp.DiffMain(article.CategoryFrontId, oldArticle.CategoryFrontId, false)
			categoryFrontDelta = ar.dmp.DiffToDelta(categoryFrontDiffs)
		}

		if contentDelta != "" || titleDelta != "" || urlDelta != "" || categoryFrontDelta != "" {
			_, err = ar.store.Article.AddHistory(article.Id, currUserId, article.UpdatedAt, oldArticle.UpdatedAt, titleDelta, urlDelta, contentDelta, categoryFrontDelta, isHidden)
		}
	}

	if err != nil {
		fmt.Println("add article history error:", err)
		return
	}
}

func (ar *ArticleResource) Item(w http.ResponseWriter, r *http.Request) {
	ar.handleItem(w, r, ArticlePageDetail)
}

type ArticlePageType string

const (
	ArticlePageDel          ArticlePageType = "del"
	ArticlePageReply                        = "reply"
	ArticlePageDetail                       = "detail"
	ArticlePageBlockRegions                 = "block_regions"
)

// {{- $regions := list "mainland_china" "us" "in" -}}
// {{- $regionsMap := dict "mainland_china" (local "MainlandChina") "us" (local "UnitedStates") "in" (local "India") -}}
type Region struct {
	Name, Value string
	Checked     bool
}

func (ar *ArticleResource) handleItem(w http.ResponseWriter, r *http.Request, pageType ArticlePageType) {
	idParam := chi.URLParam(r, "articleId")
	sortTypeQ := r.URL.Query().Get("sort")
	pageQ := r.URL.Query().Get("page")
	page, _ := strconv.Atoi(pageQ)

	var sortType model.ArticleSortType
	defaultSort := model.DefaultReplyListSortType

	if page < 1 {
		page = 1
	}

	repliesLayout := model.RepliesLayoutTree
	if uiSettings, ok := r.Context().Value("ui_settings").(*model.UISettings); ok {
		// fmt.Println("replies layout:", string(uiSettings.RepliesLayout))
		repliesLayout = uiSettings.RepliesLayout
		defaultSort = uiSettings.DefaultReplySortType
	}

	if model.ValidArticleSort(sortTypeQ) {
		sortType = model.ArticleSortType(sortTypeQ)
	} else {
		sortType = defaultSort
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

	var rootArticle *model.Article
	var articleList []*model.Article
	var pinnedList []*model.Article
	var reactList []*model.ArticleReact
	var reactMap map[string]*model.ArticleReact
	var totalReplyCount int
	var regionList []*Region
	var regionMap map[string]*Region

	if pageType == ArticlePageBlockRegions {
		regionList = []*Region{
			{
				ar.Local("MainlandChina"),
				"mainland_china",
				false,
			},
			{
				ar.Local("UnitedStates"),
				"us",
				false,
			},
			{
				ar.Local("India"),
				"in",
				false,
			},
		}

		regionMap = make(map[string]*Region)

		for _, region := range regionList {
			regionMap[region.Value] = region
		}
	}

	startTime := time.Now()

	var wg sync.WaitGroup
	ch := make(chan any, 5)

	wg.Add(5)

	go func() {
		defer wg.Done()
		item, err := ar.store.Article.Item(articleId, currUserId)
		// fmt.Println("root article:", item)
		if err != nil {
			ch <- err
			return
		}
		ch <- item
	}()

	go func() {
		defer wg.Done()
		totalReplyCount, err = ar.store.Article.CountTotalReply(articleId)
		// fmt.Println("total reply count:", totalReplyCount)
		if err != nil {
			ch <- err
			return
		}
		fmt.Printf("get total count duration: %dms\n", time.Since(startTime).Milliseconds())
		ch <- totalReplyCount
	}()

	go ar.getReplyList(articleId, currUserId, page, DefaultPageSize, sortType, pageType, &wg, ch, startTime, repliesLayout, false)

	if page == 1 {
		go ar.getReplyList(articleId, currUserId, page, DefaultPageSize, sortType, pageType, &wg, ch, startTime, repliesLayout, true)
	} else {
		wg.Done()
	}

	go func() {
		defer wg.Done()
		rList, err := ar.store.Article.GetReactList()
		if err != nil {
			ch <- err
			return
		}

		ch <- rList

	}()

	go func() {
		wg.Wait()
		close(ch)
	}()

	for result := range ch {
		switch v := result.(type) {
		case error:
			if v != nil {
				if errors.Is(v, pgx.ErrNoRows) || errors.Is(v, model.AppErrArticleNotExist) {
					ar.NotFound(w, r)
				} else {
					ar.Error("", v, w, r, http.StatusInternalServerError)
				}
				return
			}
		case *model.Article:
			// fmt.Println("result root article:", v)
			rootArticle = v
		case int:
			totalReplyCount = v
		case *aList:
			if v.Pinned {
				pinnedList = v.List
			} else {
				articleList = v.List
			}
		case []*model.ArticleReact:
			reactList = v
		}
	}

	// fmt.Println("totalReplyCount:", totalReplyCount)
	fmt.Printf("get article item duration: %dms\n", time.Since(startTime).Milliseconds())

	// fmt.Println("root article after channel:", rootArticle)
	// fmt.Println("article list length:", len(articleList))
	// fmt.Println("article count:", totalReplyCount)
	if rootArticle == nil || rootArticle.Id == 0 {
		// ar.Error("the article is gone", err, w, r, http.StatusNotFound)
		ar.NotFound(w, r)
		return
	}

	// if len(articleList) == 0 {
	// 	// http.Redirect(w, r, "/404", http.StatusNotFound)
	// 	ar.Error("", nil, w, r, http.StatusNotFound)
	// 	return
	// }

	requestRegionCode := r.Context().Value("region_country_iso_code")
	if code, ok := requestRegionCode.(string); ok {
		rootArticle.UpdateBlockedState(code)

		for _, item := range articleList {
			item.UpdateBlockedState(code)
		}
	}

	canEditOthers := ar.CheckPermit(r, "article", "edit_others")
	// fmt.Println("can edit others:", canEditOthers)
	articleList = filterArray(articleList, func(item *model.Article) bool {
		return canEditOthers || !item.Blocked
	})

	articleList = append(articleList, rootArticle)

	start1 := time.Now()

	for _, item := range articleList {
		item.CheckShowScore(currUserId)

		if item.ReplyToArticle != nil {
			item.ReplyToArticle.GenSummary(100)
		}

		// if item.Id == articleId {
		// 	rootArticle = item
		// }
	}

	fmt.Printf("format article item duration: %dms\n", time.Since(start1).Milliseconds())

	if rootArticle.Blocked && !ar.CheckPermit(r, "article", "edit_others") {
		ar.Error("", errors.New(fmt.Sprintf("blocke in the contry:%v\n", requestRegionCode)), w, r, http.StatusUnavailableForLegalReasons)
		return
	}

	reactMap = make(map[string]*model.ArticleReact)
	for _, item := range reactList {
		reactMap[item.FrontId] = item
	}

	rootArticle.TotalReplyCount = totalReplyCount

	if pageType == ArticlePageDel {
		// currUserId, err := GetLoginUserId(ar.sessStore, w, r)
		// if err != nil {
		// 	ar.Error("Please login", err, w, r, http.StatusUnauthorized)
		// 	return
		// }

		if (rootArticle.AuthorId != currUserId && !ar.CheckPermit(r, "article", "delete_others")) || !ar.CheckPermit(r, "article", "delete_mine") {
			// http.Redirect(w, r, fmt.Sprintf("/articles/%d", articleId), http.StatusFound)
			ar.Error("", err, w, r, http.StatusForbidden)
			return
		}
	}

	for _, item := range articleList {
		item.FormatDeleted()
	}

	if repliesLayout == model.RepliesLayoutTile {
		rootArticle, _ = genArticleTileList(rootArticle, articleList, page, sortType, totalReplyCount)
	} else {
		rootArticle, _ = genArticleTree(rootArticle, articleList, page, sortType)
	}

	// fmt.Println("reply pinnedList:", pinnedList)
	// fmt.Println("reply list:", rootArticle.Replies.List)
	if rootArticle.Replies != nil {
		rootArticle.Replies.List = append(pinnedList, rootArticle.Replies.List...)
	} else {
		rootArticle.Replies = model.NewArticleList(pinnedList, sortType, page, DefaultPageSize, totalReplyCount)
	}

	// if err != nil {
	// 	fmt.Printf("generate article tree error: %v\n", err)
	// }
	// fmt.Println("replySort: ", replySort)
	// rootArticle = sortArticleTree(rootArticle, sortType)
	// rootArticle = pagingArticleTree(rootArticle, page)

	rootArticle.UpdateDisplayTitle()

	if pageType == ArticlePageBlockRegions {
		for _, region := range regionList {
			for _, blockedRegion := range rootArticle.BlockedRegionsISOCode {
				if region.Value == "mainland_china" && (blockedRegion == "CN" || blockedRegion == "HK") {
					region.Checked = true
				} else if strings.ToUpper(region.Value) == blockedRegion {
					region.Checked = true
				}
			}
		}
	}

	if pageType == ArticlePageDel && rootArticle.AuthorId == currUserId {
		go func() {
			recoverRPC := -rootArticle.VoteDown * model.ReputationChangeValues[model.RPCTypeDownvoted]

			if rootArticle.FadeOut {
				recoverRPC += -model.ReputationChangeValues[model.RPCTypeFadeOut]
			}

			// fmt.Println("recover reputation", recoverRPC)

			err := ar.store.User.AddReputationVal(rootArticle.AuthorName, recoverRPC, "recover reputation on deletion by user", false)
			if err != nil {
				fmt.Println("recover reputation error:", err)
				return
			}
		}()
	}

	// if rootArticle.Deleted {
	// 	w.WriteHeader(http.StatusGone)
	// }

	type itemPageData struct {
		Article *model.Article
		// DelPage  bool
		MaxDepth        int
		PageType        ArticlePageType
		ReactOptions    []*model.ArticleReact
		ReactMap        map[string]*model.ArticleReact
		RegionOptions   []*Region
		RegionMap       map[string]*Region
		DefaultSortType model.ArticleSortType
		SortTabList     []model.ArticleSortType
		SortTabNames    map[model.ArticleSortType]string
	}

	ar.Render(w, r, "article", &model.PageData{
		Title:       rootArticle.DisplayTitle,
		Description: rootArticle.Summary,
		Data: &itemPageData{
			rootArticle,
			// delPage,
			utils.GetReplyDepthSize(),
			pageType,
			reactList,
			reactMap,
			regionList,
			regionMap,
			defaultSort,
			model.GetSortTypeList(true, defaultSort),
			model.GetSortTypeNames(ar.i18nCustom),
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Path: fmt.Sprintf("/categories/%s", rootArticle.Category.FrontId),
				Name: rootArticle.Category.Name,
			},
		},
	})
}

func (ar *ArticleResource) getReplyList(
	articleId,
	currUserId,
	page,
	pageSize int,
	sortType model.ArticleSortType,
	pageType ArticlePageType,
	wg *sync.WaitGroup,
	ch chan<- any,
	startTime time.Time,
	repliesLayout string,
	pinned bool,
) {
	// fmt.Println("get article tree list root id:", articleId)
	// fmt.Println("get article tree list pageType:", string(pageType))
	defer wg.Done()
	var list []*model.Article
	var err error
	if repliesLayout == model.RepliesLayoutTree {
		list, err = ar.store.Article.ReplyTree(page, pageSize, articleId, sortType, pinned)
	} else {
		list, err = ar.store.Article.ReplyList(page, pageSize, articleId, sortType, pinned)
	}

	if err != nil {
		ch <- err
		return
	}
	// fmt.Println("item tree list top id:", list[0].Id)
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

	fmt.Printf("tree list total duration: %dms\n", time.Since(startTime).Milliseconds())
	ch <- &aList{
		Pinned: pinned,
		List:   list,
	}
}

const DefaultReplyPageSize = 50

func genArticleTree(root *model.Article, list []*model.Article, page int, sortType model.ArticleSortType) (*model.Article, error) {
	// fmt.Println("root:", root)
	parentMap := make(map[int]*model.Article)
	nodeMap := make(map[int][]*model.Article)

	for _, item := range list {
		nodeMap[item.ReplyToId] = append(nodeMap[item.ReplyToId], item)
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
				// fmt.Printf("item reply to: %#v \n", item.ReplyToId)
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

func genArticleTileList(root *model.Article, list []*model.Article, page int, sortType model.ArticleSortType, totalReplyCount int) (*model.Article, error) {
	// fmt.Println("root article in gen tile:", root)
	var replies []*model.Article
	// replyToMap := make(map[int]*model.Article)

	for _, item := range list {
		if item.Id == root.Id {
			continue
		}

		replies = append(replies, item)

		// if item, ok := replyToMap[item.ReplyToId]; ok {
		// 	replyToMap[item.ReplyToId] = item
		// }
	}

	// for _, item := range list {
	// 	if child, ok := replyToMap[item.Id]; ok {
	// 		item.GenSummary(100)
	// 		child.ReplyToArticle = item
	// 	}
	// }

	root.Replies = model.NewArticleList(replies, sortType, page, DefaultReplyPageSize, totalReplyCount)
	return root, nil
}

// Remove deleted replies with no replies
func removeEmptyItem(item *model.Article) {
	if item.ReplyToId == 0 {
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

	err = ar.checkLocked(rId, r)
	if err != nil {
		ar.Forbidden(err, w, r)
		return
	}

	currUser := ar.GetLoginedUserData(r)

	article, err := ar.store.Article.Item(rId, currUser.Id)
	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	if (article.AuthorId != currUser.Id && !ar.CheckPermit(r, "article", "delete_others")) || !ar.CheckPermit(r, "article", "delete_mine") {
		// http.Redirect(w, r, fmt.Sprintf("/articles/%d", rId), http.StatusFound)
		ar.Error("", err, w, r, http.StatusForbidden)
		return
	}

	rootArticleId, err := ar.store.Article.Delete(rId)
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	// if article.VoteDown > 0 || article.FadeOut {
	// 	go func() {
	// 		recoverRPCVal := -article.VoteDown*model.ReputationChangeValues[model.RPCTypeDownvoted] - model.ReputationChangeValues[model.RPCTypeFadeOut]
	// 		// fmt.Println("recover reputation:", recoverRPCVal)
	// 		err := ar.store.User.AddReputationVal(article.AuthorName, recoverRPCVal, "recover_on_delete", false)
	// 		if err != nil {
	// 			fmt.Println("add reputation error", err)
	// 			return
	// 		}
	// 	}()
	// }

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

	err = ar.checkLocked(articleId, r)
	if err != nil {
		ar.Forbidden(err, w, r)
		return
	}

	rootId := r.Form.Get("root")

	// userId := ar.GetLoginedUserId(w, r)
	user := ar.GetLoginedUserData(r)
	if user.Id != 0 {
		code, err := ar.store.Article.ToggleVote(articleId, user.Id, voteType)
		if err != nil {
			ar.ServerErrorp("", err, w, r)
			return
		}

		// go func() {
		// 	article, err := ar.store.Article.Item(articleId, 0)
		// 	if err != nil {
		// 		fmt.Println("update reputation error", err)
		// 		return
		// 	}

		// 	if article.AuthorId == user.Id {
		// 		return
		// 	}

		// 	err = ar.store.User.UpdateReputation(article.AuthorName)
		// 	if err != nil {
		// 		fmt.Println("update reputation error:", err)
		// 	}
		// }()

		go func() {
			article, err := ar.store.Article.Item(articleId, 0)
			if err != nil {
				fmt.Println("add reputation error", err)
				return
			}

			if article.AuthorId == user.Id {
				return
			}

			changeType := model.RPCTypeUpvoted
			if voteType == "down" {
				changeType = model.RPCTypeDownvoted
			}
			var isRevert bool
			if code == -1 {
				isRevert = true
			}

			if code == 2 {
				var prevChangeType model.ReputationChangeType
				if changeType == model.RPCTypeUpvoted {
					prevChangeType = model.RPCTypeDownvoted
				} else {
					prevChangeType = model.RPCTypeUpvoted
				}

				err = ar.store.User.AddReputation(article.AuthorName, prevChangeType, true)
				if err != nil {
					fmt.Println("add reputation error", err)
					return
				}
			}

			err = ar.store.User.AddReputation(article.AuthorName, changeType, isRevert)
			if err != nil {
				fmt.Println("add reputation error", err)
				return
			}
		}()
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

	err = ar.checkLocked(articleId, r)
	if err != nil {
		ar.Forbidden(err, w, r)
		return
	}

	rootId := r.Form.Get("root")
	userId := ar.GetLoginedUserId(w, r)
	if userId != 0 {
		err = ar.store.Article.ToggleSave(articleId, userId)
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

// Check the react front id count reputation
func reactCountRPC(frontId string) bool {
	return frontId == "thanks" || frontId == "happy"
}

// React front id to change type
func reactToChangeType(frontId string) model.ReputationChangeType {
	switch frontId {
	case "thanks":
		return model.RPCTypeThanked
	case "happy":
		return model.RPCTypeLaughed
	default:
		return ""
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

	err = ar.checkLocked(articleId, r)
	if err != nil {
		ar.Forbidden(err, w, r)
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
	reactItem, err := ar.store.Article.ReactItem(reactId)
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
		code, prevFrontId, err := ar.store.Article.ToggleReact(articleId, userId, reactId)
		if err != nil {
			ar.ServerErrorp("", err, w, r)
			return
		}

		// fmt.Println("prev react:", prevFrontId)
		// fmt.Println("curr react:", reactItem.FrontId)

		// go func() {
		// 	article, err := ar.store.Article.Item(articleId, 0)
		// 	if err != nil {
		// 		fmt.Println("update reputation error", err)
		// 		return
		// 	}

		// 	if article.AuthorId == userId {
		// 		return
		// 	}

		// 	err = ar.store.User.UpdateReputation(article.AuthorName)
		// 	if err != nil {
		// 		fmt.Println("update reputation error:", err)
		// 	}
		// }()

		go func() {
			article, err := ar.store.Article.Item(articleId, 0)
			if err != nil {
				fmt.Println("add reputation error", err)
				return
			}

			if article.AuthorId == userId {
				return
			}

			var changeType model.ReputationChangeType
			var isRevert bool

			switch code {
			case -1:
				if !reactCountRPC(reactItem.FrontId) {
					return
				}
				changeType = reactToChangeType(reactItem.FrontId)
				isRevert = true
			case 1:
				if !reactCountRPC(reactItem.FrontId) {
					return
				}
				changeType = reactToChangeType(reactItem.FrontId)
				isRevert = false
			case 2:
				if !reactCountRPC(prevFrontId) && !reactCountRPC(reactItem.FrontId) {
					return
				} else if reactCountRPC(prevFrontId) && !reactCountRPC(reactItem.FrontId) {
					changeType = reactToChangeType(prevFrontId)
					isRevert = true
				} else if !reactCountRPC(prevFrontId) && reactCountRPC(reactItem.FrontId) {
					changeType = reactToChangeType(reactItem.FrontId)
					isRevert = false
				} else {
					changeType = reactToChangeType(prevFrontId)
					isRevert = true
					if string(changeType) != "" {
						err = ar.store.User.AddReputation(article.AuthorName, changeType, isRevert)
						if err != nil {
							fmt.Println("add reputation error", err)
							return
						}
					}

					changeType = reactToChangeType(reactItem.FrontId)
					isRevert = false
				}
			}

			// fmt.Println("react changeType:", changeType)

			if string(changeType) != "" {
				err = ar.store.User.AddReputation(article.AuthorName, changeType, isRevert)
				if err != nil {
					fmt.Println("add reputation error", err)
					return
				}
			}
		}()
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

	err = ar.checkLocked(articleId, r)
	if err != nil {
		ar.Forbidden(err, w, r)
		return
	}

	rootId := r.Form.Get("root")
	userId := ar.GetLoginedUserId(w, r)
	if userId != 0 {
		err = ar.store.Article.ToggleSubscribe(articleId, userId)
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

func (ar *ArticleResource) HistoryPage(w http.ResponseWriter, r *http.Request) {
	articleId, err := strconv.Atoi(chi.URLParam(r, "articleId"))
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	currUserId := ar.GetLoginedUserId(w, r)
	var wg sync.WaitGroup
	var res = make(chan any, 3)

	wg.Add(3)
	go func() {
		defer wg.Done()
		list, err := ar.store.Article.ListHistory(articleId)
		// fmt.Println("log list:", list)
		if err != nil {
			res <- err
			return
		}

		res <- list
	}()

	go func() {
		defer wg.Done()
		item, err := ar.store.Article.Item(articleId, currUserId)
		if err != nil {
			res <- err
			return
		}

		res <- item
	}()

	go func() {
		defer wg.Done()
		list, err := ar.store.Category.List(model.CategoryStateAll)
		if err != nil {
			res <- err
			return
		}

		res <- list
	}()

	go func() {
		wg.Wait()
		close(res)
	}()

	var logList []*model.ArticleLog
	var article *model.Article
	var categoryMap = make(map[string]string)

	for v := range res {
		switch val := v.(type) {
		case error:
			if err != nil {
				ar.ServerErrorp("", val, w, r)
				return

			}

		case []*model.ArticleLog:
			logList = val
		case *model.Article:
			article = val
		case []*model.Category:
			for _, category := range val {
				categoryMap[category.FrontId] = category.Name
			}
		}
	}

	// for _, item := range logList {
	// 	item.PrimaryArticle.Content = html.UnescapeString(item.PrimaryArticle.Content)
	// }

	article.Content = html.UnescapeString(article.Content)

	logList, err = model.GenArticleDiffsFromDelta(ar.dmp, article, logList)
	if err != nil {
		ar.ServerErrorp("", err, w, r)
		return
	}

	article.Content = html.EscapeString(article.Content)
	article.FormatNullValues()
	article.UpdateDisplayTitle()

	type pageData struct {
		Article     *model.Article
		List        []*model.ArticleLog
		CategoryMap map[string]string
	}

	ar.Render(w, r, "article_history", &model.PageData{
		Title: ar.Local("EditHistoryTitle", "Title", article.DisplayTitle),
		Data: &pageData{
			Article:     article,
			List:        logList,
			CategoryMap: categoryMap,
		},
	})

}

func (ar *ArticleResource) checkLocked(articleId int, r *http.Request) error {
	locked, err := ar.store.Article.CheckLocked(articleId)
	if err != nil {
		return err
	}

	// fmt.Println("locked:", locked)
	// fmt.Println("lock permission:", ar.CheckPermit(r, "article", "lock"))

	if !ar.CheckPermit(r, "article", "lock") && locked {
		return errors.New("article is locked, you have no permission to update it")
	}

	return nil
}

func (ar *ArticleResource) ToggleHideHistory(w http.ResponseWriter, r *http.Request) {
	articleId, err := strconv.Atoi(chi.URLParam(r, "articleId"))
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	historyId, err := strconv.Atoi(chi.URLParam(r, "historyId"))
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}
	toHide := r.Form.Get("to_hide")

	var setHidden bool

	if toHide == "1" {
		setHidden = true
	}

	err = ar.store.Article.ToggleHideHistory(historyId, setHidden)
	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/articles/%d/history", articleId), http.StatusFound)
}

func (ar *ArticleResource) Recover(w http.ResponseWriter, r *http.Request) {
	articleId, err := strconv.Atoi(chi.URLParam(r, "articleId"))
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	err = ar.store.Article.Recover(articleId)
	if err != nil {
		ar.Error("", err, w, r, http.StatusInternalServerError)
		return
	}

	ar.ToRefererUrl(w, r)
}

func (ar *ArticleResource) Share(w http.ResponseWriter, r *http.Request) {
	articleId, err := strconv.Atoi(chi.URLParam(r, "articleId"))
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	article, err := ar.store.Article.Item(articleId, 0)
	if err != nil {
		ar.ServerErrorp("", err, w, r)
		return
	}

	type pageData struct {
		Article    *model.Article
		RefererURL string
	}

	ar.Render(w, r, "article_share", &model.PageData{
		Title: ar.Local("Share") + " - " + article.DisplayTitle,
		Data: &pageData{
			Article:    article,
			RefererURL: r.Referer(),
		},
		BreadCrumbs: []*model.BreadCrumb{
			{
				Name: ar.Local("Share"),
			},
		},
	})
}

func (ar *ArticleResource) BlockRegionsPage(w http.ResponseWriter, r *http.Request) {
	ar.handleItem(w, r, ArticlePageBlockRegions)
}

func (ar *ArticleResource) BlockRegions(w http.ResponseWriter, r *http.Request) {
	articleId, err := strconv.Atoi(chi.URLParam(r, "articleId"))
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	// fmt.Println("article id:", articleId)

	regions := r.PostForm["blocked_regions"]
	// fmt.Println("blocked regions:", regions)
	var blockedRegions []string
	for _, region := range regions {
		if region == "mainland_china" {
			blockedRegions = append(blockedRegions, "CN")
			blockedRegions = append(blockedRegions, "HK")
		} else {
			blockedRegions = append(blockedRegions, strings.ToUpper(region))
		}
	}

	err = ar.store.Article.SetBlockRegions(articleId, blockedRegions)
	if err != nil {
		ar.ServerErrorp("", err, w, r)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/articles/%d", articleId), http.StatusFound)
}

func (ar *ArticleResource) ToggleLock(w http.ResponseWriter, r *http.Request) {
	articleId, err := strconv.Atoi(chi.URLParam(r, "articleId"))
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	rootId, _ := strconv.Atoi(r.Form.Get("root"))

	err = ar.store.Article.ToggleLock(articleId)
	if err != nil {
		ar.ServerErrorp("", err, w, r)
		return
	}

	ar.toReplyAnchor(rootId, articleId, w, r)
}

func (ar *ArticleResource) toReplyAnchor(rootId, replyId int, w http.ResponseWriter, r *http.Request) {
	referer := r.Referer()
	refererUrl, _ := url.Parse(r.Referer())
	if IsRegisterdPage(refererUrl, ar.router) && rootId != 0 && rootId != replyId {
		http.Redirect(w, r, fmt.Sprintf("%s#ar_%d", referer, replyId), http.StatusFound)
	} else {
		http.Redirect(w, r, referer, http.StatusFound)
	}
}

func (ar *ArticleResource) ToggleFadeOut(w http.ResponseWriter, r *http.Request) {
	articleId, err := strconv.Atoi(chi.URLParam(r, "articleId"))
	if err != nil {
		ar.Error("", err, w, r, http.StatusBadRequest)
		return
	}

	rootId, _ := strconv.Atoi(r.Form.Get("root"))

	code, err := ar.store.Article.ToggleFadeOut(articleId)
	if err != nil {
		ar.ServerErrorp("", err, w, r)
		return
	}

	// userId := ar.GetLoginedUserId(w, r)
	// go func() {
	// 	article, err := ar.store.Article.Item(articleId, 0)
	// 	if err != nil {
	// 		fmt.Println("add reputation error", err)
	// 		return
	// 	}

	// 	if article.AuthorId == userId {
	// 		return
	// 	}

	// 	err = ar.store.User.UpdateReputation(article.AuthorName)
	// 	if err != nil {
	// 		fmt.Println("update reputation error:", err)
	// 	}
	// }()

	go func() {
		article, err := ar.store.Article.Item(articleId, 0)
		if err != nil {
			fmt.Println("add reputation error", err)
			return
		}

		var isRevert bool
		if code == -1 {
			isRevert = true
		}

		err = ar.store.User.AddReputation(article.AuthorName, model.RPCTypeFadeOut, isRevert)
		if err != nil {
			fmt.Println("add reputation error", err)
			return
		}
	}()

	ar.toReplyAnchor(rootId, articleId, w, r)
}
