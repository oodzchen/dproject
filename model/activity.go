package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/xeonx/timeago"
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
	AcActionRegister      AcAction = "register"
	AcActionLogin                  = "login"
	AcActionLogout                 = "logout"
	AcActionUpdateInro             = "update_intro"
	AcActionCreateArticle          = "create_article"
	AcActionEditArticle            = "edit_article"
	AcActionDeleteArticle          = "delete_article"
	AcActionSaveArticle            = "save_article"
	AcActionVoteArticle            = "vote_article"
	AcActionReactArticle           = "react_article"
	AcActionSetRole                = "set_role"
	AcActionAddRole                = "add_role"
	AcActionEditRole               = "edit_role"
)

var AcActionTextMap = map[AcAction]string{
	AcActionRegister:      "registered",
	AcActionLogin:         "login",
	AcActionLogout:        "logout",
	AcActionUpdateInro:    "update introduction",
	AcActionCreateArticle: "created article",
	AcActionEditArticle:   "edited article",
	AcActionDeleteArticle: "deleted article",
	AcActionSaveArticle:   "saved article",
	AcActionVoteArticle:   "voted article",
	AcActionReactArticle:  "reacted to article",
	AcActionSetRole:       "update role of user",
	AcActionAddRole:       "added role",
	AcActionEditRole:      "edited role",
}

func AcActionText(action AcAction) string {
	return AcActionTextMap[action]
}

type AcModel string

const (
	AcModelEmpty   AcModel = ""
	AcModelUser            = "user"
	AcModelArticle         = "article"
	AcModelRole            = "role"
)

type Activity struct {
	Id            int
	UserId        int
	UserName      string
	Type          ActivityType
	Action        string
	TargetId      int
	TargetModel   string
	CreatedAt     *time.Time
	IpAddr        string
	DeviceInfo    string
	Details       string
	FormattedText string
}

func ActivityValidErr(str string) error {
	return errors.Join(ErrValidActivityFailed, errors.New(str))
}

func (act *Activity) Format() {
	text := fmt.Sprintf("<a href=\"/users/%d\">%s</a> %s", act.UserId, act.UserName, AcActionText(AcAction(act.Action)))
	// if AcModel(act.TargetModel) == AcModelArticle {
	// 	text += fmt.Sprintf(" <a href=\"/articles/%d\">/article/%d</a>", act.TargetId, act.TargetId)
	// }

	switch AcModel(act.TargetModel) {
	case AcModelArticle:
		text += fmt.Sprintf(" <a href=\"/articles/%d\">/article/%d</a>", act.TargetId, act.TargetId)
	case AcModelUser:
		text += fmt.Sprintf(" <a href=\"/users/%d\">/users/%d</a>", act.TargetId, act.TargetId)
	default:
	}

	text += fmt.Sprintf(" at <time title=\"%s\">%s</time>", act.CreatedAt.String(), timeago.English.Format(*act.CreatedAt))

	act.FormattedText = text
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
