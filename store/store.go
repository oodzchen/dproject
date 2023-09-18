package store

import (
	"github.com/oodzchen/dproject/model"
)

type Store struct {
	Article    ArticleStore
	User       UserStore
	Role       RoleStore
	Permission PermissionStore
}

type DBStore interface {
	NewArticleStore() (any, error)
	NewUserStore() (any, error)
	NewPermissionStore() (any, error)
	NewRoleStore() (any, error)
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
	React(id, loginedUserId int, reactType string) error
}

type UserStore interface {
	List(page, pageSize int, oldest bool) ([]*model.User, error)
	Create(email, password, name string) (int, error)
	Update(u *model.User, fields []string) (int, error)
	Item(int) (*model.User, error)
	Delete(int) error
	Ban(int) error
	Login(email string, pwd string) (int, error)
	GetPosts(userId int, listType string) ([]*model.Article, error)
	GetSavedPosts(int) ([]*model.Article, error)
	Count() (int, error)
}

type PermissionStore interface {
	List(page, pageSize int, module string) ([]*model.Permission, error)
	Create(module, frontId, name string) (int, error)
	Update(name string) (int, error)
	Item(int) (*model.Permission, error)
	// Delete(int) error
}

type RoleStore interface {
	List(page, pageSize int) ([]*model.Role, error)
	Create(frontId, name string, permissions []int) (int, error)
	Update(id int, name string, permissions []int) (int, error)
	Item(int) (*model.Role, error)
	Delete(int) error
}

func New(dbStore DBStore) (*Store, error) {
	article, err := dbStore.NewArticleStore()
	user, err := dbStore.NewUserStore()
	permission, err := dbStore.NewPermissionStore()
	role, err := dbStore.NewRoleStore()

	if err != nil {
		return nil, err
	}

	articleStore := article.(ArticleStore)
	userStore := user.(UserStore)
	permissionStore := permission.(PermissionStore)
	roleStore := role.(RoleStore)

	return &Store{
		Article:    articleStore,
		User:       userStore,
		Role:       roleStore,
		Permission: permissionStore,
	}, nil
}
