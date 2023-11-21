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
		ID:    "BtnUnsave",
		Other: "Unsave",
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

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnDelete",
		Other: "Delete",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnReply",
		Other: "Reply",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnEdit",
		Other: "Edit",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnParent",
		Other: "Parent",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnFold",
		Other: "Fold",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnMore",
		Other: "More",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnSubscribe",
		Other: "Subscribe",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnUnsubscribe",
		Other: "Unsubscribe",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnNextStep",
		Other: "Next step",
	})

	// ic.AddLocalizeConfig(&i18n.Message{
	// 	ID:    "BtnIgnore",
	// 	Other: "Ignore",
	// })
}
