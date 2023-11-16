package service

import (
	"github.com/microcosm-cc/bluemonday"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
)

type Article struct {
	Store         *store.Store
	SantizePolicy *bluemonday.Policy
}

func (a *Article) Create(title, url, content string, authorId, replyTo int, categoryFrontId string) (int, error) {
	article := &model.Article{
		Title:           title,
		AuthorId:        authorId,
		Link:            url,
		Content:         content,
		ReplyTo:         replyTo,
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

	id, err := a.Store.Article.Create(article.Title, article.Link, article.Content, article.AuthorId, article.ReplyTo, article.CategoryFrontId)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (a *Article) Reply(target int, content string, authorId int) (int, error) {
	article := &model.Article{
		AuthorId: authorId,
		Content:  content,
		ReplyTo:  target,
	}

	article.TrimSpace()
	article.Sanitize(a.SantizePolicy)

	err := article.Valid(true)
	if err != nil {
		// ar.Error(err.Error(), err, w, r, http.StatusBadRequest)
		// return
		return 0, err
	}

	return a.Create("", "", content, authorId, target, "")
}
