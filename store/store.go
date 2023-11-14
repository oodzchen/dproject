package store

import (
	"time"

	"github.com/oodzchen/dproject/model"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	Rdb        *redis.Client
	Article    ArticleStore
	User       UserStore
	Role       RoleStore
	Permission PermissionStore
	Activity   ActivityStore
	Message    MessageStore
}

// type DBStore interface {
// 	NewArticleStore() (any, error)
// 	NewUserStore() (any, error)
// 	NewPermissionStore() (any, error)
// 	NewRoleStore() (any, error)
// 	NewActivity() (any, error)
// 	NewMessage() (any, error)
// }

type ArticleStore interface {
	// pageSize < 0 to list all undeleted data
	List(page, pageSize int, sortType model.ArticleSortType) ([]*model.Article, int, error)
	ListUserState(ids []int, userId int) ([]*model.Article, error)
	ListLatestCount(start, end time.Time) (int, error)
	Create(title, url, content string, authorId, replyTo int) (int, error)
	Update(a *model.Article, fields []string) (int, error)
	Item(id, loginedUserId int) (*model.Article, error)
	Delete(id int) (int, error)
	ItemTree(page, pageSize, ariticleId int, sortType model.ArticleSortType) ([]*model.Article, error)
	ItemTreeUserState(ids []int, userId int) ([]*model.Article, error)
	Count() (int, error)
	CountTotalReply(id int) (int, error)
	VoteCheck(id, userId int) (error, string)
	Vote(id, loginedUserId int, voteType string) error
	Save(id, loginedUserId int) error
	React(id, loginedUserId, reactId int) error
	Subscribe(id, loginedUserId int) error
	CheckSubscribe(id, loginedUserId int) (int, error)
	Notify(senderUserId, sourceArticleId int, content string) error
	GetReactList() ([]*model.ArticleReact, error)
	ReactItem(int) (*model.ArticleReact, error)
}

type UserStore interface {
	List(page, pageSize int, oldest bool, username, roleForntId string) ([]*model.User, int, error)
	Create(email, password, name, roleFrontId string) (int, error)
	CreateWithOAuth(email, name, roleFrontId, authTyp string) (int, error)
	Update(u *model.User, fields []string) (int, error)
	Item(int) (*model.User, error)
	ItemWithEmail(email string) (*model.User, error)
	ItemWithUsername(username string) (*model.User, error)
	ItemWithUsernameEmail(usernameEmail string) (*model.User, error)
	Exists(email, username string) (int, error)
	// Delete(int) error
	// Ban(int) error
	GetPosts(username string, listType string) ([]*model.Article, error)
	GetSavedPosts(username string) ([]*model.Article, error)
	GetSubscribedPosts(username string) ([]*model.Article, error)
	Count() (int, error)
	SetRole(userId int, roleFrontId string) (int, error)
	SetRoleManyWithFrontId([]*model.User) error
	GetPassword(usernameEmail string) (string, error)
	UpdatePassword(email, password string) (int, error)
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
	Create(userId int, actType, action, targetModel string, targetId any, ipAddr, deviceInfo, details string) (int, error)
}

type MessageStore interface {
	List(userId int, status string, page, pageSize int) ([]*model.Message, int, error)
	Create(senderUserId, reciverUserId, sourceArticleId int, content string) (int, error)
	Read(messageId int) error
	ReadMany(messageIds []any) error
	UnreadCount(loginedUserId int) (int, error)
}

// func New(dbStore DBStore, rdb *redis.Client) (*Store, error) {
// 	article, err := dbStore.NewArticleStore()
// 	user, err := dbStore.NewUserStore()
// 	permission, err := dbStore.NewPermissionStore()
// 	role, err := dbStore.NewRoleStore()
// 	activity, err := dbStore.NewActivity()
// 	message, err := dbStore.NewMessage()

// 	if err != nil {
// 		return nil, err
// 	}

// 	articleStore := article.(ArticleStore)
// 	userStore := user.(UserStore)
// 	permissionStore := permission.(PermissionStore)
// 	roleStore := role.(RoleStore)
// 	activityStore := activity.(ActivityStore)
// 	messageStore := message.(MessageStore)

// 	cache := &Cache{rdb}

// 	return &Store{
// 		rdb:        rdb,
// 		cache:      cache,
// 		Article:    proxyArticleStore(articleStore, cache),
// 		User:       userStore,
// 		Role:       roleStore,
// 		Permission: permissionStore,
// 		Activity:   activityStore,
// 		Message:    messageStore,
// 	}, nil
// }

func proxyArticleStore(store ArticleStore) ArticleStore {
	// originalList := store.List
	// store.List = func(page, pageSize, userId int, sortType model.ArticleSortType) ([]*model.Article, int, error) {
	// 	// var list []*model.Article
	// 	// err := rdb.ZRangeByScore(context.Background(), "store_article_list", &redis.ZRangeBy{
	// 	// 	Offset: int64(pageSize) * int64(page - 1),
	// 	// 	Count: int64(pageSize),
	// 	// }).ScanSlice(&list)
	// 	// if err != nil {
	// 	// 	return nil, 0, err
	// 	// }

	// 	list, total, err := originalList(page, pageSize, userId, sortType)
	// 	if err != nil {
	// 		return nil, 0, err
	// 	}

	// 	err = cache.SetList(page, pageSize, userId, sortType, list)
	// 	if err != nil {
	// 		return nil, 0, err
	// 	}

	// 	return list, total, nil
	// }

	return store
}
