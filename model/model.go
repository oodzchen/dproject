package model

import (
	i18nc "github.com/oodzchen/dproject/i18n"
)

type OptionItem struct {
	Value any
	Name  string
}

type StringEnum interface {
	Text(upCaseHead bool, i18nCustom *i18nc.I18nCustom) string
}

func ConvertEnumToOPtions(values []StringEnum, upCaseHead bool, enumName string, i18nCustom *i18nc.I18nCustom) []*OptionItem {
	var options []*OptionItem

	for _, val := range values {
		item := &OptionItem{
			Value: val,
			Name:  val.Text(upCaseHead, i18nCustom),
		}

		options = append(options, item)
	}

	return options
}
