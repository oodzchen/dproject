package i18nc

import (
	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type LocalizeConfig interface {
	ParseData(data any) *i18n.LocalizeConfig
}

var Lang = "en-US"
var Bundle *i18n.Bundle
var Localizer *i18n.Localizer
var configs = make(map[string]*i18n.LocalizeConfig)

func Init() {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	Localizer = i18n.NewLocalizer(Bundle, "en-US")

	AddLocalizeConfig("ReplyNum", &i18n.Message{
		ID:          "ReplyNum",
		Description: "Reply number",
		One:         "{{.Count}} reply",
		Other:       "{{.Count}} replies",
	})
}

func AddLocalizeConfig(id string, message *i18n.Message) {
	configs[id] = &i18n.LocalizeConfig{
		DefaultMessage: message,
	}
}

func MustLocalize(id string, templateData any, pluralcount any) string {
	config := configs[id]
	config.TemplateData = templateData
	config.PluralCount = pluralcount

	return Localizer.MustLocalize(config)
}
