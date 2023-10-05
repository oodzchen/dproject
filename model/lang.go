//go:generate go-enum --names --values -t ./enum_text.tmpl

package model

// Language list
/*
   ENUM(
   en, // English
   zh-Hans, // Simplified Chinese
   zh-Hant, // Traditional Chinese
   )
*/
type Lang string
