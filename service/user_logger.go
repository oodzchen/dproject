package service

import (
	"fmt"
	"net/http"

	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
)

type UserLogger struct {
	Store *store.Store
}

type UserLogData struct {
	TargetId                    int
	IPAddr, DeviceInfo, Details string
}

type UserLogHandler func(r *http.Request) *UserLogData

func (ul *UserLogger) Log(u *model.User, action, targetModel string, handler UserLogHandler, r *http.Request) error {
	var actType model.ActivityType
	var userId int

	if u == nil {
		userId = 0
		actType = model.ActivityTypeAnonymous
	} else if u.Super || u.RoleFrontId == "admin" || u.RoleFrontId == "moderator" {
		userId = u.Id
		actType = model.ActivityTypeManage
	} else {
		userId = u.Id
		actType = model.ActivityTypeUser
	}

	var lackedField string
	if actType == "" {
		lackedField = "action type"
	}

	if action == "" {
		lackedField = "action"
	}

	if len(lackedField) > 0 {
		return model.ActivityValidErr(fmt.Sprintf("%s is required", lackedField))
	}

	logData := handler(r)

	fmt.Println("logger data: ", logData)

	_, err := ul.Store.Activity.Create(userId, string(actType), action, targetModel, logData.TargetId, logData.IPAddr, logData.DeviceInfo, logData.Details)
	if err != nil {
		return err
	}
	return nil
}
