package service

import (
	"errors"

	"github.com/microcosm-cc/bluemonday"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
)

type UserListType string

const (
	UserListAll        UserListType = "all"
	UserListSaved                   = "saved"
	UserListArticle                 = "article"
	UserListReply                   = "reply"
	UserListActivity                = "activity"
	UserListSubscribed              = "subscribed"
)

var AuthRequiedUserTabMap = map[UserListType]bool{
	UserListSaved:      true,
	UserListSubscribed: true,
	UserListActivity:   true,
}

func CheckUserTabAuthRequired(tab UserListType) bool {
	return AuthRequiedUserTabMap[tab]
}

type User struct {
	Store         *store.Store
	SantizePolicy *bluemonday.Policy
}

func (u *User) Register(email string, password string, name string) (int, error) {
	if len(password) == 0 {
		return 0, errors.New("lack of password")
	}

	user := &model.User{
		Email: email,
		Name:  name,
	}
	err := user.Valid(false)
	if err != nil {
		return 0, err
	}

	return u.Store.User.Create(email, password, name, string(model.DefaultUserRoleCommon))
}

func (u *User) GetPosts(username string, listType UserListType) ([]*model.Article, error) {
	// fmt.Println("user tab:", listType)
	switch listType {
	case UserListSaved:
		return u.Store.User.GetSavedPosts(username)
	case UserListSubscribed:
		return u.Store.User.GetSubscribedPosts(username)
	default:
		return u.Store.User.GetPosts(username, string(listType))
	}
}
