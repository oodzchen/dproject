//go:generate go-enum --names --values -t enum_int_i18n.tmpl

package model

import (
	"fmt"
)

// App Error
/*
   ENUM(
   AlreadyRegistered = 1000, // already registered
   NotRegistered, // not registered
   UserValidFailed, // user data validation failed
   ArticleValidFailed, // article data validation failed
   PermissionValidFailed, // permission data validation failed
   RoleValidFailed, // role data validation failed
   ActivityValidFailed, // activity data validation failed
   UserNotExist, // user dose not exist
   )
*/
type AppError int

// type AppError struct {
// 	Err     error
// 	ErrCode AppErrCode
// }

func (x AppError) Error() string {
	return fmt.Sprintf("error code: %d, %s", x, x.Text(false, translator))
}

// type AppErrCode int

// const (
// 	ErrAlreadyRegistered AppErrCode = 1000
// 	ErrNotRegistered                = 1001
// )

// var (
// 	ErrValidUserFailed       = errors.New("user validation failed")
// 	ErrValidArticleFailed    = errors.New("article validation failed")
// 	ErrValidPermissionFailed = errors.New("permission validation failed")
// 	ErrValidRoleFailed       = errors.New("role validation failed")
// 	ErrValidActivityFailed   = errors.New("activity validation failed")

// 	ErrUserNotExist = errors.New("user dose not exist")
// )

// func NewAppError(err error, code AppErrCode) *AppError {
// 	return &AppError{err, code}
// }
