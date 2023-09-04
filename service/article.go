package service

import (
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
)

type Article struct {
	Store *store.Store
}

func (a *Article) Create(title, content string, authorId, replyTo int) (int, error) {
	article := &model.Article{
		Title:    title,
		AuthorId: authorId,
		Content:  content,
		ReplyTo:  replyTo,
	}

	article.Sanitize()

	err := article.Valid(false)
	if err != nil {
		// ar.Error(err.Error(), err, w, r, http.StatusBadRequest)
		// return
		return 0, err
	}

	id, err := a.Store.Article.Create(article.Title, article.Content, authorId, replyTo)
	if err != nil {
		return 0, err
	}
	err = a.Store.Article.Vote(id, authorId, "up")
	return id, nil
}

func (a *Article) Reply(target int, content string, authorId int) (int, error) {
	return a.Create("", content, authorId, target)
}
