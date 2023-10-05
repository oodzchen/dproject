package service

import (
	"github.com/microcosm-cc/bluemonday"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
	"github.com/oodzchen/dproject/utils"
)

type UserListType string

const (
	UserListAll      UserListType = "all"
	UserListSaved                 = "saved"
	UserListArticle               = "article"
	UserListReply                 = "reply"
	UserListActivity              = "activity"
)

var AuthRequiedUserTabMap = map[UserListType]bool{
	UserListSaved:    true,
	UserListActivity: true,
}

func CheckUserTabAuthRequired(tab UserListType) bool {
	return AuthRequiedUserTabMap[tab]
}

type User struct {
	Store         *store.Store
	SantizePolicy *bluemonday.Policy
}

// const DefaultUserRoleFrontId = "common_user"

func (u *User) Register(email string, password string, name string) (int, error) {
	user := &model.User{
		Email:    email,
		Name:     name,
		Password: password,
	}

	user.TrimSpace()

	if user.Name == "" {
		user.Name = utils.ExtractNameFromEmail(user.Email)
	}

	user.Sanitize(u.SantizePolicy)
	err := user.Valid()
	if err != nil {
		return 0, err
	}

	// log.Printf("user model is %v", user)

	err = user.EncryptPassword()
	if err != nil {
		return 0, err
	}

	// fmt.Printf("Password value: %s\n", user.Password)
	return u.Store.User.Create(user.Email, user.Password, user.Name, string(model.DefaultUserRoleCommon))
}

func (u *User) GetPosts(userId int, listType UserListType) ([]*model.Article, error) {
	switch listType {
	case UserListSaved:
		return u.Store.User.GetSavedPosts(userId)
	default:
		return u.Store.User.GetPosts(userId, string(listType))
	}
}
