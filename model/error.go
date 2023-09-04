package model

import "fmt"

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

func NewAppError(err error, code AppErrCode) *AppError {
	return &AppError{err, code}
}
