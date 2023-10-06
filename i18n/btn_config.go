package i18nc

import "github.com/nicksnyder/go-i18n/v2/i18n"

func AddBtnConfigs() {
	AddLocalizeConfig(&i18n.Message{
		ID:    "BtnClose",
		Other: "Close",
	})

	AddLocalizeConfig(&i18n.Message{
		ID:    "BtnSave",
		Other: "Save",
	})
}
