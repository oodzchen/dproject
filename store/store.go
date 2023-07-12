package store

import (
	"github.com/oodzchen/dproject/model"
)

type Store struct {
	ArticleStore ArticleStore
}

type ArticleStore interface {
	List() ([]*model.Article, error)
	Create(*model.Article) (int, error)
	Update(*model.Article) (int, error)
	Item(int) (*model.Article, error)
	Delete(int) error
}

func New() *Store {
	//...f
	return &Store{}
}
