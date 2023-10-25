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
	ActionType                  model.AcType
	Action                      model.AcAction
	TargetModel                 model.AcModel
	TargetId                    any
	IPAddr, DeviceInfo, Details string
}

type UserLogHandler func(r *http.Request) *UserLogData

func (ul *UserLogger) Log(u *model.User, handler UserLogHandler, r *http.Request) error {
	logData := handler(r)

	var userId int

	if u == nil {
		userId = 1
		logData.ActionType = model.AcTypeAnonymous
	} else {
		userId = u.Id
	}

	var lackedField string
	if logData.Action == "" {
		lackedField = "action"
	}

	if len(lackedField) > 0 {
		return model.ActivityValidErr(fmt.Sprintf("%s is required", lackedField))
	}

	// fmt.Println("userId :", userId)
	// fmt.Printf("logger data: %#v\n", logData)

	_, err := ul.Store.Activity.Create(
		userId,
		string(logData.ActionType),
		string(logData.Action),
		string(logData.TargetModel),
		logData.TargetId,
		logData.IPAddr,
		logData.DeviceInfo,
		logData.Details,
	)
	if err != nil {
		return err
	}

	return nil
}
