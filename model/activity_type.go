//go:generate go-enum --names --values -t ./enum_i18n.tmpl

package model

// Activity Type
/*
   ENUM(
   user, // User
   manage, // Management
   anonymous, // Anonymous
   dev, // Development
   )
*/
type AcType string
