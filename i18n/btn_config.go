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

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnHide",
		Other: "Hide",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnUnhide",
		Other: "Unhide",
	})

	// ic.AddLocalizeConfig(&i18n.Message{
	// 	ID:    "BtnIgnore",
	// 	Other: "Ignore",
	// })

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnRecover",
		Other: "Recover",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnBlockRegions",
		Other: "Block Regions",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnLock",
		Other: "Lock",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnUnlock",
		Other: "Unlock",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnFadeOut",
		Other: "Fade Out",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnCancelFadeOut",
		Other: "Cancel Fade Out",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnBan",
		Other: "Ban",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnUnban",
		Other: "Unban",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnEditIntro",
		Other: "Edit Introduction",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "BtnConfirm",
		Other: "Confirm",
	})
}
