package mdw

import (
	"fmt"
	"testing"

	"github.com/oodzchen/dproject/model"
)

func TestParseStrLang(t *testing.T) {
	tests := []struct {
		in   string
		want model.Lang
	}{
		{in: "zh-cmn", want: model.LangZhHans},
		{in: "zh-cmn-Hans", want: model.LangZhHans},
		{in: "zh-cmn-Hant", want: model.LangZhHant},
		{in: "zh-Hans", want: model.LangZhHans},
		{in: "zh-Hans-CN", want: model.LangZhHans},
		{in: "zh-Hans-HK", want: model.LangZhHans},
		{in: "zh-Hans-MO", want: model.LangZhHans},
		{in: "zh-Hans-SG", want: model.LangZhHans},
		{in: "zh-Hans-TW", want: model.LangZhHans},
		{in: "zh-Hant", want: model.LangZhHant},
		{in: "zh-Hant-HK", want: model.LangZhHant},
		{in: "zh-Hant-MO", want: model.LangZhHant},
		{in: "zh-Hant-SG", want: model.LangZhHant},
		{in: "zh-CN", want: model.LangZhHans},
		{in: "zh-SG", want: model.LangZhHans},
		{in: "zh-HK", want: model.LangZhHant},
		{in: "zh-TW", want: model.LangZhHant},
		{in: "zh-MO", want: model.LangZhHant},
		{in: "zh", want: model.LangZhHans},
		{in: "aa", want: model.LangEn},
		{in: "zhx", want: model.LangEn},
		{in: "zha", want: model.LangEn},
		{in: "en", want: model.LangEn},
		{in: "en-GB", want: model.LangEn},
		{in: "en-GB-oed", want: model.LangEn},
		{in: "en-US", want: model.LangEn},
		{in: "en-CA", want: model.LangEn},
		{in: "jp", want: model.LangJp},
		{in: "jp-JP", want: model.LangJp},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("match string: %s", tt.in), func(t *testing.T) {
			got := parseStrLang(tt.in)
			if got != tt.want {
				t.Errorf("want %s, but got %s", tt.want, got)
			}
		})
	}
}
