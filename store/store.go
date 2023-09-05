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
	// pageSize < 0 to list all undeleted data
	List(page, pageSize, userId int) ([]*model.Article, error)
	Create(title, content string, authorId, replyTo int) (int, error)
	Update(a *model.Article, fields []string) (int, error)
	Item(id, loginedUserId int) (*model.Article, error)
	Delete(id int, authorId int) (int, error)
	ItemTree(ariticleId, userId int) ([]*model.Article, error)
	Count() (int, error)
	VoteCheck(id, userId int) (error, string)
	Vote(id, loginedUserId int, voteType string) error
	Save(id, loginedUserId int) error
}

type UserStore interface {
	List(page, pageSize int, oldest bool) ([]*model.User, error)
	Create(email, password, name string) (int, error)
	Update(u *model.User, fields []string) (int, error)
	Item(int) (*model.User, error)
	Delete(int) error
	Ban(int) error
	Login(email string, pwd string) (int, error)
	GetPosts(int) ([]*model.Article, error)
	GetSavedPosts(int) ([]*model.Article, error)
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
