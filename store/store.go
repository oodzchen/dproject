package store

import (
	"time"

	"github.com/oodzchen/dproject/model"
)

type Store struct {
	Article    ArticleStore
	User       UserStore
	Role       RoleStore
	Permission PermissionStore
	Activity   ActivityStore
	Message    MessageStore
	Category   CategoryStore
}

func New(
	article ArticleStore,
	user UserStore,
	role RoleStore,
	permission PermissionStore,
	activity ActivityStore,
	message MessageStore,
	category CategoryStore,
) *Store {
	return &Store{
		article,
		user,
		role,
		permission,
		activity,
		message,
		category,
	}
}

type ArticleStore interface {
	// pageSize < 0 to list all undeleted data
	List(page,
		pageSize int,
		sortType model.ArticleSortType,
		categoryFrontId string,
		pinned, deleted, includeReplies bool,
		keywords string,
	) ([]*model.Article, int, error)
	ListUserState(ids []int, userId int) ([]*model.Article, error)
	ListLatestCount(start, end time.Time) (int, error)
	Create(title, url, content string, authorId, replyToId int, categoryFrontId string, pinnedExpireAt time.Time, locked bool) (int, error)
	// Update(a *model.Article, fields []string) (int, error)
	UpdateRootArticle(id int, title, content, link, categoryFrontId string, pinnedExpireAt time.Time, locked bool) (int, error)
	UpdateReply(id int, content string, pinnedExpireAt time.Time, locked bool) (int, error)
	Item(id, loginedUserId int) (*model.Article, error)
	Delete(id int) (int, error)
	ReplyTree(page, pageSize, ariticleId int, sortType model.ArticleSortType, pinned bool) ([]*model.Article, error)
	ReplyList(page, pageSize, ariticleId int, sortType model.ArticleSortType, pinned bool) ([]*model.Article, error)
	ItemTreeUserState(ids []int, userId int) ([]*model.Article, error)
	Count(categoryFrontId string, includePinned bool) (int, error)
	CountTotalReply(id int) (int, error)
	VoteCheck(id, userId int) (error, string)
	// Return int value, 0 for error, -1 for canceled, 1 for added, 2 for updated
	ToggleVote(id, loginedUserId int, voteType string) (int, error)
	ToggleSave(id, loginedUserId int) error
	// Return int value, 0 for error, -1 for canceled, 1 for added, 2 for updated
	// String value for previous react id
	ToggleReact(id, loginedUserId, reactId int) (int, string, error)
	ToggleSubscribe(id, loginedUserId int) error
	CheckSubscribe(id, loginedUserId int) (int, error)
	Notify(senderUserId, sourceArticleId, contentArticleId int) error
	GetReactList() ([]*model.ArticleReact, error)
	ReactItem(int) (*model.ArticleReact, error)
	Tag(id int, tagFrontId string) error
	AddHistory(
		articleId,
		operatorId int,
		curr,
		prev time.Time,
		titleDelta,
		urlDelta,
		contentDelta,
		categoryFrontDelta string,
		isHidden bool,
	) (int, error)
	ListHistory(articleId int) ([]*model.ArticleLog, error)
	ToggleHideHistory(historyId int, isHidden bool) error
	ToggleLock(articleId int) error
	CheckLocked(id int) (bool, error)
	Pin(articleId int, expireAt time.Time) error
	Unpin(articleId int) error
	// DeletedList() ([]*model.Article, error)
	Recover(articleId int) error
	SetBlockRegions(articleId int, regions []string) error
	// Return int value, 0 for error, -1 for canceled, 1 for added
	ToggleFadeOut(articleId int) (int, error)
}

type UserStore interface {
	List(page, pageSize int, oldest bool, username, roleForntId string, authType model.AuthType) ([]*model.User, int, error)
	Create(email, password, name, roleFrontId string) (int, error)
	CreateWithOAuth(email, name, roleFrontId, authTyp string) (int, error)
	// Update(u *model.User, fields []string) (int, error)
	UpdateIntroduction(username, introduction string) error
	Item(int) (*model.User, error)
	ItemWithEmail(email string) (*model.User, error)
	ItemWithUsername(username string) (*model.User, error)
	ItemWithUsernameEmail(usernameEmail string) (*model.User, error)
	Exists(email, username string) (int, error)
	// Delete(int) error
	Ban(username string, bannedDays int) (int, error)
	Unban(username string) (int, error)
	GetPosts(username string, listType string) ([]*model.Article, error)
	GetSavedPosts(username string) ([]*model.Article, error)
	GetSubscribedPosts(username string) ([]*model.Article, error)
	Count() (int, error)
	SetRole(userId int, roleFrontId string) (int, error)
	SetRoleManyWithFrontId([]*model.User) error
	GetPassword(usernameEmail string) (string, error)
	UpdatePassword(email, password string) (int, error)
	AddReputation(username string, changeType model.ReputationChangeType, isRevert bool) error
	AddReputationVal(username string, value int, comment string, isRevert bool) error
	// UpdateReputation(username string) error
	GetVotedPosts(username string, voteType model.VoteType) ([]*model.Article, error)
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

type CategoryStore interface {
	List(state model.CategoryState) ([]*model.Category, error)
	Create(frontId, name, describe string, authorId int) (int, error)
	Update(frontId, name, describe string) (int, error)
	Item(frontId string, loginedUserId int) (*model.Category, error)
	Approval(frontId string, pass bool, comment string) error
	Delete(frontId string) error
	Subscribe(frontId string, loginedUserId int) error
	Notify(frontId string, senderUserId, contentArticleId int) error
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
	// Create(senderUserId, reciverUserId, sourceArticleId, contentArticleId int) (int, error)
	Read(messageId int) error
	ReadMany(messageIds []any) error
	UnreadCount(loginedUserId int) (int, error)
	ReadAll(userId int) error
}
