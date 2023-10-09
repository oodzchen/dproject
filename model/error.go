//go:generate go-enum --names --values -t enum_int_i18n.tmpl -t enum_error.tmpl

package model

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
type AppErrCode int

type AppError struct {
	ErrCode AppErrCode
}

func (x AppError) Error() string {
	return x.ErrCode.Text(false, translator)
}

func NewAppError(code AppErrCode) error {
	return &AppError{
		ErrCode: code,
	}
}
