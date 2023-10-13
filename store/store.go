package store

import (
	"github.com/oodzchen/dproject/model"
)

type Store struct {
	Article    ArticleStore
	User       UserStore
	Role       RoleStore
	Permission PermissionStore
	Activity   ActivityStore
	Message    MessageStore
}

type DBStore interface {
	NewArticleStore() (any, error)
	NewUserStore() (any, error)
	NewPermissionStore() (any, error)
	NewRoleStore() (any, error)
	NewActivity() (any, error)
	NewMessage() (any, error)
}

type ArticleStore interface {
	// pageSize < 0 to list all undeleted data
	List(page, pageSize, userId int) ([]*model.Article, error)
	Create(title, url, content string, authorId, replyTo int) (int, error)
	Update(a *model.Article, fields []string) (int, error)
	Item(id, loginedUserId int) (*model.Article, error)
	Delete(id int, authorId int) (int, error)
	ItemTree(ariticleId, userId int) ([]*model.Article, error)
	Count() (int, error)
	VoteCheck(id, userId int) (error, string)
	Vote(id, loginedUserId int, voteType string) error
	Save(id, loginedUserId int) error
	React(id, loginedUserId int, reactType string) error
	Subscribe(id, loginedUserId int) error
	Notify(senderUserId, sourceArticleId int, content string) error
}

type UserStore interface {
	List(page, pageSize int, oldest bool, username, roleForntId string) ([]*model.User, int, error)
	Create(email, password, name string, roleFrontId string) (int, error)
	Update(u *model.User, fields []string) (int, error)
	Item(int) (*model.User, error)
	Delete(int) error
	Ban(int) error
	Login(username, pwd string) (int, error)
	GetPosts(userId int, listType string) ([]*model.Article, error)
	GetSavedPosts(int) ([]*model.Article, error)
	GetSubscribedPosts(int) ([]*model.Article, error)
	Count() (int, error)
	SetRole(userId int, roleFrontId string) (int, error)
	SetRoleManyWithFrontId([]*model.User) error
}

type PermissionStore interface {
	List(page, pageSize int, module string) ([]*model.Permission, error)
	Create(module, frontId, name string) (int, error)
	CreateMany(list []*model.Permission) error
	Update(name string) (int, error)
	Item(int) (*model.Permission, error)
	Clear() error
	// Delete(int) error
}

type RoleStore interface {
	List(page, pageSize int) ([]*model.Role, error)
	Create(frontId, name string, permissions []int) (int, error)

	//Create role use permission front id
	CreateWithFrontId(frontId, name string, permissionFrontIds []string) (int, error)

	// Create multi role with permission front id
	CreateManyWithFrontId([]*model.Role) error
	Update(id int, name string, permissions []int) (int, error)

	// Update role use permission front id
	UpdateWithFrontId(roleId int, name string, permissionFrontIds []string) (int, error)
	Item(int) (*model.Role, error)
	Delete(int) error
}

type ActivityStore interface {
	List(userId int, userName, actType, action string, page, pageSize int) ([]*model.Activity, int, error)
	Create(userId int, actType, action, targetModel string, targetId int, ipAddr, deviceInfo, details string) (int, error)
}

type MessageStore interface {
	List(userId int, status int, page, pageSize int) ([]*model.Message, int, error)
	Create(senderUserId, reciverUserId, sourceArticleId int, content string) (int, error)
	Read(messageId int) error
	UnreadCount(loginedUserId int) (int, error)
}

func New(dbStore DBStore) (*Store, error) {
	article, err := dbStore.NewArticleStore()
	user, err := dbStore.NewUserStore()
	permission, err := dbStore.NewPermissionStore()
	role, err := dbStore.NewRoleStore()
	activity, err := dbStore.NewActivity()
	message, err := dbStore.NewMessage()

	if err != nil {
		return nil, err
	}

	articleStore := article.(ArticleStore)
	userStore := user.(UserStore)
	permissionStore := permission.(PermissionStore)
	roleStore := role.(RoleStore)
	activityStore := activity.(ActivityStore)
	messageStore := message.(MessageStore)

	return &Store{
		Article:    articleStore,
		User:       userStore,
		Role:       roleStore,
		Permission: permissionStore,
		Activity:   activityStore,
		Message:    messageStore,
	}, nil
}
