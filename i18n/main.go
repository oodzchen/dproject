package i18nc

import (
	"fmt"

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

func Init(files []string) {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	for _, path := range files {
		Bundle.MustLoadMessageFile(path)
	}

	Localizer = i18n.NewLocalizer(Bundle, "en")

	AddConfigs()
}

func AddLocalizeConfig(message *i18n.Message) {
	if message.ID == "" {
		panic(fmt.Errorf("Message lack of id: %v", message))
	}

	configs[message.ID] = &i18n.LocalizeConfig{
		DefaultMessage: message,
	}
}

func MustLocalize(id string, templateData any, pluralcount any) string {
	config := configs[id]
	config.TemplateData = templateData
	config.PluralCount = pluralcount

	return Localizer.MustLocalize(config)
}

func SwitchLang(lang string) {
	// fmt.Println("switch lang: ", lang)
	Localizer = i18n.NewLocalizer(Bundle, lang)

	// fmt.Println("login str:", MustLocalize("Login", "", 0))
}
