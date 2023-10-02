//go:generate go-enum --names --values -t ./enum_text.tmpl

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
