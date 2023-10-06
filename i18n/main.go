package i18nc

import (
	"fmt"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

type LocalizeConfig interface {
	ParseData(data any) *i18n.LocalizeConfig
}

var defaultLang = "en"

// var Bundle *i18n.Bundle
// var Localizer *i18n.Localizer
// var configs = make(map[string]*i18n.LocalizeConfig)

type I18nCustom struct {
	CurrLang  string
	Bundle    *i18n.Bundle
	Localizer *i18n.Localizer
	configs   map[string]*i18n.LocalizeConfig
}

func New(files []string) *I18nCustom {
	Bundle := i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	for _, path := range files {
		Bundle.MustLoadMessageFile(path)
	}

	Localizer := i18n.NewLocalizer(Bundle, defaultLang)

	custom := &I18nCustom{
		CurrLang:  defaultLang,
		Bundle:    Bundle,
		Localizer: Localizer,
		configs:   make(map[string]*i18n.LocalizeConfig),
	}

	custom.AddConfigs()
	custom.AddBtnConfigs()

	return custom
}

func (ic *I18nCustom) AddLocalizeConfig(message *i18n.Message) {
	if message.ID == "" {
		panic(fmt.Errorf("Message lack of id: %v", message))
	}

	ic.configs[message.ID] = &i18n.LocalizeConfig{
		DefaultMessage: message,
	}
}

func (ic *I18nCustom) MustLocalize(id string, templateData any, pluralcount any) string {
	config := ic.configs[id]

	if templateData != "" && templateData != nil {
		config.TemplateData = templateData
	}

	if pluralcount != "" && pluralcount != nil {
		config.PluralCount = pluralcount
	}

	return ic.Localizer.MustLocalize(config)
}

func (ic *I18nCustom) SwitchLang(lang string) {
	// fmt.Println("switch lang: ", lang)
	ic.Localizer = i18n.NewLocalizer(ic.Bundle, lang)
	ic.CurrLang = lang
	// fmt.Println("login str:", MustLocalize("Login", "", 0))
}

func (ic *I18nCustom) LocalTpl(id string, data ...any) string {
	var tplData = make(map[any]any)
	for idx, item := range data {
		if idx%2 == 0 {
			val := data[idx+1]
			if item == "Count" {
				switch v := val.(type) {
				case string:
					tplData[item], _ = strconv.Atoi(v)
				case int32:
					tplData[item] = int(v)
				case int64:
					tplData[item] = int(v)
				case int:
					tplData[item] = v
				case float32:
					tplData[item] = int(v)
				case float64:
					tplData[item] = int(v)
				default:
					// fmt.Println("Count data type: ", reflect.TypeOf(v))
					tplData[item] = 0
				}
			} else {
				tplData[item] = data[idx+1]
			}
		}
	}

	return ic.MustLocalize(id, tplData, tplData["Count"])
}
