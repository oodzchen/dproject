package store

import (
	"github.com/oodzchen/dproject/model"
)

type Store struct {
	Article ArticleStore
	User    UserStore
}

type DBStore interface {
	NewArticle() (any, error)
	NewUser() (any, error)
}

type ArticleStore interface {
	List(page, pageSize int) ([]*model.Article, error)
	Create(*model.Article) (int, error)
	Update(*model.Article) (int, error)
	Item(int) (*model.Article, error)
	Delete(id int, authorId int) (int, error)
	ItemTree(int) ([]*model.Article, error)
	Count() (int, error)
}

type UserStore interface {
	List(page, pageSize int) ([]*model.User, error)
	Create(*model.User) (int, error)
	Update(u *model.User, fields []string) (int, error)
	Item(int) (*model.User, error)
	Delete(int) error
	Ban(int) error
	Login(email string, pwd string) (int, error)
	GetPosts(int) ([]*model.Article, error)
	Count() (int, error)
}

func New(dbStore DBStore) (*Store, error) {
	//...f
	article, err := dbStore.NewArticle()
	user, err := dbStore.NewUser()
	if err != nil {
		return nil, err
	}

	articleStore := article.(ArticleStore)
	userStore := user.(UserStore)

	return &Store{
		Article: articleStore,
		User:    userStore,
	}, nil
}
