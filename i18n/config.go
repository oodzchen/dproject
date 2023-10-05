package i18nc

import "github.com/nicksnyder/go-i18n/v2/i18n"

func AddConfigs() {
	AddLocalizeConfig(&i18n.Message{
		ID:    "ReplyNum",
		One:   "{{.Count}} reply",
		Other: "{{.Count}} replies",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "AddNew",
		Other: "New",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Login",
		Other: "Login",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Register",
		Other: "Register",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Logout",
		Other: "Logout",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Settings",
		One:   "Setting",
		Other: "Settings",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Permission",
		One:   "Permission",
		Other: "Permissions",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Role",
		One:   "Role",
		Other: "Roles",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "User",
		One:   "User",
		Other: "Users",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Activity",
		One:   "Activity",
		Other: "Activities",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Best",
		Other: "Best",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Latest",
		Other: "Latest",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Hot",
		Other: "Hot",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "PublishInfo",
		Other: "By {{.Username}} {{.TimeAgo}}",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "VoteScore",
		Other: "vote score {{.VoteScore}}",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Discuss",
		Other: "discuss",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Weight",
		Other: "weight {{.Weight}}",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "Participate",
		One:   "{{.ParticipateNum}} participate",
		Other: "{{.ParticipateNum}} participates",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "BtnClose",
		Other: "Close",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "UISaveSuccess",
		Other: "UI settings successfully saved",
	})

}
