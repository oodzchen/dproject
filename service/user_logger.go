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

func (ul *UserLogger) Log(u *model.User, actType model.ActivityType, action model.AcAction, targetModel model.AcModel, handler UserLogHandler, r *http.Request) error {
	var userId int

	if u == nil {
		userId = 1
		actType = model.ActivityTypeAnonymous
	} else {
		userId = u.Id
	}

	var lackedField string
	if action == "" {
		lackedField = "action"
	}

	if len(lackedField) > 0 {
		return model.ActivityValidErr(fmt.Sprintf("%s is required", lackedField))
	}

	logData := handler(r)

	// fmt.Println("userId :", userId)
	// fmt.Printf("logger data: %#v\n", logData)

	_, err := ul.Store.Activity.Create(userId, string(actType), string(action), string(targetModel), logData.TargetId, logData.IPAddr, logData.DeviceInfo, logData.Details)
	if err != nil {
		return err
	}

	return nil
}
