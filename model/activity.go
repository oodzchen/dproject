package model

import (
	"errors"
	"fmt"
	"time"

	i18nc "github.com/oodzchen/dproject/i18n"
)

type Activity struct {
	Id            int
	UserId        int
	UserName      string
	Type          AcType
	Action        string
	TargetId      string
	TargetModel   string
	CreatedAt     *time.Time
	IpAddr        string
	DeviceInfo    string
	Details       string
	FormattedText string
}

func ActivityValidErr(str string) error {
	return errors.Join(AppErrActivityValidFailed, errors.New(str))
}

func (act *Activity) Format(i18nCustom *i18nc.I18nCustom) {
	acAction, _ := ParseAcAction(act.Action)
	text := fmt.Sprintf("<a href=\"/users/%s\">%s</a> %s", act.UserName, act.UserName, acAction.Text(false, i18nCustom))
	// if AcModel(act.TargetModel) == AcModelArticle {
	// 	text += fmt.Sprintf(" <a href=\"/articles/%d\">/article/%d</a>", act.TargetId, act.TargetId)
	// }

	switch AcModel(act.TargetModel) {
	case AcModelArticle:
		text += fmt.Sprintf(" <a href=\"/articles/%s\">/article/%s</a>", act.TargetId, act.TargetId)
	case AcModelUser:
		text += fmt.Sprintf(" <a href=\"/users/%s\">/users/%s</a>", act.TargetId, act.TargetId)
	default:
	}

	text += fmt.Sprintf(" <time title=\"%s\">%s</time>", act.CreatedAt.String(), i18nCustom.TimeAgo.Format(*act.CreatedAt))

	act.FormattedText = text
}
