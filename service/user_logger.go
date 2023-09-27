package service

import (
	"net/http"

	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
)

type UserLogger struct {
	LoginedUser *model.User
	Store       *store.Store
}

func (ul *UserLogger) SetUser(u *model.User) {
	ul.LoginedUser = u
}

func (ul *UserLogger) Log(action, targetModel string, targetId int, details string, r *http.Request) error {
	var actType model.ActivityType
	var userId int

	if ul.LoginedUser == nil {
		userId = 0
		actType = model.ActivityTypeAnonymous
	} else if ul.LoginedUser.Super || ul.LoginedUser.RoleFrontId == "admin" || ul.LoginedUser.RoleFrontId == "moderator" {
		userId = ul.LoginedUser.Id
		actType = model.ActivityTypeManage
	} else {
		userId = ul.LoginedUser.Id
		actType = model.ActivityTypeUser
	}

	_, err := ul.Store.Activity.Create(userId, string(actType), action, targetModel, targetId, r.RemoteAddr, r.UserAgent(), details)
	if err != nil {
		return err
	}
	return nil
}
