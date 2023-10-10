package i18nc

import (
	"fmt"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/xeonx/timeago"
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
	Configs   map[string]*i18n.LocalizeConfig
	TimeAgo   *timeago.Config
}

func New(files []string) *I18nCustom {
	Bundle := i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	for _, path := range files {
		Bundle.MustLoadMessageFile(path)
	}

	Localizer := i18n.NewLocalizer(Bundle, defaultLang)

	timeago.English.PastPrefix = " "
	timeago.Chinese.PastPrefix = "于 "
	timeago.Chinese.DefaultLayout = "于 2006-01-02"

	custom := &I18nCustom{
		CurrLang:  defaultLang,
		Bundle:    Bundle,
		Localizer: Localizer,
		Configs:   make(map[string]*i18n.LocalizeConfig),
		TimeAgo:   &timeago.English,
	}

	custom.AddConfigs()
	custom.AddBtnConfigs()
	custom.AddErrorConfigs()

	return custom
}

func (ic *I18nCustom) AddLocalizeConfig(message *i18n.Message) {
	if message.ID == "" {
		panic(fmt.Errorf("Message lack of id: %v", message))
	}

	ic.Configs[message.ID] = &i18n.LocalizeConfig{
		DefaultMessage: message,
	}
}

func (ic *I18nCustom) MustLocalize(id string, templateData any, pluralcount any) string {
	config, ok := ic.Configs[id]
	if !ok {
		fmt.Printf("lack of i18n config: %s\n", id)
		return ""
	}

	if templateData != "" && templateData != nil {
		config.TemplateData = templateData
	}

	if pluralcount != "" && pluralcount != nil {
		config.PluralCount = pluralcount
	}

	return ic.Localizer.MustLocalize(config)
}

var timeAgoZHHant = &timeago.Config{
	PastPrefix:   "於 ",
	PastSuffix:   "前",
	FuturePrefix: "於 ",
	FutureSuffix: "",

	Periods: []timeago.FormatPeriod{
		{D: time.Second, One: "1 秒", Many: "%d 秒"},
		{D: time.Minute, One: "1 分鐘", Many: "%d 分鐘"},
		{D: time.Hour, One: "1 小時", Many: "%d 小時"},
		{D: timeago.Day, One: "1 天", Many: "%d 天"},
		{D: timeago.Month, One: "1 月", Many: "%d 月"},
		{D: timeago.Year, One: "1 年", Many: "%d 年"},
	},

	Zero: "1 秒",

	Max:           73 * time.Hour,
	DefaultLayout: "於 2006-01-02",
}

func (ic *I18nCustom) SwitchLang(lang string) {
	// fmt.Println("switch lang: ", lang)
	ic.Localizer = i18n.NewLocalizer(ic.Bundle, lang)
	ic.CurrLang = lang

	switch lang {
	case "zh-Hans":
		ic.TimeAgo = &timeago.Chinese
	case "zh-Hant":
		ic.TimeAgo = timeAgoZHHant
	// case "jp":
	// ic.TimeAgo = &timeago.Japanese
	default:
		ic.TimeAgo = &timeago.English
	}
	// fmt.Println("login str:", MustLocalize("Login", "", 0))
}

func (ic *I18nCustom) LocalTpl(id string, data ...any) string {
	// fmt.Println("data: ", data)
	// fmt.Println("len(data): ", len(data))

	if len(data) == 0 {
		return ic.MustLocalize(id, "", "")
	}

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
