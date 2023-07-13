package store

import (
	"github.com/oodzchen/dproject/model"
)

type Store struct {
	Article ArticleStore
}

type DBStore interface {
	NewArticle() (any, error)
}

type ArticleStore interface {
	List() ([]*model.Article, error)
	Create(*model.Article) (int, error)
	Update(*model.Article) (int, error)
	Item(int) (*model.Article, error)
	Delete(int) error
}

func New(dbStore DBStore) (*Store, error) {
	//...f
	article, err := dbStore.NewArticle()
	if err != nil {
		return nil, err
	}

	articleStore := article.(ArticleStore)

	return &Store{
		Article: articleStore,
	}, nil
}
