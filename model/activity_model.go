//go:generate go-enum --names --values -t ./enum_i18n.tmpl

package model

// Activity Model
/*
   ENUM(
   empty, // Empty
   user, // User
   article, // Article
   role, // Role
   )
*/
type AcModel string
