package i18nc

import "github.com/nicksnyder/go-i18n/v2/i18n"

func (ic *I18nCustom) AddBtnConfigs() {
	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnClose",
		Other: "Close",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnSave",
		Other: "Save",
	})
}