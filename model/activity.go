package model

import (
	"errors"
	"time"
)

type ActivityType string

const (
	ActivityTypeUser      ActivityType = "user"
	ActivityTypeManage                 = "manage"
	ActivityTypeAnonymous              = "anonymous"
	ActivityTypeDev                    = "dev"
)

type AcAction string

const (
	AcActionRegister   AcAction = "register"
	AcActionLogin               = "login"
	AcActionLogout              = "logout"
	AcActionUpdateInro          = "update_intro"
)

type AcModel string

const (
	AcModelEmpty   AcModel = ""
	AcModelUser            = "user"
	AcModelArticle         = "article"
)

type Activity struct {
	Id         int
	UserId     int
	UserName   string
	Type       ActivityType
	Action     string
	TargetId   int
	Target     any
	CreatedAt  *time.Time
	IpAddr     string
	DeviceInfo string
	Detail     string
}

func ActivityValidErr(str string) error {
	return errors.Join(ErrValidActivityFailed, errors.New(str))
}

// func (act *Activity) Valid() error {
// 	var lackedField string
// 	if act.UserId == 0 {
// 		lackedField = "user id"
// 	}

// 	if act.IpAddr == "" {
// 		lackedField = "IP address"
// 	}

// 	if act.Type == "" {
// 		lackedField = "action type"
// 	}

// 	if act.Action == "" {
// 		lackedField = "action"
// 	}

// 	if len(lackedField) > 0 {
// 		return ActivityValidErr(fmt.Sprintf("%s is required", lackedField))
// 	}

// 	return nil
// }
