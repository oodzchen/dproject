package i18nc

import "github.com/nicksnyder/go-i18n/v2/i18n"

func (ic *I18nCustom) AddConfigs() {
	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "ReplyNum",
		One:   "{{.Count}} reply",
		Other: "{{.Count}} replies",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "AddNew",
		Other: "New",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Login",
		Other: "Login",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Register",
		Other: "Register",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Logout",
		Other: "Logout",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Settings",
		One:   "Setting",
		Other: "Settings",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Permission",
		One:   "Permission",
		Other: "Permissions",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Role",
		One:   "Role",
		Other: "Roles",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "User",
		One:   "User",
		Other: "Users",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Activity",
		One:   "Activity",
		Other: "Activities",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Best",
		Other: "Best",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Latest",
		Other: "Latest",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Hot",
		Other: "Hot",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "PublishInfo",
		Other: "By {{.Username}} ",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "VoteScore",
		Other: "vote score {{.Score}}",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Discuss",
		Other: "discuss",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Weight",
		Other: "weight {{.Weight}}",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Participate",
		One:   "{{.ParticipateNum}} participate",
		Other: "{{.ParticipateNum}} participates",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "UISaveSuccess",
		Other: "UI settings successfully saved",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Account",
		Other: "Account",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "UI",
		Other: "UI",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Username",
		Other: "Username",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Introduction",
		Other: "Introduction",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Language",
		Other: "Language",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Theme",
		Other: "Theme",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "ThemeLight",
		Other: "Light",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "ThemeDark",
		Other: "Dark",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "ThemeSystem",
		Other: "OS Default",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "ThemeSystemTip",
		Other: "Must enable JavaScript",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "PageLayout",
		Other: "Page Layout",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "PageLayoutFull",
		Other: "Full",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "PageLayoutCentered",
		Other: "Centered",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "AccountSaveSuccess",
		Other: "Account settings successfully saved",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "AccountCreateSuccess",
		Other: "Account created successfully",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Type",
		Other: "Type",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Action",
		Other: "Action",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "All",
		Other: "All",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Manage",
		Other: "Manage",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Re",
		Other: "Re",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Link",
		Other: "Link",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Anchor",
		Other: "Anchor",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Deleted",
		Other: "Deleted",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "ConfirmDelete",
		Other: "Confirm to delete",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "ReactTip",
		Other: "React to content",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "AddContent",
		Other: "Add content",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Content",
		Other: "Content",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Title",
		Other: "Title",
	})
}
