package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
)

type Article struct {
	Store         *store.Store
	SantizePolicy *bluemonday.Policy
	WG            *sync.WaitGroup
}

func (a *Article) Create(title, url, content string, authorId, replyToId int, categoryFrontId string, pinnedExpireAt time.Time, locked bool) (int, error) {
	article := &model.Article{
		Title:           title,
		AuthorId:        authorId,
		Link:            url,
		Content:         content,
		ReplyToId:       replyToId,
		CategoryFrontId: categoryFrontId,
	}

	article.TrimSpace()
	article.Sanitize(a.SantizePolicy)

	err := article.Valid(false)
	if err != nil {
		// ar.Error(err.Error(), err, w, r, http.StatusBadRequest)
		// return
		return 0, err
	}

	id, err := a.Store.Article.Create(article.Title, article.Link, article.Content, article.AuthorId, article.ReplyToId, article.CategoryFrontId, pinnedExpireAt, locked)
	if err != nil {
		return 0, err
	}

	err = a.Store.Article.ToggleSubscribe(id, authorId)
	if err != nil {
		return 0, err
	}

	go func() {
		if a.WG != nil {
			defer a.WG.Done()
		}

		// fmt.Println("store:", a.Store, a.Store.Category)
		// fmt.Println("args:", categoryFrontId, authorId, id)
		err = a.Store.Category.Notify(categoryFrontId, authorId, id)
		if err != nil {
			// ar.ServerErrorp("", err, w, r)
			fmt.Println("notify to category subscribers error: ", err)
			return
		}
	}()

	return id, nil
}

func (a *Article) Reply(target int, content string, authorId int, pinnedExpireAt time.Time, locked bool) (int, error) {
	article := &model.Article{
		AuthorId:  authorId,
		Content:   content,
		ReplyToId: target,
	}

	article.TrimSpace()
	article.Sanitize(a.SantizePolicy)

	err := article.Valid(true)
	if err != nil {
		return 0, err
	}

	id, err := a.Store.Article.Create("", "", article.Content, authorId, target, "", pinnedExpireAt, locked)
	if err != nil {
		return 0, err
	}

	count, err := a.Store.Article.CheckSubscribe(id, authorId)
	if err != nil {
		fmt.Printf("check subscribe error: %v\n", err)
		return 0, err
	}

	// fmt.Println("check subscribe count: ", count)
	if count == 0 {
		err = a.Store.Article.ToggleSubscribe(id, authorId)
		if err != nil {
			return 0, err
		}
	}

	go func() {
		if a.WG != nil {
			defer a.WG.Done()
		}

		err = a.Store.Article.Notify(authorId, target, id)
		if err != nil {
			// ar.ServerErrorp("", err, w, r)
			fmt.Println("notify to article subscribers error: ", err)
			return
		}
	}()

	return id, nil
}
