//go:generate go-enum --names --values -t ./enum_i18n.tmpl

package model

// Language list
/*
   ENUM(
   en, // English
   zh-Hans, // 简体中文
   zh-Hant, // 繁體中文
   jp, // 日本語
   )
*/
type Lang string
