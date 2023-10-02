//go:generate go-enum --names --values -t ./enum_text.tmpl

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
