package model

import (
	"errors"
	"fmt"
)

type AppError struct {
	Err     error
	ErrCode AppErrCode
}

func (ae *AppError) Error() string {
	return fmt.Sprintf("error code: %d, %s", ae.ErrCode, ae.Err)
}

type AppErrCode int

const (
	ErrAlreadyRegistered AppErrCode = 1000
	ErrNotRegistered                = 1001
)

var (
	ErrValidUserFailed       = errors.New("user validation failed")
	ErrValidArticleFailed    = errors.New("article validation failed")
	ErrValidPermissionFailed = errors.New("permission validation failed")
	ErrValidRoleFailed       = errors.New("role validation failed")
)

func NewAppError(err error, code AppErrCode) *AppError {
	return &AppError{err, code}
}
