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

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnReset",
		Other: "Reset",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnSearch",
		Other: "Search",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnSubmit",
		Other: "Submit",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnPrevPage",
		Other: "Previous page",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnNextPage",
		Other: "Next page",
	})
}
