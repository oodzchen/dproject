package web

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/feeds"
	"github.com/oodzchen/dproject/config"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/utils"
)

type RSSResource struct {
	*Renderer
	articleResource *ArticleResource
}

func NewRSSResource(renderer *Renderer, ar *ArticleResource) *RSSResource {
	return &RSSResource{
		renderer,
		ar,
	}
}

func (rr *RSSResource) Routes() http.Handler {
	rt := chi.NewRouter()

	rt.Get("/", rr.Atom)
	rt.Get("/{rssSort}", rr.Atom)

	return rt
}

func (rr *RSSResource) Atom(w http.ResponseWriter, r *http.Request) {
	var wg sync.WaitGroup
	ch := make(chan any)

	var sortType model.ArticleSortType
	defaultSort := model.DefaultArticleListSortType

	sort := chi.URLParam(r, "rssSort")
	if model.ValidArticleSort(sort) {
		sortType = model.ArticleSortType(sort)
	} else {
		sortType = defaultSort
	}

	wg.Add(1)
	go rr.articleResource.getArticleList(&wg, 1, DefaultPageSize, sortType, "", 0, time.Now(), ch, false)

	go func() {
		wg.Wait()
		close(ch)
	}()

	var list []*model.Article
	for v := range ch {
		switch val := v.(type) {
		case error:
			rr.ServerErrorp("", val, w, r)
			return
		case *aList:
			list = val.List
		}
	}

	serverUrl := config.Config.GetServerURL()
	feed := &feeds.Feed{
		Title:   rr.Local("BrandName") + " - " + model.GetSortTypeNames(rr.i18nCustom)[sortType],
		Link:    &feeds.Link{Href: fmt.Sprintf("%s/articles?sort=%s", serverUrl, sortType)},
		Created: time.Now(),
	}

	for _, item := range list {
		item.GenSummary(100)

		var sourceHTML string
		if item.Link != "" {
			sourceHTML = "<p>" + rr.Local("Source") + ": " + fmt.Sprintf("<a href=\"%s\">%s</a></p>", item.Link, item.Link)
		}

		feed.Items = append(feed.Items, &feeds.Item{
			Id:          fmt.Sprintf("article:%d", item.Id),
			Title:       item.Title,
			Link:        &feeds.Link{Href: fmt.Sprintf("%s/articles/%d", serverUrl, item.Id)},
			Source:      &feeds.Link{Href: item.Link},
			Description: item.Summary,
			Author:      &feeds.Author{Name: item.AuthorName},
			Created:     item.CreatedAt,
			Updated:     item.UpdatedAt,
			Content:     sourceHTML + utils.NewLine2BR(utils.ReplaceLink(item.Content)),
		})
	}

	atom, err := feed.ToAtom()
	if err != nil {
		rr.ServerErrorp("", err, w, r)
		return
	}

	// fmt.Println("atom:", atom)

	w.Header().Set("Content-Type", "application/xml;charset=utf-8")

	fmt.Fprint(w, atom)
}
