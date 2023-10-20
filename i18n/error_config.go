package i18nc

import "github.com/nicksnyder/go-i18n/v2/i18n"

func (ic *I18nCustom) AddErrorConfigs() {
	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Required",
		Other: "{{.FieldNames}} is required",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "Incorrect",
		Other: "{{.FieldNames}} is incorrect",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "FormatError",
		Other: "{{.FieldNames}} format error",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "NotRegistered",
		Other: "The {{.FieldNames}} has not been registered",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "AlreadyExists",
		Other: "The {{.FieldNames}} already exists",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "NotExceed",
		Other: "{{.FieldNames}} must not exceed {{.Num}} characters",
	})

	ic.AddLocalizeConfig(&i18n.Message{
		ID:    "PasswordConfirmError",
		Other: "The passwords entered do not match",
	})
}
